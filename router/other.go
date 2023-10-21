package router

import (
	"bookget/app"
	"bookget/site/China/idp"
)

type NormalIIIF struct{}

func (p NormalIIIF) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	for i, s := range sUrl {
		var iiif app.IIIF
		iiif.AutoDetectManifest(i+1, s)
	}
	return nil, nil
}

type NormalHttp struct{}

func (p NormalHttp) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	var wget app.Wget
	wget.InitMultiple(sUrl)
	return nil, nil
}

type IDP struct{}

func (p IDP) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	for i, s := range sUrl {
		idp.Init(i+1, s)
	}
	return nil, nil
}

type KyudbSnu struct{}

func (p KyudbSnu) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	for i, s := range sUrl {
		n := i + 1
		var kyudbsnu app.KyudbSnu
		kyudbsnu.Init(n, s)
	}
	return nil, nil
}

type Sillokgokr struct{}

func (p Sillokgokr) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	for i, s := range sUrl {
		n := i + 1
		var sillok app.SillokGoKr
		sillok.Init(n, s)
	}
	return nil, nil
}

type DlibGoKr struct{}

func (p DlibGoKr) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	for i, s := range sUrl {
		n := i + 1
		var libgokr app.DlibGoKr
		libgokr.Init(n, s)
	}
	return nil, nil
}

type RslRu struct{}

func (p RslRu) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	for i, s := range sUrl {
		n := i + 1
		var rslru app.RslRu
		rslru.Init(n, s)
	}
	return nil, nil
}

type Nomfoundation struct{}

func (p Nomfoundation) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	for i, s := range sUrl {
		n := i + 1
		var nomfoundation app.Nomfoundation
		nomfoundation.Init(n, s)
	}
	return nil, nil
}
