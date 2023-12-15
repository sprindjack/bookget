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
	"regexp"
	"strings"
)

type Szmuseum struct {
	dt    *DownloadTask
	title string
}

func (r *Szmuseum) Init(iTask int, sUrl string) (msg string, err error) {
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

func (r *Szmuseum) getBookId(sUrl string) (bookId string) {
	m := regexp.MustCompile(`BookDetails/([A-z0-9_-]+)`).FindStringSubmatch(sUrl)
	if m != nil {
		bookId = m[1]
	}
	return bookId
}

func (r *Szmuseum) getTitle(text string) (title string) {
	match := regexp.MustCompile(` <title>([^>]+)</title>`).FindStringSubmatch(text)
	if match != nil {
		m := strings.Split(match[1], "-")
		title = strings.TrimSpace(m[0])
	}
	return title
}

func (r *Szmuseum) download() (msg string, err error) {
	name := util.GenNumberSorted(r.dt.Index)
	log.Printf("Get %s  %s\n", name, r.dt.Url)

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
		r.dt.SavePath = CreateDirectory(r.dt.UrlParsed.Host, r.dt.BookId, vid)
		canvases, err := r.getCanvases(vol.Url, r.dt.Jar)
		if err != nil || canvases == nil {
			fmt.Println(err)
			continue
		}
		log.Printf(" %d/%d volume, %d pages \n", i+1, len(respVolume), len(canvases))
		r.do(canvases)
	}
	return "", nil
}

func (r *Szmuseum) do(imgUrls []string) (msg string, err error) {
	if imgUrls == nil {
		return
	}
	fmt.Println()
	referer := r.dt.Url
	size := len(imgUrls)
	ctx := context.Background()
	for i, uri := range imgUrls {
		if uri == "" || !config.PageRange(i, size) {
			continue
		}
		sortId := util.GenNumberSorted(i + 1)
		filename := sortId + config.Conf.FileExt
		dest := r.dt.SavePath + filename
		if FileExist(dest) {
			continue
		}
		log.Printf("Get %d/%d, URL: %s\n", i+1, size, uri)
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
		resp, err := gohttp.FastGet(ctx, uri, opts)
		if err != nil || resp.GetStatusCode() != 200 {
			fmt.Println(err)
			util.PrintSleepTime(60)
			continue
		}
		util.PrintSleepTime(config.Conf.Speed)
		fmt.Println()
	}
	fmt.Println()
	return "", err
}

func (r *Szmuseum) getVolumes(sUrl string, jar *cookiejar.Jar) (volumes []Volume, err error) {
	bs, err := r.getBody(sUrl, jar)
	if err != nil {
		return
	}
	subText := string(bs)
	matches := regexp.MustCompile(`f_ChangeImg\('([A-z\d:_-]+)',\s*this\);">([^<]*)</li>`).FindAllStringSubmatch(subText, -1)
	if matches == nil {
		return
	}
	volumes = make([]Volume, 0, len(matches))
	for _, m := range matches {
		vUrl := fmt.Sprintf("https://%s/Ancient/BookDetails/%s?guid=%s", r.dt.UrlParsed.Host, r.dt.BookId, m[1])
		vol := Volume{
			Title: m[2],
			Url:   vUrl,
			Seq:   0,
		}
		volumes = append(volumes, vol)
	}
	return volumes, nil
}

func (r *Szmuseum) getCanvases(sUrl string, jar *cookiejar.Jar) (canvases []string, err error) {
	bs, err := r.getBody(sUrl, jar)
	if err != nil {
		return
	}
	subText := util.SubText(string(bs), "<ul id=\"images\">", " <div id=\"controls\">")
	matches := regexp.MustCompile(`<img src="([^"]+)"`).FindAllStringSubmatch(subText, -1)
	if matches == nil {
		return
	}
	for _, match := range matches {
		imgUrl := match[1]
		canvases = append(canvases, imgUrl)
	}
	return canvases, nil
}
func (r *Szmuseum) getBody(apiUrl string, jar *cookiejar.Jar) ([]byte, error) {
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
	if resp.GetStatusCode() != 200 || bs == nil {
		return nil, errors.New(fmt.Sprintf("ErrCode:%d, %s", resp.GetStatusCode(), resp.GetReasonPhrase()))
	}
	return bs, nil
}
