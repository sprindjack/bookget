package router

import (
	"bookget/app"
)

type OxacUk struct{}

func (p OxacUk) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	for i, s := range sUrl {
		var oxacuk app.Oxacuk
		oxacuk.Init(i+1, s)
	}
	return nil, nil
}

type DigitalBerlin struct{}

func (p DigitalBerlin) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	for i, s := range sUrl {
		var berlin app.Berlin
		berlin.Init(i+1, s)
	}
	return nil, nil
}

type BlUk struct{}

func (p BlUk) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	for i, s := range sUrl {
		var bluk app.Bluk
		bluk.Init(i+1, s)
	}
	return nil, nil
}

type Sammlungen struct{}

func (p Sammlungen) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	for i, s := range sUrl {
		var sammlungen = app.Sammlungen{}
		sammlungen.Init(i+1, s)
	}
	return nil, nil
}

type SearchworksStanford struct{}

func (p SearchworksStanford) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	for i, s := range sUrl {
		var stanford app.Stanford
		stanford.Init(i+1, s)
	}
	return nil, nil
}

type FamilySearch struct{}

func (p FamilySearch) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	for i, s := range sUrl {
		var familysearch app.Familysearch
		familysearch.Init(i+1, s)
	}
	return nil, nil
}

type SiEdu struct{}

func (p SiEdu) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	for i, s := range sUrl {
		var siedu app.SiEdu
		siedu.Init(i+1, s)
	}
	return nil, nil
}

type Berkeley struct{}

func (p Berkeley) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	for i, s := range sUrl {
		var berkeley app.Berkeley
		berkeley.Init(i+1, s)
	}
	return nil, nil
}

type OnbDigital struct{}

func (p OnbDigital) getRouterInit(sUrl []string) (map[string]interface{}, error) {
	for i, s := range sUrl {
		var onb app.OnbDigital
		onb.Init(i+1, s)
	}
	return nil, nil
}
