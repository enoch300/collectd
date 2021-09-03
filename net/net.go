package collectd

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/enoch300/collectd/utils"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

// IsInEth 判断是否为内网
func (n *Ifi) IsInEth(filterIps []string) bool {
	if n.Ip == "" {
		return false
	}

	for _, ipPrefix := range filterIps {
		utils.Trim(&ipPrefix)
		if strings.HasPrefix(n.Ip, ipPrefix) {
			return true
		}
	}
	return false
}

type Ifi struct {
	Name              string  //网卡接口
	Ip                string  //网卡IP
	Speed             float64 //网卡速率
	OutRecvPkgErrRate float64 //外网收包错误率
	OutSendPkgErrRate float64 //外网发包错误率
	RecvByte          uint64  //接收的字节数
	RecvPkg           uint64  //接收正确的包数
	RecvErr           uint64  //接收错误的包数
	RecvDrop          uint64  //接收错误的包数
	SendByte          uint64  //发送的字节数
	SendPkg           uint64  //发送正确的包数
	SendErr           uint64  //发送错误的包数
	SendDrop          uint64  //发送错误的包数

	RecvByteAvg  float64 //一个周期平均每秒接收字节数
	RecvPkgAvg   float64 //一个周期平均每秒收包数
	RecvErrRate  float64 //一个周期收包错误率
	RecvDropRate float64 //一个周期收包丢包率

	SendByteAvg  float64 //一个周期平均每秒发送字节数
	SendPkgAvg   float64 //一个周期平均每秒发包数
	SendErrRate  float64 //一个周期发包错误率
	SendDropRate float64 //一个周期发包丢包率

	BandwidthLimit int   //0不限制, 1被限制
	Last           int64 //上次采集时间
}

type NetWork struct {
	IfiMap    map[string]*Ifi
	IfiNames  []string
	FilterIps []string //过滤不监控IP

	//内网
	InRecvByteAvg float64 //所有内网网络接口平均每秒接收字节数之和
	InSendByteAvg float64 //所有内网网络接口平均每秒发送字节数之和
	InRecvPkgAvg  float64 //所有内网网络接口平均每秒收包数之和
	InSendPkgAvg  float64 //所有内网网络接口平均每秒发包数之和

	//外网
	OutRecvDropPkgAvg float64 //所有外网网络接口接收丢包数之和
	OutSendDropPkgAvg float64 //所有外网网络接口发送丢包数之和

	OutRecvErrPkgAvg float64 //所有外网网络接口接收错误数之和
	OutSendErrPkgAvg float64 //所有外网网络接口发送错误数之和

	OutRecvPkgAvg float64 //所有外网网络接口平均每秒收包数之和
	OutSendPkgAvg float64 //所有外网网络接口平均每秒发包数之和

	OutRecvByteAvg float64 //所有外网网络接口平均每秒接收字节数
	OutSendByteAvg float64 //所有外网网络接口平均每秒发送字节数

	RetransSegs float64 //所有外网网络接口TCP重传数
	OutSegs     float64 //所有外网网络接口TCP发包总数
	RetranRate  float64 //所有外网网络接口TCP重传率, RetransSegs/OutSegs

	EthInMaxUseRate  float64 //内网网卡使用率
	EthOutMaxUseRate float64 //外网网卡使用率

	RecvSendDetail string //收发接口收发字节数详细信息
	ModelDetail    string //网络接口型号带宽详细信息

	/*
		//外网网卡流入环比
		OutRecvByteSum10Sum   float64 //外网网卡平均每秒接收字节累加和
		OutRecvByteSum10Times int     //外网网卡平均每秒接收字节累加次数
		OutRecvByteSum10      float64 //外网网卡流入10分钟环比
		OutRecvByteSum10Last  int64

		OutRecvByteSum60Sum   float64 //外网网卡平均每秒接收字节累加和
		OutRecvByteSum60Times int     //外网网卡平均每秒接收字节累加次数
		OutRecvByteSum60      float64 //外网网卡流入60分钟环比
		OutRecvByteSum60Last  int64

		OutRecvByteSumDaySum   float64 //外网网卡平均每秒接收字节累加和
		OutRecvByteSumDayTimes int     //外网网卡平均每秒接收字节累加次数
		OutRecvByteSumDay      float64 //外网网卡流入日同比
		OutRecvByteSumDayLast  int64
	*/
}

