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

type Modernhistory struct {
	dt *DownloadTask
}

type ResponseModernhistoryIiif struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Result  struct {
		CollectNum string `json:"collectNum"`
		Info       struct {
			Title   string `json:"title"`
			IiifObj struct {
				FileCode    string      `json:"fileCode"`
				UniqTag     interface{} `json:"uniqTag"`
				VolumeInfo  interface{} `json:"volumeInfo"`
				DirName     string      `json:"dirName"`
				DirCode     string      `json:"dirCode"`
				CurrentPage string      `json:"currentPage"`
				StartPageId string      `json:"startPageId"`
				ImgUrl      string      `json:"imgUrl"`
				Content     string      `json:"content"`
				JsonUrl     string      `json:"jsonUrl"`
				IsUp        interface{} `json:"isUp"`
			} `json:"iiifObj"`
		} `json:"info"`
	} `json:"result"`
}

type ResponseModernhistoryManifest struct {
	Sequences []struct {
		Canvases []struct {
			Height string `json:"height"`
			Images []struct {
				Id         string `json:"@id"`
				Type       string `json:"@type"`
				Motivation string `json:"motivation"`
				On         string `json:"on"`
				Resource   struct {
					Format  string `json:"format"`
					Height  string `json:"height"`
					Id      string `json:"@id"`
					Type    string `json:"@type"`
					Service struct {
						Protocol string `json:"protocol"`
						Profile  string `json:"profile"`
						Width    int    `json:"width"`
						Id       string `json:"@id"`
						Context  string `json:"@context"`
						Height   int    `json:"height"`
					} `json:"service"`
					Width string `json:"width"`
				} `json:"resource"`
			} `json:"images"`
			Id    string `json:"@id"`
			Type  string `json:"@type"`
			Label string `json:"label"`
			Width string `json:"width"`
		} `json:"canvases"`
		Id               string `json:"@id"`
		Type             string `json:"@type"`
		Label            string `json:"label"`
		ViewingDirection string `json:"viewingDirection"`
		ViewingHint      string `json:"viewingHint"`
	} `json:"sequences"`
}

func (p *Modernhistory) Init(iTask int, sUrl string) (msg string, err error) {
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

func (p *Modernhistory) getBookId(sUrl string) (bookId string) {
	m := regexp.MustCompile(`fileCode=([A-z0-9_-]+)`).FindStringSubmatch(sUrl)
	if m != nil {
		return m[1]
	}
	return ""
}

func (p *Modernhistory) download() (msg string, err error) {
	name := util.GenNumberSorted(p.dt.Index)
	log.Printf("Get %s  %s\n", name, p.dt.Url)
	p.dt.SavePath = config.CreateDirectory(p.dt.Url, p.dt.BookId)
	apiUrl := "https://" + p.dt.UrlParsed.Host + "/backend-prod/esBook/findDetailsInfo/" + p.dt.BookId
	manifestUrl, err := p.getManifestUrl(apiUrl, p.dt.Jar)
	if err != nil {
		return "", err
	}
	canvases, err := p.getCanvases(manifestUrl, p.dt.Jar)
	if err != nil || canvases == nil {
		return "", err
	}
	log.Printf(" %d pages \n", len(canvases))
	return p.do(canvases)
}

func (p *Modernhistory) do(canvases []string) (msg string, err error) {
	if canvases == nil {
		return "", nil
	}
	referer := url.QueryEscape(p.dt.Url)
	args := []string{
		"-H", "Origin:" + referer,
		"-H", "Referer:" + referer,
		"-H", "User-Agent:" + config.Conf.UserAgent,
	}
	size := len(canvases)
	for i, uri := range canvases {
		if uri == "" || !config.PageRange(i, size) {
			continue
		}
		sortId := util.GenNumberSorted(i + 1)
		filename := sortId + config.Conf.FileExt
		inputUri := p.dt.SavePath + string(os.PathSeparator) + sortId + "_info.json"
		bs, err := p.getBody(uri, p.dt.Jar)
		if err != nil {
			continue
		}
		bsNew := regexp.MustCompile(`profile":\[([^{]+)\{"formats":([^\]]+)\],`).ReplaceAll(bs, []byte(`profile":[{"formats":["jpg"],`))
		err = os.WriteFile(inputUri, bsNew, os.ModePerm)
		if err != nil {
			return "", err
		}
		dest := p.dt.SavePath + string(os.PathSeparator) + filename
		if FileExist(dest) {
			continue
		}
		log.Printf("Get %s  %s\n", sortId, uri)
		if ret := util.StartProcess(inputUri, dest, args); ret == true {
			os.Remove(inputUri)
		}
	}
	return "", err
}

func (p *Modernhistory) getManifestUrl(apiUrl string, jar *cookiejar.Jar) (manifestUrl string, err error) {
	bs, err := p.getBody(apiUrl, jar)
	if err != nil {
		return "", err
	}
	var resp ResponseModernhistoryIiif
	if err = json.Unmarshal(bs, &resp); err != nil {
		return "", err
	}
	return resp.Result.Info.IiifObj.JsonUrl, nil
}

func (p *Modernhistory) getVolumes(sUrl string, jar *cookiejar.Jar) (volumes []string, err error) {
	//TODO implement me
	panic("implement me")
}

func (p *Modernhistory) getCanvases(sUrl string, jar *cookiejar.Jar) (canvases []string, err error) {
	bs, err := p.getBody(sUrl, jar)
	if err != nil {
		return nil, err
	}
	var manifest = new(ResponseModernhistoryManifest)
	if err = json.Unmarshal(bs, manifest); err != nil {
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
			iiiInfo := fmt.Sprintf("%s/info.json", image.Resource.Service.Id)
			canvases = append(canvases, iiiInfo)
		}
	}
	return canvases, nil
}

func (p *Modernhistory) getBody(sUrl string, jar *cookiejar.Jar) ([]byte, error) {
	ctx := context.Background()
	cli := gohttp.NewClient(ctx, gohttp.Options{
		CookieFile: config.Conf.CookieFile,
		CookieJar:  jar,
		Headers: map[string]interface{}{
			"User-Agent":      config.Conf.UserAgent,
			"Accept-Language": "zh-CN,zh;q=0.8,zh-TW;q=0.7,zh-HK;q=0.5,en-US;q=0.3,en;q=0.2",
			"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8",
		},
	})
	resp, err := cli.Get(sUrl)
	if err != nil {
		return nil, err
	}
	bs, _ := resp.GetBody()
	if resp.GetStatusCode() != 200 || bs == nil {
		return nil, errors.New(fmt.Sprintf("ErrCode:%d, %s", resp.GetStatusCode(), resp.GetReasonPhrase()))
	}
	return bs, nil
}
func (p *Modernhistory) postBody(sUrl string, d []byte) ([]byte, error) {
	//TODO implement me
	panic("implement me")
}
