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
	"sync"
)

type NdlJP struct {
	dt *DownloadTask
}

func (p *NdlJP) Init(iTask int, sUrl string) (msg string, err error) {
	p.dt = new(DownloadTask)
	p.dt.UrlParsed, err = url.Parse(sUrl)
	p.dt.Url = sUrl
	p.dt.Index = iTask
	p.dt.BookId = p.getBookId(p.dt.Url)
	if p.dt.BookId == "" {
		return "requested URL was not found.", err
	}
	p.dt.Jar, _ = cookiejar.New(nil)
	return p.download()
}

func (p *NdlJP) getBookId(sUrl string) (bookId string) {
	if m := regexp.MustCompile(`/pid/([A-Za-z0-9]+)`).FindStringSubmatch(sUrl); m != nil {
		bookId = m[1]
	}
	return bookId
}

func (p *NdlJP) download() (msg string, err error) {
	respVolume, err := p.getVolumes(p.dt.Url, p.dt.Jar)
	if err != nil {
		fmt.Println(err)
		return "getVolumes", err
	}
	for i, vol := range respVolume {
		if config.Conf.Volume > 0 && config.Conf.Volume != i+1 {
			continue
		}
		iiifUrl, _ := p.getManifestUrl(vol)
		if iiifUrl == "" {
			continue
		}
		vid := util.GenNumberSorted(i + 1)
		p.dt.VolumeId = p.dt.BookId + "_vol." + vid
		canvases, err := p.getCanvases(iiifUrl, p.dt.Jar)
		if err != nil || canvases == nil {
			fmt.Println(err)
			continue
		}
		p.dt.SavePath = config.CreateDirectory(p.dt.Url, p.dt.VolumeId)
		log.Printf(" %d/%d volume, %d pages \n", i+1, len(respVolume), len(canvases))
		p.do(canvases)
	}
	return msg, err
}

func (p *NdlJP) do(imgUrls []string) (msg string, err error) {
	if imgUrls == nil {
		return "", nil
	}
	size := len(imgUrls)
	fmt.Println()
	var wg sync.WaitGroup
	q := QueueNew(int(config.Conf.Threads))
	for i, uri := range imgUrls {
		if uri == "" || !config.PageRange(i, size) {
			continue
		}
		sortId := util.GenNumberSorted(i + 1)
		filename := sortId + config.Conf.FileExt
		dest := p.dt.SavePath + string(os.PathSeparator) + filename
		if FileExist(dest) {
			continue
		}
		imgUrl := uri
		fmt.Println()
		log.Printf("Get %d/%d  %s\n", i+1, size, imgUrl)
		wg.Add(1)
		q.Go(func() {
			defer wg.Done()
			ctx := context.Background()
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
			gohttp.FastGet(ctx, imgUrl, opts)
			fmt.Println()
		})
	}
	wg.Wait()
	fmt.Println()
	return "", err
}

func (p *NdlJP) getVolumes(sUrl string, jar *cookiejar.Jar) (volumes []string, err error) {
	apiUrl := "https://" + p.dt.UrlParsed.Host + "/api/meta/search/toc/facet/" + p.dt.BookId
	bs, err := p.getBody(apiUrl, jar)
	if err != nil {
		return
	}
	type ResponseBody struct {
		Pid      string `json:"pid"`
		Id       string `json:"id"`
		Title    string `json:"title"`
		Children []struct {
			Pid     string `json:"pid"`
			Id      string `json:"id"`
			Title   string `json:"title"`
			SortKey string `json:"sortKey"`
			Parent  string `json:"parent"`
			Level   string `json:"level"`
		} `json:"children"`
	}
	var result = new(ResponseBody)
	if err = json.Unmarshal(bs, result); err != nil {
		log.Printf("json.Unmarshal failed: %s\n", err)
		return
	}
	if result.Children == nil {
		bs, err := p.getBody("https://"+p.dt.UrlParsed.Host+"/api/item/search/info:ndljp/pid/"+p.dt.BookId, jar)
		if err != nil {
			return nil, err
		}
		type ResponseBody2 struct {
			Item struct {
				IiifManifestUrl string `json:"iiifManifestUrl"`
			} `json:"item"`
		}
		var result2 = new(ResponseBody2)
		if err = json.Unmarshal(bs, result2); err != nil {
			log.Printf("json.Unmarshal failed: %s\n", err)
			return nil, err
		}
		volumes = append(volumes, result2.Item.IiifManifestUrl)
		return volumes, nil
	}

	volumes = make([]string, 0, len(result.Children))

	for _, v := range result.Children {
		volumes = append(volumes, v.Id)
	}
	return volumes, nil
}

func (p *NdlJP) getCanvases(sUrl string, jar *cookiejar.Jar) (canvases []string, err error) {
	bs, err := p.getBody(sUrl, jar)
	if err != nil {
		return nil, err
	}
	var manifest = new(ResponseManifest)
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
			//JPEG URL
			imgUrl := image.Resource.Service.Id + "/" + config.Conf.Format
			canvases = append(canvases, imgUrl)
		}
	}
	return canvases, nil
}

func (p *NdlJP) getBody(apiUrl string, jar *cookiejar.Jar) ([]byte, error) {
	referer := url.QueryEscape(apiUrl)
	ctx := context.Background()
	cli := gohttp.NewClient(ctx, gohttp.Options{
		CookieFile: config.Conf.CookieFile,
		CookieJar:  jar,
		Headers: map[string]interface{}{
			"User-Agent": config.Conf.UserAgent,
			"Referer":    referer,
		},
	})
	resp, err := cli.Get(apiUrl)
	if err != nil {
		return nil, err
	}
	bs, _ := resp.GetBody()
	if bs == nil {
		return nil, errors.New(fmt.Sprintf("ErrCode:%d, %s", resp.GetStatusCode(), resp.GetReasonPhrase()))
	}
	return bs, nil
}

func (p *NdlJP) getManifestUrl(id string) (iiifUrl string, err error) {
	type ResponseBody struct {
		Item struct {
			Pid             string `json:"pid"`
			Parent          string `json:"parent"`
			IiifManifestUrl string `json:"iiifManifestUrl"`
		} `json:"item"`
		MetaMap interface{} `json:"metaMap"`
	}
	apiUrl := "https://dl.ndl.go.jp/api/item/search/info:ndljp/pid/" + id
	bs, err := p.getBody(apiUrl, p.dt.Jar)
	if err != nil {
		return "", err
	}
	var result ResponseBody
	if err = json.Unmarshal(bs, &result); err != nil {
		log.Printf("json.Unmarshal failed: %s\n", err)
		return
	}
	return result.Item.IiifManifestUrl, nil
}