func (n *NetWork) InitNetwork() {
	n.IfiMap = make(map[string]*Ifi)
	n.FilterIps = []string{"127.", "0.", "10.", "192.", "172."}

	n.InSendByteAvg = 0
	n.InRecvByteAvg = 0
	n.InRecvPkgAvg = 0
	n.InSendPkgAvg = 0

	n.OutRecvPkgAvg = 0
	n.OutSendPkgAvg = 0

	n.OutRecvErrPkgAvg = 0
	n.OutSendErrPkgAvg = 0

	n.OutRecvDropPkgAvg = 0
	n.OutSendDropPkgAvg = 0

	n.OutRecvByteAvg = 0
	n.OutSendByteAvg = 0

	n.OutSegs = 0
	n.RetransSegs = 0
	n.RetranRate = 0

	n.EthInMaxUseRate = 0
	n.EthOutMaxUseRate = 0

	n.RecvSendDetail = ""
	n.ModelDetail = ""
}

func (n *NetWork) Collect() error {
	f, err := os.Open("/proc/net/dev")
	if err != nil {
		return err
	}
	defer f.Close()
	reader := bufio.NewReader(f)

	//内网
	n.InRecvByteAvg = 0
	n.InSendByteAvg = 0

	n.InRecvPkgAvg = 0
	n.InSendPkgAvg = 0

	//外网
	n.OutRecvPkgAvg = 0
	n.OutSendPkgAvg = 0

	n.OutRecvByteAvg = 0
	n.OutSendByteAvg = 0

	n.OutSendErrPkgAvg = 0
	n.OutSendDropPkgAvg = 0

	n.OutRecvErrPkgAvg = 0
	n.OutRecvDropPkgAvg = 0

	for {
		line, err := reader.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if !strings.Contains(line, ":") {
			continue
		}

		fields := strings.Split(line, ":")
		if len(fields) < 2 {
			continue
		}

		ethname := fields[0]
		utils.Trim(&ethname)

		//过滤掉docker网卡
		if strings.HasPrefix(ethname, "docker") {
			continue
		}

		fields = strings.Fields(fields[1])
		if len(fields) != 16 {
			continue
		}

		recvByte, _ := strconv.ParseUint(fields[0], 10, 64)
		recvPkg, _ := strconv.ParseUint(fields[1], 10, 64)
		recvErr, _ := strconv.ParseUint(fields[2], 10, 64)
		recvDrop, _ := strconv.ParseUint(fields[3], 10, 64)

		sendByte, _ := strconv.ParseUint(fields[8], 10, 64)
		sendPkg, _ := strconv.ParseUint(fields[9], 10, 64)
		sendErr, _ := strconv.ParseUint(fields[10], 10, 64)
		sendDrop, _ := strconv.ParseUint(fields[11], 10, 64)

		//根据网卡名得到对应的网络接口
		netifi, err := net.InterfaceByName(ethname)
		if err != nil {
			continue
		}

		var addrs []net.Addr
		addrs, err = netifi.Addrs()
		if err != nil {
			continue
		}

		if len(addrs) == 0 {
			continue
		}

		moniTag := true
		for _, addr := range addrs {
			cidr := addr.String()
			//过滤IPV6
			if strings.Contains(cidr, ":") {
				if len(addrs) == 1 {
					moniTag = false
					break
				} else {
					continue
				}
			}

			for _, ipPrefix := range n.FilterIps {
				utils.Trim(&ipPrefix)
				if strings.HasPrefix(cidr, ipPrefix) {
					moniTag = false
					break
				}
			}
		}

		if moniTag == false {
			continue
		}

		_, exists := n.IfiMap[ethname]
		if !exists {
			n.IfiMap[ethname] = &Ifi{}
			n.IfiNames = append(n.IfiNames, ethname)
		}
		ifi, _ := n.IfiMap[ethname]

		var (
			recvByteAvg    float64
			recvPkgAvg     float64
			recvErrRate    float64
			recvDropRate   float64
			recvErrPkgAvg  float64
			recvDropPkgAvg float64

			sendByteAvg    float64
			sendPkgAvg     float64
			sendErrRate    float64
			sendDropRate   float64
			sendErrPkgAvg  float64
			sendDropPkgAvg float64
		)
		now := time.Now().Unix()
		difftime := float64(now - ifi.Last)
		if ifi.Last == 0 {
			//第一次采集，没有时间差，不计算
		} else {
			if difftime > 0 {
				recvByteAvg = float64(recvByte-ifi.RecvByte) / difftime    //平均每秒接收字节数
				recvPkgAvg = float64(recvPkg-ifi.RecvPkg) / difftime       //平均每秒接收包数
				recvErrPkgAvg = float64(recvErr-ifi.RecvErr) / difftime    //平均每秒接收错误数
				recvDropPkgAvg = float64(recvDrop-ifi.RecvDrop) / difftime //平均每秒接收丢包数
				if recvPkg-ifi.RecvPkg > 0 {
					recvErrRate = float64(recvErr-ifi.RecvErr) / float64(recvPkg-ifi.RecvPkg)    //一个周期收包错误率
					recvDropRate = float64(recvDrop-ifi.RecvDrop) / float64(recvPkg-ifi.RecvPkg) //一个周期收包丢包率
				}

				sendByteAvg = float64(sendByte-ifi.SendByte) / difftime    //平均每秒发送字节数
				sendPkgAvg = float64(sendPkg-ifi.SendPkg) / difftime       //平均每秒发送包数
				sendErrPkgAvg = float64(sendErr-ifi.SendErr) / difftime    //平均每秒发送错误包数
				sendDropPkgAvg = float64(sendDrop-ifi.SendDrop) / difftime //平均每秒发送丢包包数
				if sendPkg-ifi.SendPkg > 0 {
					sendErrRate = float64(sendErr-ifi.SendErr) / float64(sendPkg-ifi.SendPkg)    //一个周期发包错误率
					sendDropRate = float64(sendDrop-ifi.SendDrop) / float64(sendPkg-ifi.SendPkg) //一个周期发包丢包率
				}
			}
		}

		ifi.Name = ethname
		ifi.Ip = strings.Split(addrs[0].String(), "/")[0]

		ifi.RecvByte = recvByte
		ifi.RecvPkg = recvPkg
		ifi.RecvErr = recvErr
		ifi.RecvDrop = recvDrop

		ifi.RecvPkgAvg = recvPkgAvg
		ifi.RecvByteAvg = recvByteAvg
		ifi.RecvErrRate = recvErrRate
		ifi.RecvDropRate = recvDropRate

		ifi.SendByte = sendByte
		ifi.SendPkg = sendPkg
		ifi.SendErr = sendErr
		ifi.SendDrop = sendDrop

		ifi.SendPkgAvg = sendPkgAvg
		ifi.SendByteAvg = sendByteAvg
		ifi.SendErrRate = sendErrRate
		ifi.SendDropRate = sendDropRate

		ifi.Last = now

		if ifi.IsInEth(n.FilterIps) {
			//内网
			n.InRecvByteAvg += recvByteAvg
			n.InSendByteAvg += sendByteAvg

			n.InRecvPkgAvg += recvPkgAvg
			n.InSendPkgAvg += sendPkgAvg
		} else {
			//外网
			n.OutRecvPkgAvg += recvPkgAvg
			n.OutSendPkgAvg += sendPkgAvg

			n.OutRecvByteAvg += recvByteAvg
			n.OutSendByteAvg += sendByteAvg

			n.OutSendErrPkgAvg += sendErrPkgAvg
			n.OutSendDropPkgAvg += sendDropPkgAvg

			n.OutRecvErrPkgAvg += recvErrPkgAvg
			n.OutRecvDropPkgAvg += recvDropPkgAvg
		}

		n.RecvSendDetail += ifi.Ip + "=" + ifi.Name + "=(" + strconv.FormatFloat(recvByteAvg, 'f', 0, 64) + "|" +
			strconv.FormatFloat(sendByteAvg, 'f', 0, 64) + ")$"

		cmd := fmt.Sprintf("/sbin/ethtool %s 2>/dev/null", ethname)
		output, err := utils.Exec(cmd)
		if err != nil {
			continue
		}
		lines := strings.Split(output, "\n")
		for _, line = range lines {
			if strings.Contains(line, "Speed") {
				fields = strings.Split(line, ":")
				if len(fields) != 2 {
					continue
				}
				field2 := fields[1]
				utils.Trim(&field2)
				field2 = strings.Replace(field2, "Mb/s", "", -1)
				speed, err := strconv.ParseFloat(field2, 64) //Mb/s, 注意是小b
				if err != nil {
					continue
				}
				ifi.Speed = speed
				if speed > 0 {
					inEthUseRate := recvByteAvg * 8 * 100 / (speed * 1024 * 1024)
					if inEthUseRate > n.EthInMaxUseRate {
						n.EthInMaxUseRate = inEthUseRate
					}
					outEthUseRate := sendByteAvg * 8 * 100 / (speed * 1024 * 1024)
					if outEthUseRate > n.EthOutMaxUseRate {
						n.EthOutMaxUseRate = outEthUseRate
					}
				}
				break
			}
		}
		n.ModelDetail += fmt.Sprintf("%v|%v|%v$", ifi.Name, ifi.Ip, ifi.Speed)
	}
	return nil
}

