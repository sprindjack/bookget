package router

import (
	"bookget/app"
	"bookget/site/Japan/emuseum"
	"bookget/site/Japan/gprime"
	"bookget/site/Japan/kokusho"
	"bookget/site/Japan/kyoto"
	"bookget/site/Japan/national"
	"bookget/site/Japan/niiac"
	"bookget/site/Japan/utokyo"
	"bookget/site/Japan/yonezawa"
)

type RmdaKyoto struct{}

func (p RmdaKyoto) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	for i, s := range sUrl {
		kyoto.Init(i+1, s)
	}
	return nil, nil
}

type NdlGo struct{}

func (p NdlGo) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	for i, s := range sUrl {
		var ndl app.NdlJP
		ndl.Init(i+1, s)
	}
	return nil, nil
}

type EmuseumNich struct{}

func (p EmuseumNich) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	for i, s := range sUrl {
		emuseum.Init(i+1, s)
	}
	return nil, nil
}

type SidoKeio struct{}

func (p SidoKeio) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	for i, s := range sUrl {
		var keio app.Keio
		keio.Init(i+1, s)
	}
	return nil, nil
}

type ShanbenuTokyo struct{}

func (p ShanbenuTokyo) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	for i, s := range sUrl {
		utokyo.Init(i+1, s)
	}
	return nil, nil
}

type ArchivesGo struct{}

func (p ArchivesGo) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	for i, s := range sUrl {
		national.Init(i+1, s)
	}
	return nil, nil
}

type DsrNiiAc struct{}

func (p DsrNiiAc) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	for i, s := range sUrl {
		niiac.Init(i+1, s)
	}
	return nil, nil
}

type KokushoNijlAc struct{}

func (p KokushoNijlAc) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	for i, s := range sUrl {
		kokusho.Init(i+1, s)
	}
	return nil, nil
}

type Kyotou struct{}

func (p Kyotou) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	for i, s := range sUrl {
		var kyotou app.Kyotou
		kyotou.Init(i+1, s)
	}
	return nil, nil
}

type ElibGprime struct{}

func (p ElibGprime) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	for i, s := range sUrl {
		gprime.Init(i+1, s)
	}
	return nil, nil
}

type KhirinRekihaku struct{}

func (p KhirinRekihaku) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	taskUrls := make([]string, 0, 10)
	for _, s := range sUrl {
		dUrl := ExplanRegexpUrl(s)
		taskUrls = append(taskUrls, dUrl...)
	}
	for i, s := range taskUrls {
		var khirin = app.Khirin{}
		khirin.Init(i+1, s)
	}
	return nil, nil
}

type LibYonezawa struct{}

func (p LibYonezawa) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	for i, s := range sUrl {
		yonezawa.Init(i+1, s)
	}
	return nil, nil
}

type WebarchivesTnm struct{}

func (p WebarchivesTnm) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	for i, s := range sUrl {
		var tnm app.Tnm
		tnm.Init(i+1, s)
	}
	return nil, nil
}

type Waseda struct{}

func (p Waseda) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	for i, s := range sUrl {
		var waseda app.Waseda
		waseda.Init(i+1, s)
	}
	return nil, nil
}

type Ryukoku struct{}

func (p Ryukoku) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	for i, s := range sUrl {
		var ryukoku app.Ryukoku
		ryukoku.Init(i+1, s)
	}
	return nil, nil
}
