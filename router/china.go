package router

import (
	"bookget/app"
	"bookget/site/China/bjdpm"
	"bookget/site/China/luoyang"
	"bookget/site/China/ncpssd"
	"bookget/site/China/npmtw"
	"bookget/site/China/ouroots"
	"bookget/site/China/rbkdocnpmtw"
	"bookget/site/China/sdutcm"
	"bookget/site/China/szlib"
	"bookget/site/China/tianyige"
	"bookget/site/China/usthk"
	"bookget/site/China/wzlib"
	"bookget/site/China/ynutcm"
)

type ChinaNcl struct{}

func (p ChinaNcl) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	for i, s := range sUrl {
		var nlc app.ChinaNlc
		nlc.Init(i+1, s)
	}
	return nil, nil
}

type RbookNcl struct{}

func (p RbookNcl) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	//for i, s := range sUrl {
	//	var nlc app.NclTw
	//	nlc.Init(i+1, s)
	//}
	return nil, nil
}

type RbkdocNpmTw struct{}

func (p RbkdocNpmTw) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	for i, s := range sUrl {
		rbkdocnpmtw.Init(i+1, s)
	}
	return nil, nil
}

type DigitalarchiveNpmTw struct{}

func (p DigitalarchiveNpmTw) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	for i, s := range sUrl {
		npmtw.Init(i+1, s)
	}
	return nil, nil
}

type CuHk struct{}

func (p CuHk) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	//for i, s := range sUrl {
	//	var cuhk app.Cuhk
	//	cuhk.Init(i+1, s)
	//}
	return nil, nil
}

type UstHk struct{}

func (p UstHk) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	for i, s := range sUrl {
		usthk.Init(i+1, s)
	}
	return nil, nil
}

type LuoYang struct{}

func (p LuoYang) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	for i, s := range sUrl {
		luoyang.Init(i+1, s)
	}
	return nil, nil
}

type OyjyWzlib struct{}

func (p OyjyWzlib) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	for i, s := range sUrl {
		wzlib.Init(i+1, s)
	}
	return nil, nil
}

type YunSzlib struct{}

func (p YunSzlib) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	for i, s := range sUrl {
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
		tianyige.Init(i+1, s)
	}
	return nil, nil
}

type GujiSclib struct{}

func (p GujiSclib) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	for i, s := range sUrl {
		var dziCnlib app.DziCnLib
		dziCnlib.Init(i+1, s)
	}
	return nil, nil
}

type GuijiJslib struct{}

func (p GuijiJslib) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	for i, s := range sUrl {
		var dziCnLib app.DziCnLib
		dziCnLib.Init(i+1, s)
	}
	return nil, nil
}

type MinghuajiBjDpm struct{}

func (p MinghuajiBjDpm) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	for i, s := range sUrl {
		bjdpm.Init(i+1, s)
	}
	return nil, nil
}

type OurootsNlc struct{}

func (p OurootsNlc) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	for i, s := range sUrl {
		ouroots.Init(i+1, s)
	}
	return nil, nil
}

type Ncpssd struct{}

func (p Ncpssd) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	for i, s := range sUrl {
		ncpssd.Init(i+1, s)
	}
	return nil, nil
}

type GujiYnutcm struct{}

func (p GujiYnutcm) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	for i, s := range sUrl {
		ynutcm.Init(i+1, s)
	}
	return nil, nil
}

type Sdutcm struct{}

func (p Sdutcm) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	for i, s := range sUrl {
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

type Szmuseum struct{}

func (p Szmuseum) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	for i, s := range sUrl {
		var szlib app.Szmuseum
		szlib.Init(i+1, s)
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