// OutSendErrAvg 所有外网发包错误率
func (n *NetWork) OutSendErrAvg() float64 {
	return utils.FormatFloat(n.OutSendErrPkgAvg)
}

// OutSendDropAvg 所有外网发包丢包率
func (n *NetWork) OutSendDropAvg() float64 {
	return utils.FormatFloat(n.OutSendDropPkgAvg)
}

// OutRecvErrAvg 所有外网收包错误率
func (n *NetWork) OutRecvErrAvg() float64 {
	return utils.FormatFloat(n.OutRecvErrPkgAvg)
}

// OutRecvDropAvg 所有外网收包丢包率
func (n *NetWork) OutRecvDropAvg() float64 {
	return utils.FormatFloat(n.OutRecvDropPkgAvg)
}

//  +++++ 整机指标 +++++

// InSendPkgSumFunc 所有内网平均发包速率(pkg/s)
func (n *NetWork) InSendPkgSumFunc() float64 {
	return utils.FormatFloat(n.InSendPkgAvg)
}

// InRecvPkgSumFunc 所有内网平均收包速率(pkg/s)
func (n *NetWork) InRecvPkgSumFunc() float64 {
	return utils.FormatFloat(n.InRecvPkgAvg)
}

// InEthRecvByteAvgFunc 所有内网平均网入带宽(byte/s)
func (n *NetWork) InEthRecvByteAvgFunc() float64 {
	return utils.FormatFloat(n.InRecvByteAvg)
}

