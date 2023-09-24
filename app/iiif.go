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
	"os"
	"regexp"
)

type IIIF struct {
	dt         *DownloadTask
	xmlContent []byte
	BookId     string
}

// ResponseManifest  by view-source:https://iiif.lib.harvard.edu/manifests/drs:53262215
type ResponseManifest struct {
	Sequences []struct {
		Canvases []struct {
			Id     string `json:"@id"`
			Type   string `json:"@type"`
			Height int    `json:"height"`
			Images []struct {
				Id       string `json:"@id"`
				Type     string `json:"@type"`
				On       string `json:"on"`
				Resource struct {
					Id      string `json:"@id"`
					Type    string `json:"@type"`
					Format  string `json:"format"`
					Height  int    `json:"height"`
					Service struct {
						Id string `json:"@id"`
					} `json:"service"`
					Width int `json:"width"`
				} `json:"resource"`
			} `json:"images"`
			Label string `json:"label"`
			Width int    `json:"width"`
		} `json:"canvases"`
	} `json:"sequences"`
}

func (f IIIF) Init(iTask int, sUrl string) (msg string, err error) {
	f.dt = new(DownloadTask)
	f.dt.UrlParsed, err = url.Parse(sUrl)
	f.dt.Url = sUrl
	f.dt.Index = iTask
	f.dt.Jar, _ = cookiejar.New(nil)
	f.dt.BookId = f.getBookId(f.dt.Url)
	if f.dt.BookId == "" {
		return "requested URL was not found.", err
	}
	return f.download()
}

func (f IIIF) InitWithId(iTask int, sUrl string, id string) (msg string, err error) {
	f.dt = new(DownloadTask)
	f.dt.UrlParsed, err = url.Parse(sUrl)
	f.dt.Url = sUrl
	f.dt.Index = iTask
	f.dt.Jar, _ = cookiejar.New(nil)
	f.dt.BookId = id
	return f.download()
}

func (f IIIF) getBookId(sUrl string) (bookId string) {
	m := regexp.MustCompile(`/([^/]+)/manifest.json`).FindStringSubmatch(sUrl)
	if m != nil {
		bookId = m[1]
		return
	}
	return getBookId(sUrl)
}

func (f IIIF) download() (msg string, err error) {
	f.xmlContent, err = f.getBody(f.dt.Url, f.dt.Jar)
	if err != nil || f.xmlContent == nil {
		return "requested URL was not found.", err
	}
	canvases, err := f.getCanvases(f.dt.Url, f.dt.Jar)
	if err != nil || canvases == nil {
		return
	}
	f.dt.SavePath = config.CreateDirectory(f.dt.Url, f.dt.BookId)
	return f.do(canvases)
}

func (f IIIF) do(imgUrls []string) (msg string, err error) {
	if config.Conf.UseDziRs {
		f.doDezoomifyRs(imgUrls)
	} else {
		f.doNormal(imgUrls)
	}
	return "", nil
}

func (f IIIF) getCanvases(sUrl string, jar *cookiejar.Jar) (canvases []string, err error) {
	var manifest = new(ResponseManifest)
	if err = json.Unmarshal(f.xmlContent, manifest); err != nil {
		log.Printf("json.Unmarshal failed: %s\n", err)
		return
	}
	if len(manifest.Sequences) == 0 {
		return
	}
	newWidth := ""
	//>2400使用原图
	if config.Conf.FullImageWidth > 2400 {
		newWidth = "full/full"
	} else if config.Conf.FullImageWidth >= 1000 {
		newWidth = fmt.Sprintf("full/%d,", config.Conf.FullImageWidth)
	}

	size := len(manifest.Sequences[0].Canvases)
	canvases = make([]string, 0, size)
	for _, canvase := range manifest.Sequences[0].Canvases {
		for _, image := range canvase.Images {
			if config.Conf.UseDziRs {
				//iifUrl, _ := url.QueryUnescape(image.Resource.Service.Id)
				//dezoomify-rs URL
				iiiInfo := fmt.Sprintf("%s/info.json", image.Resource.Service.Id)
				canvases = append(canvases, iiiInfo)
			} else {
				//JPEG URL
				imgUrl := fmt.Sprintf("%s/%s/0/default.jpg", image.Resource.Service.Id, newWidth)
				canvases = append(canvases, imgUrl)
			}
		}
	}
	return canvases, nil
}

func (f IIIF) getBody(sUrl string, jar *cookiejar.Jar) ([]byte, error) {
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
	//fix bug https://www.dh-jac.net/db1/books/results-iiif.php?f1==nar-h13-01-01&f12=1&enter=portal
	//delete '?'
	if bs[0] != 123 {
		for i := 0; i < len(bs); i++ {
			if bs[i] == 123 {
				bs = bs[i:]
				break
			}
		}
	}
	return bs, nil
}

func (f IIIF) doDezoomifyRs(iiifUrls []string) bool {
	if iiifUrls == nil {
		return false
	}
	referer := url.QueryEscape(f.dt.Url)
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
		dest := f.dt.SavePath + string(os.PathSeparator) + filename
		if FileExist(dest) {
			continue
		}
		log.Printf("Get %d/%d  %s\n", i+1, size, uri)
		util.StartProcess(uri, dest, args)
	}
	return true
}

func (f IIIF) doNormal(imgUrls []string) bool {
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
		dest := f.dt.SavePath + string(os.PathSeparator) + filename
		if FileExist(dest) {
			continue
		}
		log.Printf("Get %d/%d  %s\n", i+1, size, uri)
		opts := gohttp.Options{
			DestFile:    dest,
			Overwrite:   false,
			Concurrency: 1,
			CookieFile:  config.Conf.CookieFile,
			CookieJar:   f.dt.Jar,
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
