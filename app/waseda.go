package app

import (
	"bookget/config"
	"bookget/lib/gohttp"
	"bookget/lib/util"
	"context"
	"errors"
	"fmt"
	"log"
	"net/http/cookiejar"
	"net/url"
	"path"
	"regexp"
	"sort"
)

type Waseda struct {
	dt *DownloadTask
}

func (r Waseda) Init(iTask int, sUrl string) (msg string, err error) {
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

func (r Waseda) getBookId(sUrl string) (bookId string) {
	if m := regexp.MustCompile(`kosho/[A-Za-z0-9_-]+/([A-Za-z0-9_-]+)/([A-Za-z0-9_-]+)/`).FindStringSubmatch(sUrl); m != nil {
		return m[2]
	}
	if m := regexp.MustCompile(`kosho/[A-Za-z0-9_-]+/([A-Za-z0-9_-]+)/`).FindStringSubmatch(sUrl); m != nil {
		return m[1]
	}
	return bookId
}

func (r Waseda) download() (msg string, err error) {
	respVolume, err := r.getVolumes(r.dt.Url, r.dt.Jar)
	if err != nil {
		fmt.Println(err)
		return "getVolumes", err
	}
	if config.Conf.FileExt == ".pdf" {
		for i, vol := range respVolume {
			if config.Conf.Volume > 0 && config.Conf.Volume != i+1 {
				continue
			}
			sortId := util.GenNumberSorted(i + 1)
			r.dt.SavePath = config.CreateDirectory(r.dt.Url, r.dt.BookId)
			log.Printf(" %d/%d volume, URL:%s \n", i+1, len(respVolume), vol)
			filename := sortId + config.Conf.FileExt
			dest := config.GetDestPath(r.dt.Url, r.dt.BookId, filename)
			r.doDownload(vol, dest)
		}
	} else {
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
	}

	return "", nil
}

func (r Waseda) do(imgUrls []string) (msg string, err error) {
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
			Concurrency: config.Conf.Threads,
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

func (r Waseda) getVolumes(sUrl string, jar *cookiejar.Jar) (volumes []string, err error) {
	bs, err := r.getBody(sUrl, jar)
	if err != nil {
		return
	}
	text := string(bs)
	//取册数
	matches := regexp.MustCompile(`href=["'](.+?)\.html["']`).FindAllStringSubmatch(text, -1)
	if matches == nil {
		return
	}
	ids := make([]string, 0, len(matches))
	for _, match := range matches {
		ids = append(ids, match[1])
	}
	sort.Sort(strs(ids))
	volumes = make([]string, 0, len(ids))
	for _, v := range ids {
		var htmlUrl string
		if config.Conf.FileExt == ".pdf" {
			htmlUrl = sUrl + v + ".pdf"
		} else {
			htmlUrl = sUrl + v + ".html"
		}
		volumes = append(volumes, htmlUrl)
	}
	return volumes, nil
}

func (r Waseda) getCanvases(sUrl string, jar *cookiejar.Jar) (canvases []string, err error) {
	bs, err := r.getBody(sUrl, jar)
	if err != nil {
		return
	}
	text := string(bs)
	//取册数
	matches := regexp.MustCompile(`href=["'](.+?)\.jpg["']`).FindAllStringSubmatch(text, -1)
	if matches == nil {
		return
	}
	ids := make([]string, 0, len(matches))
	for _, match := range matches {
		ids = append(ids, match[1])
	}
	sort.Sort(strs(ids))
	canvases = make([]string, 0, len(ids))
	dir, _ := path.Split(sUrl)
	for _, v := range ids {
		imgUrl := dir + v + ".jpg"
		canvases = append(canvases, imgUrl)
	}
	return canvases, nil
}

func (r Waseda) getBody(apiUrl string, jar *cookiejar.Jar) ([]byte, error) {
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

func (r Waseda) doDownload(dUrl, dest string) bool {
	if FileExist(dest) {
		return false
	}
	referer := url.QueryEscape(r.dt.Url)
	opts := gohttp.Options{
		DestFile:    dest,
		Overwrite:   false,
		Concurrency: config.Conf.Threads,
		CookieFile:  config.Conf.CookieFile,
		CookieJar:   r.dt.Jar,
		Headers: map[string]interface{}{
			"User-Agent": config.Conf.UserAgent,
			"Referer":    referer,
		},
	}
	ctx := context.Background()
	_, err := gohttp.FastGet(ctx, dUrl, opts)
	if err == nil {
		fmt.Println()
		return true
	}
	fmt.Println(err)
	fmt.Println()
	return false
}
