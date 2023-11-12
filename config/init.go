package config

import (
	"strconv"
	"strings"
)

var Conf Input

const version = "231112"

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