// InEthSendByteAvgFunc 所有内网平均网出带宽(byte/s)
func (n *NetWork) InEthSendByteAvgFunc() float64 {
	return utils.FormatFloat(n.InSendByteAvg)
}

// OutSendPkgAvgFunc 所有外网平均发包速度(pkg/s)
func (n *NetWork) OutSendPkgAvgFunc() float64 {
	return utils.FormatFloat(n.OutSendPkgAvg)
}

// OutRecvPkgAvgFunc 所有外网平均收包速度(pkg/s)
func (n *NetWork) OutRecvPkgAvgFunc() float64 {
	return utils.FormatFloat(n.OutRecvPkgAvg)
}

// OutEthRecvByteAvgFunc 所有外网平均入带宽(byte/s)
func (n *NetWork) OutEthRecvByteAvgFunc() float64 {
	return utils.FormatFloat(n.OutRecvByteAvg)
}

// OutEthSendByteAvgFunc 所有外网平均出带宽(byte/s)
func (n *NetWork) OutEthSendByteAvgFunc() float64 {
	return utils.FormatFloat(n.OutSendByteAvg)
}

// OutRecvErrPkgRateFun 所有外网网卡接收错误率
func (n *NetWork) OutRecvErrPkgRateFun() float64 {
	if n.OutRecvErrPkgAvg == 0 && n.OutRecvPkgAvg == 0 {
		return 0
	}
	return utils.FormatFloat(n.OutRecvErrPkgAvg / n.OutRecvPkgAvg * 100)
}

// OutRecvDropPkgRateFun 所有外网网卡接收丢包率
func (n *NetWork) OutRecvDropPkgRateFun() float64 {
	if n.OutRecvDropPkgAvg == 0 && n.OutRecvPkgAvg == 0 {
		return 0
	}
	return utils.FormatFloat(n.OutRecvDropPkgAvg / n.OutRecvPkgAvg * 100)
}

