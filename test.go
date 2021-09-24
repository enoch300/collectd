/*
* @Author: wangqilong
* @Description:
* @File: main
* @Date: 2021/9/24 2:43 下午
 */

package main

import (
	"collectd/net"
	"fmt"
	"time"
)

func main() {
	Net := net.NewNetwork([]string{}, []string{"docker", "lo"}, []string{}, []string{})
	for {
		err := Net.Collect()
		if err != nil {
			fmt.Printf("Net.Collect: %v\n", err.Error())
			time.Sleep(60 * time.Second)
			continue
		}

		fmt.Printf("Monitor interface: %v\n", Net.IfiNames)
		for iFace, _ := range Net.IfiMap {
			fmt.Printf("Iface details >>> iface: %v\n", iFace)
		}
		time.Sleep(5 * time.Second)
	}
}
