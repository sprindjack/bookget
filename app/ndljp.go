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
	"strings"
)

type NdlJP struct {
	dt *DownloadTask
}

func (r NdlJP) Init(iTask int, sUrl string) (msg string, err error) {
	r.dt = new(DownloadTask)
	r.dt.UrlParsed, err = url.Parse(sUrl)
	r.dt.Url = sUrl
	r.dt.Index = iTask
	r.dt.BookId = r.getBookId(r.dt.Url)
	if r.dt.BookId == "" {
		return "requested URL was not found.", err
	}
	r.dt.Jar, _ = cookiejar.New(nil)
	return r.download()
}

func (r NdlJP) getBookId(sUrl string) (bookId string) {
	if m := regexp.MustCompile(`/pid/([A-Za-z0-9]+)`).FindStringSubmatch(sUrl); m != nil {
		bookId = m[1]
	}
	return bookId
}

func (r NdlJP) download() (msg string, err error) {
	respVolume, err := r.getVolumes(r.dt.Url, r.dt.Jar)
	if err != nil {
		fmt.Println(err)
		return "getVolumes", err
	}
	for i, vol := range respVolume {
		if config.Conf.Volume > 0 && config.Conf.Volume != i+1 {
			continue
		}
		vid := util.GenNumberSorted(i + 1)
		r.dt.VolumeId = r.dt.BookId + "_vol." + vid
		canvases, err := r.getCanvases(vol, r.dt.Jar)
		if err != nil || canvases == nil {
			fmt.Println(err)
			continue
		}
		r.dt.SavePath = config.CreateDirectory(r.dt.Url, r.dt.VolumeId)
		log.Printf(" %d/%d volume, %d pages \n", i+1, len(respVolume), len(canvases))
		r.do(canvases)
	}
	return msg, err
}

func (r NdlJP) do(imgUrls []string) (msg string, err error) {
	if imgUrls == nil {
		return
	}
	fmt.Println()
	referer := url.QueryEscape(r.dt.Url)
	size := len(imgUrls)
	ctx := context.Background()
	for i, uri := range imgUrls {
		if uri == "" || !config.PageRange(i, size) {
			continue
		}
		sortId := util.GenNumberSorted(i + 1)
		filename := sortId + config.Conf.FileExt
		dest := config.GetDestPath(r.dt.Url, r.dt.VolumeId, filename)
		if FileExist(dest) {
			continue
		}
		log.Printf("Get %d/%d page, URL: %s\n", i+1, size, uri)
		opts := gohttp.Options{
			DestFile:    dest,
			Overwrite:   false,
			Concurrency: 1,
			CookieFile:  config.Conf.CookieFile,
			CookieJar:   r.dt.Jar,
			Headers: map[string]interface{}{
				"User-Agent": config.Conf.UserAgent,
				"Referer":    referer,
			},
		}
		_, err = gohttp.FastGet(ctx, uri, opts)
		if err != nil {
			fmt.Println(err)
			util.PrintSleepTime(config.Conf.Speed)
		}
		fmt.Println()
	}
	fmt.Println()
	return "", err
}

func (r NdlJP) getVolumes(sUrl string, jar *cookiejar.Jar) (volumes []string, err error) {
	apiUrl := "https://" + r.dt.UrlParsed.Host + "/api/meta/search/toc/facet/" + r.dt.BookId
	bs, err := r.getBody(apiUrl, jar)
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
		Parent interface{} `json:"parent"`
	}
	var result = new(ResponseBody)
	if err = json.Unmarshal(bs, result); err != nil {
		log.Printf("json.Unmarshal failed: %s\n", err)
		return
	}
	if result.Children == nil {
		bs, err := r.getBody("https://"+r.dt.UrlParsed.Host+"/api/item/search/info:ndljp/pid/"+r.dt.BookId, jar)
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

	var template string
	for i, v := range result.Children {
		if i == 0 {
			vUrl, _ := r.getManifestUrl(v.Id)
			template = strings.Replace(vUrl, v.Id, "%s", -1)
		}
		iiifUrl := fmt.Sprintf(template, v.Id)
		volumes = append(volumes, iiifUrl)
	}
	return volumes, nil
}

func (r NdlJP) getCanvases(sUrl string, jar *cookiejar.Jar) (canvases []string, err error) {
	bs, err := r.getBody(sUrl, jar)
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
			//JPEG URL
			imgUrl := fmt.Sprintf("%s/%s/0/default.jpg", image.Resource.Service.Id, newWidth)
			canvases = append(canvases, imgUrl)
		}
	}
	return canvases, nil
}

func (r NdlJP) getBody(apiUrl string, jar *cookiejar.Jar) ([]byte, error) {
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

func (r NdlJP) getManifestUrl(id string) (iiifUrl string, err error) {
	type ResponseBody struct {
		Item struct {
			Pid             string `json:"pid"`
			Parent          string `json:"parent"`
			IiifManifestUrl string `json:"iiifManifestUrl"`
		} `json:"item"`
		MetaMap interface{} `json:"metaMap"`
	}
	apiUrl := "https://dl.ndl.go.jp/api/item/search/info:ndljp/pid/" + id
	bs, err := r.getBody(apiUrl, r.dt.Jar)
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
