package tcp

import (
	"bufio"
	"github.com/enoch300/collectd/utils"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

type TCP struct {
	OutSegs     float64 //TCP发包数
	RetransSegs float64 //TCP重传数
	RetranRate  float64 //TCP重传率
	LastTime    int64   //上次采集时间
}

// Detect 采集整机重传率
func (t *TCP) Collect() error {
	f, err := os.Open("/proc/net/snmp")
	if err != nil {
		return err
	}
	defer f.Close()
	reader := bufio.NewReader(f)
	for {
		line, err := reader.ReadString('\n')
		if err == io.EOF {
			return err
		}
		if err != nil {
			return err
		}
		if !strings.Contains(line, "Tcp") || strings.Contains(line, "RtoAlgorithm") {
			continue
		}

		fields := strings.Fields(line)
		outSegs, _ := strconv.ParseFloat(fields[11], 64)
		retransSegs, _ := strconv.ParseFloat(fields[12], 64)

		if t.LastTime == 0 { //第一次采集，没有时间差，只赋值不计算

		} else {
			t.RetranRate = utils.FormatFloat((retransSegs - t.RetransSegs) / (outSegs - t.OutSegs) * 100)
		}

		t.OutSegs = outSegs
		t.RetransSegs = retransSegs
		t.LastTime = time.Now().Unix()
	}
}

func (t *TCP) GetRetranRate() float64 {
	return t.RetranRate
}

func NewTcp() *TCP {
	return &TCP{
		OutSegs:     0,
		RetransSegs: 0,
		RetranRate:  0,
		LastTime:    0,
	}
}