// OutSendErrPkgRateFun 所有外网网卡发送错误率
func (n *NetWork) OutSendErrPkgRateFun() float64 {
	if n.OutSendErrPkgAvg == 0 && n.OutSendPkgAvg == 0 {
		return 0
	}
	return utils.FormatFloat(n.OutSendErrPkgAvg / n.OutSendPkgAvg * 100)
}

// OutSendDropPkgRateFun 所有外网网卡发送丢包率
func (n *NetWork) OutSendDropPkgRateFun() float64 {
	if n.OutSendDropPkgAvg == 0 && n.OutSendPkgAvg == 0 {
		return 0
	}
	return utils.FormatFloat(n.OutSendDropPkgAvg / n.OutSendPkgAvg * 100)
}

// EthInMaxUseRateFunc 所有网卡入带宽最大使用率
func (n *NetWork) EthInMaxUseRateFunc() float64 {
	return utils.FormatFloat(n.EthInMaxUseRate)
}

// EthOutMaxUseRateFunc 所有网卡出带宽最大使用率
func (n *NetWork) EthOutMaxUseRateFunc() float64 {
	return utils.FormatFloat(n.EthOutMaxUseRate)
}

// +++++ 单网卡 +++++

// EthSendPkgAvgFunc 网卡平均发包速度(pkg/s)
func (n *NetWork) EthSendPkgAvgFunc(args string) float64 {
	ifi, err := n.GetIfiByIndex(args)
	if err != nil {
		return 0
	}
	return utils.FormatFloat(ifi.SendPkgAvg)
}

// EthRecvPkgAvgFunc 网卡平均收包速率(pkg/s)
func (n *NetWork) EthRecvPkgAvgFunc(args string) float64 {
	ifi, err := n.GetIfiByIndex(args)
	if err != nil {
		return 0
	}
	return utils.FormatFloat(ifi.RecvPkgAvg)
}

// EthRecvByteAvgFunc 网卡平均接收字节速率(byte/s)
func (n *NetWork) EthRecvByteAvgFunc(args string) float64 {
	ifi, err := n.GetIfiByIndex(args)
	if err != nil {
		return 0
	}
	return utils.FormatFloat(ifi.RecvByteAvg)
}

// EthSendByteAvgFunc 网卡平均发送字节速率(byte/s)
func (n *NetWork) EthSendByteAvgFunc(args string) float64 {
	ifi, err := n.GetIfiByIndex(args)
	if err != nil {
		return 0
	}
	return utils.FormatFloat(ifi.SendByteAvg)
}

func (n *NetWork) GetIfiBandwidthLimitStatusByIp(ip string) (isLimit int) {
	for _, ethInfo := range n.IfiMap {
		if ethInfo.Ip == ip {
			return ethInfo.BandwidthLimit
		}
	}
	return isLimit
}

func (n *NetWork) GetIfiByIndex(args string) (*Ifi, error) {
	index, err := strconv.Atoi(args)
	if err != nil {
		return nil, err
	}
	if index < 0 {
		return nil, errors.New("invalid index")
	}
	length := len(n.IfiNames)
	if index > length-1 {
		return nil, errors.New("invalid index")
	}
	key := n.IfiNames[index]
	ifi, exists := n.IfiMap[key]
	if exists {
		return ifi, nil
	}
	return nil, errors.New("key not found")
}

// EthRecvErrRateFunc 网卡收包错误率
func (n *NetWork) EthRecvErrRateFunc(args string) float64 {
	ifi, err := n.GetIfiByIndex(args)
	if err != nil {
		return 0
	}
	return utils.FormatFloat(ifi.RecvErrRate)
}

// EthRecvDropRateFunc 网卡收包丢包率
func (n *NetWork) EthRecvDropRateFunc(args string) float64 {
	ifi, err := n.GetIfiByIndex(args)
	if err != nil {
		return 0
	}
	return utils.FormatFloat(ifi.RecvDropRate)
}

// EthSendErrRateFunc 网卡发包错误率
func (n *NetWork) EthSendErrRateFunc(args string) float64 {
	ifi, err := n.GetIfiByIndex(args)
	if err != nil {
		return 0
	}
	return utils.FormatFloat(ifi.SendErrRate)
}

