package config

import (
	"os"
	"strconv"
	"strings"
)

var Conf Input

const version = "24.0111"

// initSeq    false = 最小值 <= 当前页码 <=  最大值
func initSeq() {
	if Conf.Seq == "" || !strings.Contains(Conf.Seq, ":") {
		return
	}
	m := strings.Split(Conf.Seq, ":")
	min, _ := strconv.Atoi(m[0])
	max, _ := strconv.Atoi(m[1])
	Conf.SeqStart = min
	Conf.SeqEnd = max
	return
}

func UserHomeDir() string {
	if os.PathSeparator == '\\' {
		home := os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
		return home
	}
	return os.Getenv("HOME")
}
