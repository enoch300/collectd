/*
* @Author: wangqilong
* @Description:
* @File: string
* @Date: 2021/9/2 6:50 下午
 */

package utils

import "strings"

func Trim(str *string) {
	*str = strings.TrimSpace(*str)
	*str = strings.Replace(*str, "\n", "", -1)
}