// EthSendDropRateFunc 网卡发包丢包率
func (n *NetWork) EthSendDropRateFunc(args string) float64 {
	ifi, err := n.GetIfiByIndex(args)
	if err != nil {
		return 0
	}
	return utils.FormatFloat(ifi.SendDropRate)
}

// EthSpeedFunc 网卡速率(Mb/s)
func (n *NetWork) EthSpeedFunc(args string) float64 {
	ifi, err := n.GetIfiByIndex(args)
	if err != nil {
		return 0
	}
	return utils.FormatFloat(ifi.Speed)
}

//EthModelFunc 机器网卡信息
func (n *NetWork) EthModelFunc(args string) string {
	return n.ModelDetail
}

// EthByteSetFunc 所有网卡流量信息
func (n *NetWork) EthByteSetFunc(args string) string {
	return n.RecvSendDetail
}

/*
func (n *NetWork) AddRecvBytes(bytes float64) {
	n.OutRecvByteSum10Sum += bytes
	n.OutRecvByteSum10Times++
	n.OutRecvByteSum60Sum += bytes
	n.OutRecvByteSum60Times++
	n.OutRecvByteSumDaySum += bytes
	n.OutRecvByteSumDayTimes++
}*/

/*
func (n *NetWork) ResetRecvSum10() {
	n.OutRecvByteSum10Sum = 0
	n.OutRecvByteSum10Times = 0
}

func (n *NetWork) ResetRecvSum60() {
	n.OutRecvByteSum60Sum = 0
	n.OutRecvByteSum60Times = 0
}

func (n *NetWork) ResetRecvSumDay() {
	n.OutRecvByteSumDaySum = 0
	n.OutRecvByteSumDayTimes = 0
}*/

/*
//外网网卡流入，10分钟环比
func (n *NetWork) OutEthRecv10Func(args string) float64 {
	if time.Now().Unix()-n.OutRecvByteSum10Last < 600 {
		return 0
	}
	if n.OutRecvByteSum10Times <= 0 {
		return 0
	}
	var ret float64 = 0
	avg := n.OutRecvByteSum10Sum / float64(n.OutRecvByteSum10Times)
	//到这
	if n.OutRecvByteSum10 == 0 && avg != 0 {
		ret = 100
	} else {
		ret = (avg - n.OutRecvByteSum10) / n.OutRecvByteSum10 * 100
	}
	n.OutRecvByteSum10 = avg
	n.OutRecvByteSum10Last = time.Now().Unix()
	n.ResetRecvSum10()
	return g.ParseFloat(ret)
}

//外网网卡流入，60分钟环比
func (n *NetWork) OutEthRecv60Func(args string) float64 {
	if time.Now().Unix()-n.OutRecvByteSum60Last < 3600 {
		return 0
	}
	if n.OutRecvByteSum60Times <= 0 {
		return 0
	}
	var ret float64 = 0
	avg := n.OutRecvByteSum60Sum / float64(n.OutRecvByteSum60Times)
	//到这
	if n.OutRecvByteSum60 == 0 && avg != 0 {
		ret = 100
	} else {
		ret = (avg - n.OutRecvByteSum60) / n.OutRecvByteSum60 * 100
	}
	n.OutRecvByteSum60 = avg
	n.OutRecvByteSum60Last = time.Now().Unix()
	n.ResetRecvSum60()
	return g.ParseFloat(ret)
}

//外网网卡流入，日环比
func (n *NetWork) OutEthRecvDayFunc(args string) float64 {
	if time.Now().Unix()-n.OutRecvByteSumDayLast < 3600 {
		return 0
	}
	if n.OutRecvByteSumDayTimes <= 0 {
		return 0
	}
	var ret float64 = 0
	avg := n.OutRecvByteSumDaySum / float64(n.OutRecvByteSumDayTimes)
	//到这
	if n.OutRecvByteSumDay == 0 && avg != 0 {
		ret = 100
	} else {
		ret = (avg - n.OutRecvByteSumDay) / n.OutRecvByteSumDay * 100
	}
	n.OutRecvByteSumDay = avg
	n.OutRecvByteSumDayLast = time.Now().Unix()
	n.ResetRecvSumDay()
	return g.ParseFloat(ret)
}
*/

func ConnNumByPort(port string) string {
	return utils.ExecOutput("netstat -pnt |grep ':" + port + "\\b' |wc -l")
}
