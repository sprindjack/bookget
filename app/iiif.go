package app

import (
	"bookget/config"
	"bookget/pkg/gohttp"
	"bookget/pkg/util"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http/cookiejar"
	"net/url"
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
			Id   string `json:"@id"`
			Type string `json:"@type"`
			//兼容某些不正规的网站竟然用了string类型，见https://digitalarchive.npm.gov.tw/Antique/setJsonU?uid=58102&Dept=U
			//Height int    `json:"height"`
			Images []struct {
				Id       string `json:"@id"`
				Type     string `json:"@type"`
				On       string `json:"on"`
				Resource struct {
					Id     string `json:"@id"`
					Type   string `json:"@type"`
					Format string `json:"format"`
					//兼容digitalarchive.npm.gov.tw
					//Height  int    `json:"height"`
					Service struct {
						Id string `json:"@id"`
					} `json:"service"`
					Width int `json:"width"`
				} `json:"resource"`
			} `json:"images"`
			Label string `json:"label"`
			//Width int    `json:"width"`
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
	p.dt.SavePath = CreateDirectory(p.dt.UrlParsed.Host, p.dt.BookId, "")
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
	size := len(manifest.Sequences[0].Canvases)
	canvases = make([]string, 0, size)
	for _, canvase := range manifest.Sequences[0].Canvases {
		for _, image := range canvase.Images {
			if config.Conf.UseDziRs {
				//dezoomify-rs URL
				iiiInfo := fmt.Sprintf("%s/info.json", image.Resource.Service.Id)
				canvases = append(canvases, iiiInfo)
			} else {
				//JPEG URL
				imgUrl := image.Resource.Service.Id + "/" + config.Conf.Format
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
		dest := p.dt.SavePath + filename
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

// AutoDetectManifest
// https://dcollections.lib.keio.ac.jp/sites/default/files/iiif/KAN/110X-24-1/manifest.json
// https://snu.alma.exlibrisgroup.com/view/iiif/presentation/82SNU_INST/12748596580002591/manifest?iiifVersion=3
// https://catalog.lib.kyushu-u.ac.jp/image/manifest/1/820/1446033.json
// https://iiif.dl.itc.u-tokyo.ac.jp/repo/iiif/07956eb1-931c-74ff-61e9-e66d4c30817d/manifest
func (p *IIIF) AutoDetectManifest(iTask int, sUrl string) (msg string, err error) {
	name := util.GenNumberSorted(iTask)
	log.Printf("Auto Detect %s  %s\n", name, sUrl)
	bs, err := getBody(sUrl, nil)
	if err != nil {
		return "", err
	}
	ver, err := p.checkVersion(bs)
	if err != nil {
		jsonUrl := p.getManifestUrl(sUrl, string(bs))
		//查找到新的 jsonUrl
		if jsonUrl != sUrl && jsonUrl != "" {
			bs, err = getBody(jsonUrl, nil)
			if err != nil {
				return "", err
			}
			ver, err = p.checkVersion(bs)
			if err != nil {
				return "", err
			}
			sUrl = jsonUrl
		}
	}
	if ver == 3 {
		var iiif IIIFv3
		return iiif.Init(iTask, sUrl)
	} else if ver == 2 {
		var iiif IIIF
		return iiif.Init(iTask, sUrl)
	}
	return "", err
}

func (p *IIIF) checkVersion(bs []byte) (int, error) {
	var presentation ManifestPresentation
	if err := json.Unmarshal(bs, &presentation); err != nil {
		return 0, err
	}
	if strings.Contains(presentation.Context, "presentation/3/") {
		return 3, nil
	}
	return 2, nil
}

func (p *IIIF) getManifestUrl(pageUrl, text string) string {
	//最后是，相对URI
	u, err := url.Parse(pageUrl)
	if err != nil {
		return ""
	}
	host := fmt.Sprintf("%s://%s/", u.Scheme, u.Host)
	//url包含manifest json
	if strings.Contains(pageUrl, ".json") {
		m := regexp.MustCompile(`manifest=([^&]+)`).FindStringSubmatch(pageUrl)
		if m != nil {
			return p.padUri(host, m[1])
		}
		return pageUrl
	}
	//网页内含manifest URL
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
