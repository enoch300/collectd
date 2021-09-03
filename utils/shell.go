/*
* @Author: wangqilong
* @Description:
* @File: shell
* @Date: 2021/9/2 6:50 下午
 */

package utils

import (
	"fmt"
	"os/exec"
	"strconv"
)

func Exec(cmd string) (string, error) {
	command := exec.Command("sh", "-c", cmd)
	bytes, err := command.Output()
	return string(bytes), err
}

func ExecOutput(cmd string) string {
	output, err := Exec(cmd)
	if err != nil {
		return ""
	}
	Trim(&output)
	return output
}

func FormatFloat(f float64) float64 {
	v, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", f), 64)
	return v
}
