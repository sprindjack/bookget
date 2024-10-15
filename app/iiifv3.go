package app

import (
	"bookget/config"
	"bookget/lib/gohttp"
	"bookget/lib/util"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http/cookiejar"
	"net/url"
	"regexp"
)

type IIIFv3 struct {
	dt         *DownloadTask
	xmlContent []byte
	BookId     string
}

// ResponseManifestv3  https://iiif.io/api/presentation/3.0/#52-manifest
type ResponseManifestv3 struct {
	Id    string `json:"id"`
	Type  string `json:"type"`
	Label struct {
		None []string `json:"none"`
	} `json:"label"`
	Height   int `json:"height"`
	Width    int `json:"width"`
	Canvases []struct {
		Id     string `json:"id"`
		Type   string `json:"type"`
		Height int    `json:"height"`
		Width  int    `json:"width"`
		Items  []struct {
			Id    string `json:"id"`
			Type  string `json:"type"`
			Items []struct {
				Id         string `json:"id"`
				Type       string `json:"type"`
				Motivation string `json:"motivation"`
				Body       struct {
					Id      string `json:"id"`
					Type    string `json:"type"`
					Format  string `json:"format"`
					Service []struct {
						Id   string `json:"id"`
						Type string `json:"type"`
						//![ See https://da.library.pref.osaka.jp/api/items/03-0000183/manifest.json
						Id_   string `json:"@id"`
						Type_ string `json:"@type"`
						//]!
						Profile string `json:"profile"`
					} `json:"service"`
					Height int `json:"height"`
					Width  int `json:"width"`
				} `json:"body"`
				Target string `json:"target"`
			} `json:"items"`
		} `json:"items"`
	} `json:"items"`
	Annotations []struct {
		Id    string        `json:"id"`
		Type  string        `json:"type"`
		Items []interface{} `json:"items"`
	} `json:"annotations"`
}

type ManifestPresentation struct {
	Context string `json:"@context"`
	Id      string `json:"id"`
}

func (p *IIIFv3) Init(iTask int, sUrl string) (msg string, err error) {
	p.dt = new(DownloadTask)
	p.dt.UrlParsed, err = url.Parse(sUrl)
	p.dt.Url = sUrl
	p.dt.Index = iTask
	p.dt.Jar, _ = cookiejar.New(nil)
	p.dt.BookId = p.getBookId(p.dt.Url)
	if p.dt.BookId == "" {
		return "requested URL was not found.", err
	}
	return p.download()
}

func (p *IIIFv3) InitWithId(iTask int, sUrl string, id string) (msg string, err error) {
	p.dt = new(DownloadTask)
	p.dt.UrlParsed, err = url.Parse(sUrl)
	p.dt.Url = sUrl
	p.dt.Index = iTask
	p.dt.Jar, _ = cookiejar.New(nil)
	p.dt.BookId = id
	return p.download()
}

func (p *IIIFv3) getBookId(sUrl string) (bookId string) {
	m := regexp.MustCompile(`/([^/]+)/manifest.json`).FindStringSubmatch(sUrl)
	if m != nil {
		bookId = m[1]
		return
	}
	return getBookId(sUrl)
}

func (p *IIIFv3) download() (msg string, err error) {
	p.xmlContent, err = p.getBody(p.dt.Url, p.dt.Jar)
	if err != nil || p.xmlContent == nil {
		return "requested URL was not found.", err
	}
	canvases, err := p.getCanvases(p.dt.Url, p.dt.Jar)
	if err != nil || canvases == nil {
		return
	}
	p.dt.SavePath = CreateDirectory(p.dt.UrlParsed.Host, p.dt.BookId, "")
	return p.do(canvases)
}

func (p *IIIFv3) do(imgUrls []string) (msg string, err error) {
	if config.Conf.UseDziRs {
		p.doDezoomifyRs(imgUrls)
	} else {
		p.doNormal(imgUrls)
	}
	return "", nil
}

func (p *IIIFv3) getCanvases(sUrl string, jar *cookiejar.Jar) (canvases []string, err error) {
	var manifest = new(ResponseManifestv3)
	if err = json.Unmarshal(p.xmlContent, manifest); err != nil {
		log.Printf("json.Unmarshal failed: %s\n", err)
		return
	}
	if len(manifest.Canvases) == 0 {
		return
	}
	size := len(manifest.Canvases)
	canvases = make([]string, 0, size)
	//config.Conf.Format = strings.ReplaceAll(config.Conf.Format, "full/full", "full/max")
	for _, canvase := range manifest.Canvases {
		image := canvase.Items[0].Items[0]
		id := image.Body.Service[0].Id
		if id == "" && image.Body.Service[0].Id_ != "" {
			id = image.Body.Service[0].Id_
		}
		if config.Conf.UseDziRs {
			//dezoomify-rs URL
			iiiInfo := fmt.Sprintf("%s/info.json", id)
			canvases = append(canvases, iiiInfo)
		} else {
			//JPEG URL
			imgUrl := id + "/" + config.Conf.Format
			canvases = append(canvases, imgUrl)
		}
	}
	return canvases, nil
}

func (p *IIIFv3) getBody(sUrl string, jar *cookiejar.Jar) ([]byte, error) {
	ctx := context.Background()
	cli := gohttp.NewClient(ctx, gohttp.Options{
		CookieFile: config.Conf.CookieFile,
		CookieJar:  jar,
		Headers: map[string]interface{}{
			"User-Agent": config.Conf.UserAgent,
		},
	})
	resp, err := cli.Get(sUrl)
	if err != nil {
		return nil, err
	}
	bs, _ := resp.GetBody()
	if bs == nil {
		err = errors.New(resp.GetReasonPhrase())
		return nil, err
	}
	return bs, nil
}

func (p *IIIFv3) doDezoomifyRs(iiifUrls []string) bool {
	if iiifUrls == nil {
		return false
	}
	referer := url.QueryEscape(p.dt.Url)
	args := []string{
		"-H", "Origin:" + referer,
		"-H", "Referer:" + referer,
		"-H", "User-Agent:" + config.Conf.UserAgent,
	}
	size := len(iiifUrls)
	for i, uri := range iiifUrls {
		if uri == "" || !config.PageRange(i, size) {
			continue
		}
		sortId := util.GenNumberSorted(i + 1)
		filename := sortId + config.Conf.FileExt
		dest := p.dt.SavePath + filename
		if FileExist(dest) {
			continue
		}
		log.Printf("Get %d/%d  %s\n", i+1, size, uri)
		util.StartProcess(uri, dest, args)
	}
	return true
}

func (p *IIIFv3) doNormal(imgUrls []string) bool {
	if imgUrls == nil {
		return false
	}
	size := len(imgUrls)
	fmt.Println()
	ctx := context.Background()
	for i, uri := range imgUrls {
		if uri == "" || !config.PageRange(i, size) {
			continue
		}
		ext := util.FileExt(uri)
		sortId := util.GenNumberSorted(i + 1)
		filename := sortId + ext
		dest := p.dt.SavePath + filename
		if FileExist(dest) {
			continue
		}
		log.Printf("Get %d/%d  %s\n", i+1, size, uri)
		opts := gohttp.Options{
			DestFile:    dest,
			Overwrite:   false,
			Concurrency: 1,
			CookieFile:  config.Conf.CookieFile,
			CookieJar:   p.dt.Jar,
			Headers: map[string]interface{}{
				"User-Agent": config.Conf.UserAgent,
			},
		}
		_, err := gohttp.FastGet(ctx, uri, opts)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println()
	}
	return true
}
