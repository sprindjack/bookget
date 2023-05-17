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
	"os"
	"path/filepath"
	"regexp"
)

type Berkeley struct {
	dt *DownloadTask
}

func (r *Berkeley) Init(iTask int, sUrl string) (msg string, err error) {
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

func (r *Berkeley) getBookId(sUrl string) (bookId string) {
	m := regexp.MustCompile(`(?i)record/([A-z0-9_-]+)`).FindStringSubmatch(sUrl)
	if m != nil {
		bookId = m[1]
	}
	return bookId
}

func (r *Berkeley) download() (msg string, err error) {
	name := util.GenNumberSorted(r.dt.Index)
	log.Printf("Get %s  %s\n", name, r.dt.Url)

	r.dt.SavePath = config.CreateDirectory(r.dt.Url, r.dt.BookId)
	canvases, err := r.getCanvases(r.dt.Url, r.dt.Jar)
	if err != nil || canvases == nil {
		return "requested URL was not found.", err
	}
	log.Printf(" %d files \n", len(canvases))
	r.do(canvases)
	return "", nil
}

func (r *Berkeley) do(canvases []string) (msg string, err error) {
	if canvases == nil {
		return
	}
	fmt.Println()
	referer := r.dt.Url
	size := len(canvases)
	for i, dUrl := range canvases {
		if dUrl == "" || !config.PageRange(i, size) {
			continue
		}
		sortId := util.GenNumberSorted(i + 1)
		ext := filepath.Ext(dUrl)
		filename := sortId + ext
		dest := r.dt.SavePath + string(os.PathSeparator) + filename
		if FileExist(dest) {
			continue
		}
		log.Printf("Get %d/%d,  URL: %s\n", i+1, size, dUrl)
		ctx := context.Background()
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
		gohttp.FastGet(ctx, dUrl, opts)
		fmt.Println()
	}
	return "", err
}

func (r *Berkeley) getVolumes(sUrl string, jar *cookiejar.Jar) (volumes []string, err error) {
	//TODO implement me
	panic("implement me")
}

func (r *Berkeley) getCanvases(sUrl string, jar *cookiejar.Jar) (canvases []string, err error) {
	bs, err := r.getBody(sUrl, jar)
	if err != nil {
		return
	}
	matches := regexp.MustCompile(`value="https://([^"]+)\.pdf"`).FindAllSubmatch(bs, -1)
	if matches == nil {
		return nil, errors.New("not match")
	}
	for _, match := range matches {
		pdfUrl := "https://" + string(match[1]) + ".pdf"
		canvases = append(canvases, pdfUrl)
	}
	return
}

func (r *Berkeley) getBody(apiUrl string, jar *cookiejar.Jar) ([]byte, error) {
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
