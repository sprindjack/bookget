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
	"strings"
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

func (p *IIIF) Init(iTask int, sUrl string) (msg string, err error) {
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

func (p *IIIF) InitWithId(iTask int, sUrl string, id string) (msg string, err error) {
	p.dt = new(DownloadTask)
	p.dt.UrlParsed, err = url.Parse(sUrl)
	p.dt.Url = sUrl
	p.dt.Index = iTask
	p.dt.Jar, _ = cookiejar.New(nil)
	p.dt.BookId = id
	return p.download()
}

func (p *IIIF) getBookId(sUrl string) (bookId string) {
	m := regexp.MustCompile(`/([^/]+)/manifest.json`).FindStringSubmatch(sUrl)
	if m != nil {
		bookId = m[1]
		return
	}
	return getBookId(sUrl)
}

func (p *IIIF) download() (msg string, err error) {
	p.xmlContent, err = p.getBody(p.dt.Url, p.dt.Jar)
	if err != nil || p.xmlContent == nil {
		return "requested URL was not found.", err
	}
	canvases, err := p.getCanvases(p.dt.Url, p.dt.Jar)
	if err != nil || canvases == nil {
		return
	}
	p.dt.SavePath = config.CreateDirectory(p.dt.Url, p.dt.BookId)
	return p.do(canvases)
}

func (p *IIIF) do(imgUrls []string) (msg string, err error) {
	if config.Conf.UseDziRs {
		p.doDezoomifyRs(imgUrls)
	} else {
		p.doNormal(imgUrls)
	}
	return "", nil
}

func (p *IIIF) getCanvases(sUrl string, jar *cookiejar.Jar) (canvases []string, err error) {
	var manifest = new(ResponseManifest)
	if err = json.Unmarshal(p.xmlContent, manifest); err != nil {
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

func (p *IIIF) getBody(sUrl string, jar *cookiejar.Jar) ([]byte, error) {
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

func (p *IIIF) doDezoomifyRs(iiifUrls []string) bool {
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
		dest := p.dt.SavePath + string(os.PathSeparator) + filename
		if FileExist(dest) {
			continue
		}
		log.Printf("Get %d/%d  %s\n", i+1, size, uri)
		util.StartProcess(uri, dest, args)
	}
	return true
}

func (p *IIIF) doNormal(imgUrls []string) bool {
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
		dest := p.dt.SavePath + string(os.PathSeparator) + filename
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

func (p *IIIF) AutoDetectManifest(iTask int, taskUrl string) (msg string, err error) {
	name := util.GenNumberSorted(iTask)
	log.Printf("Auto Detect %s  %s\n", name, taskUrl)
	ctx := context.Background()
	cli := gohttp.NewClient(ctx, gohttp.Options{
		CookieFile: config.Conf.CookieFile,
		Headers: map[string]interface{}{
			"User-Agent": config.Conf.UserAgent,
		},
	})
	resp, err := cli.Get(taskUrl)
	if err != nil {
		return
	}
	bs, _ := resp.GetBody()
	text := string(bs)
	contentType := resp.GetHeaderLine("content-type")
	if strings.Contains(text, "iiif.io") && (strings.HasPrefix(contentType, "application/json") ||
		strings.HasPrefix(contentType, "application/ld+json")) {
		p.Init(iTask, taskUrl)
		return
	}
	//href="https://dcollections.lib.keio.ac.jp/ja/kanseki/110x-24-1?manifest=https://dcollections.lib.keio.ac.jp/sites/default/files/iiif/KAN/110X-24-1/manifest.json"
	manifestUrl := p.getManifestUrl(taskUrl, text)
	if manifestUrl == "" {
		msg = "URL not found: manifest.json"
		err = errors.New(msg)
		return
	}
	p.Init(iTask, manifestUrl)
	return
}

func (p *IIIF) getManifestUrl(pageUrl, text string) string {
	//最后是，相对URI
	u, err := url.Parse(pageUrl)
	if err != nil {
		return ""
	}
	host := fmt.Sprintf("%s://%s/", u.Scheme, u.Host)
	//优先明显是manifest的
	m := regexp.MustCompile(`manifest=(\S+).json["']`).FindStringSubmatch(text)
	if m != nil {
		return p.padUri(host, m[1]+".json")
	}
	m = regexp.MustCompile(`manifest=(\S+)["']`).FindStringSubmatch(text)
	if m != nil {
		return p.padUri(host, m[1])
	}
	m = regexp.MustCompile(`data-uri=["'](\S+)manifest(\S+).json["']`).FindStringSubmatch(text)
	if m != nil {
		return m[1] + "manifest" + m[2] + ".json"
	}
	m = regexp.MustCompile(`href=["'](\S+)/manifest.json["']`).FindStringSubmatch(text)
	if m == nil {
		return ""
	}
	return p.padUri(host, m[1]+"/manifest.json")
}
func (p *IIIF) padUri(host, uri string) string {
	//https:// 或 http:// 绝对URL
	if strings.HasPrefix(uri, "https://") || strings.HasPrefix(uri, "http://") {
		return uri
	}
	manifestUri := ""
	if uri[0] == '/' {
		manifestUri = uri[1:]
	} else {
		manifestUri = uri
	}
	return host + manifestUri
}
