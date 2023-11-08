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
	Height int `json:"height"`
	Width  int `json:"width"`
	Items  []struct {
		Id     string `json:"id"`
		Type   string `json:"type"`
		Height int    `json:"height"`
		Width  int    `json:"width"`
		Items  []struct {
			Type  string `json:"type"`
			Items []struct {
				Type       string `json:"type"`
				Motivation string `json:"motivation"`
				Body       struct {
					Id      string `json:"id"`
					Type    string `json:"type"`
					Format  string `json:"format"`
					Service []struct {
						Id      string `json:"id"`
						Type    string `json:"type"`
						Profile string `json:"profile"`
					} `json:"service"`
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
	p.dt.SavePath = config.CreateDirectory(p.dt.Url, p.dt.BookId)
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
	if len(manifest.Items) == 0 {
		return
	}
	newWidth := ""
	//>2400使用原图
	if config.Conf.FullImageWidth > 2400 {
		newWidth = "full/max"
	} else if config.Conf.FullImageWidth >= 1000 {
		newWidth = fmt.Sprintf("full/%d,", config.Conf.FullImageWidth)
	}

	size := len(manifest.Items)
	canvases = make([]string, 0, size)
	for _, canvase := range manifest.Items {
		for _, image := range canvase.Items[0].Items {
			if config.Conf.UseDziRs {
				//iifUrl, _ := url.QueryUnescape(image.Resource.Service[0].Id)
				//dezoomify-rs URL
				iiiInfo := fmt.Sprintf("%s/info.json", image.Body.Service[0].Id)
				canvases = append(canvases, iiiInfo)
			} else {
				//JPEG URL
				imgUrl := fmt.Sprintf("%s/%s/0/default.jpg", image.Body.Service[0].Id, newWidth)
				canvases = append(canvases, imgUrl)
			}
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
		dest := p.dt.SavePath + string(os.PathSeparator) + filename
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

func (p *IIIFv3) AutoDetectManifest(iTask int, sUrl string) (msg string, err error) {
	name := util.GenNumberSorted(iTask)
	log.Printf("Auto Detect %s  %s\n", name, sUrl)
	ctx := context.Background()
	cli := gohttp.NewClient(ctx, gohttp.Options{
		CookieFile: config.Conf.CookieFile,
		Headers: map[string]interface{}{
			"User-Agent": config.Conf.UserAgent,
		},
	})
	resp, err := cli.Get(sUrl)
	if err != nil {
		return
	}
	bs, _ := resp.GetBody()
	//https://dcollections.lib.keio.ac.jp/sites/default/files/iiif/KAN/110X-24-1/manifest.json
	//https://snu.alma.exlibrisgroup.com/view/iiif/presentation/82SNU_INST/12748596580002591/manifest?iiifVersion=3
	//https://catalog.lib.kyushu-u.ac.jp/image/manifest/1/820/1446033.json
	manifestUrl := p.getManifestUrl(sUrl, string(bs))
	//查找到新的 manifestUrl
	if manifestUrl != sUrl {
		resp, err = cli.Get(manifestUrl)
		if err != nil {
			return
		}
		bs, _ = resp.GetBody()
	}
	var presentation ManifestPresentation
	if err = json.Unmarshal(bs, &presentation); err == nil {
		if strings.Contains(presentation.Context, "presentation/3/") {
			var iiif IIIFv3
			return iiif.Init(iTask, manifestUrl)
		} else {
			return p.Init(iTask, manifestUrl)
		}
	}
	return "", errors.New("URL not found: manifest.json")
	return
}

func (p *IIIFv3) getManifestUrl(pageUrl, text string) string {
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
func (p *IIIFv3) padUri(host, uri string) string {
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
