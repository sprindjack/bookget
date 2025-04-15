package router

import (
	"bookget/app"
)

type ChinaNcl struct{}

func (p ChinaNcl) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	for i, s := range sUrl {
		var nlc app.ChinaNlc
		nlc.Init(i+1, s)
	}
	return nil, nil
}

type CuHk struct{}

func (p CuHk) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	for i, s := range sUrl {
		var cuhk app.Cuhk
		cuhk.Init(i+1, s)
	}
	return nil, nil
}

type UstHk struct{}

func (p UstHk) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	for i, s := range sUrl {
		var usthk app.Usthk
		usthk.Init(i+1, s)
	}
	return nil, nil
}

type LuoYang struct{}

func (p LuoYang) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	for i, s := range sUrl {
		var luoyang app.Luoyang
		luoyang.Init(i+1, s)
	}
	return nil, nil
}

type Wzlib struct{}

func (p Wzlib) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	for i, s := range sUrl {
		var wzlib app.Wzlib
		wzlib.Init(i+1, s)
	}
	return nil, nil
}

type YunSzlib struct{}

func (p YunSzlib) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	for i, s := range sUrl {
		var szlib app.SzLib
		szlib.Init(i+1, s)
	}
	return nil, nil
}

type GzddGzlib struct{}

func (p GzddGzlib) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	for i, s := range sUrl {
		var gzlib app.Gzlib
		gzlib.Init(i+1, s)
	}
	return nil, nil
}

type TianYiGeLib struct{}

func (p TianYiGeLib) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	for i, s := range sUrl {
		var tianyige app.Tianyige
		tianyige.Init(i+1, s)
	}
	return nil, nil
}

type MinghuajiBjDpm struct{}

func (p MinghuajiBjDpm) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	for i, s := range sUrl {
		var dpmbj app.DpmBj
		dpmbj.Init(i+1, s)
	}
	return nil, nil
}

type OurootsNlc struct{}

func (p OurootsNlc) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	for i, s := range sUrl {
		var ouroots app.Ouroots
		ouroots.Init(i+1, s)
	}
	return nil, nil
}

type Ncpssd struct{}

func (p Ncpssd) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	for i, s := range sUrl {
		var ncpssd app.Ncpssd
		ncpssd.Init(i+1, s)
	}
	return nil, nil
}

type Sdutcm struct{}

func (p Sdutcm) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	for i, s := range sUrl {
		var sdutcm app.Sdutcm
		sdutcm.Init(i+1, s)
	}
	return nil, nil
}

type Tjliblswx struct{}

func (p Tjliblswx) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	for i, s := range sUrl {
		var tjlib app.Tjlswx
		tjlib.Init(i+1, s)
	}
	return nil, nil
}

type Yndfz struct{}

func (p Yndfz) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	for i, s := range sUrl {
		var yndfz app.Yndfz
		yndfz.Init(i+1, s)
	}
	return nil, nil
}

type Hkulib struct{}

func (p Hkulib) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	for i, s := range sUrl {
		var hku app.Hkulib
		hku.Init(i+1, s)
	}
	return nil, nil
}

type Huawen struct{}

func (p Huawen) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	for i, s := range sUrl {
		var hw app.Huawen
		hw.Init(i+1, s)
	}
	return nil, nil
}

type Njuedu struct{}

func (p Njuedu) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	for i, s := range sUrl {
		var nju app.Njuedu
		nju.Init(i+1, s)
	}
	return nil, nil
}

type ZhuCheng struct{}

func (p ZhuCheng) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	for i, s := range sUrl {
		var zc app.ZhuCheng
		zc.Init(i+1, s)
	}
	return nil, nil
}

type CafaEdu struct{}

func (p CafaEdu) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	for i, s := range sUrl {
		var cafa app.CafaEdu
		cafa.Init(i+1, s)
	}
	return nil, nil
}

type War1931 struct{}

func (p War1931) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	for i, s := range sUrl {
		var war app.War1931
		war.Init(i+1, s)
	}
	return nil, nil
}
