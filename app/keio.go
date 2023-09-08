package app

import (
	"bookget/config"
	"bookget/lib/gohttp"
	"context"
	"errors"
	"fmt"
	"net/http/cookiejar"
	"net/url"
	"regexp"
)

type Keio struct {
	dt *DownloadTask
}

func (r *Keio) Init(iTask int, sUrl string) (msg string, err error) {
	r.dt = new(DownloadTask)
	r.dt.UrlParsed, err = url.Parse(sUrl)
	r.dt.Url = sUrl
	r.dt.Index = iTask
	r.dt.BookId = r.getBookId(r.dt.Url)
	if r.dt.BookId == "" {
		return "requested URL was not found.", err
	}
	r.dt.Jar, _ = cookiejar.New(nil)
	//return r.download()
	manifestUrl, err := r.getManifestUrl(sUrl)
	if err != nil {
		return "requested URL was not found.", err
	}
	var iiif IIIF
	return iiif.InitWithId(iTask, manifestUrl, r.dt.BookId)
}

func (r *Keio) getBookId(sUrl string) (bookId string) {
	m := regexp.MustCompile(`bib_frame\?id=([A-z0-9]+)`).FindStringSubmatch(sUrl)
	if m != nil {
		bookId = m[1]
	}
	return bookId
}

func (r *Keio) getManifestUrl(sUrl string) (uri string, err error) {
	//https://db2.sido.keio.ac.jp/kanseki/bib_image?id=
	apiUrl := "https://db2.sido.keio.ac.jp/kanseki/bib_image?id=" + r.dt.BookId
	bs, err := r.getBody(apiUrl, r.dt.Jar)
	if err != nil {
		return
	}
	text := string(bs)
	//"manifestUri": "https://db2.sido.keio.ac.jp/iiif/manifests/kanseki/007387/007387-001/manifest.json",
	m := regexp.MustCompile(`https://([^"]+)manifest.json`).FindStringSubmatch(text)
	if m == nil {
		return
	}
	uri = "https://" + m[1] + "manifest.json"
	return
}

func (r *Keio) download() (msg string, err error) {
	//TODO implement me
	panic("implement me")
}

func (r *Keio) do(imgUrls []string) (msg string, err error) {
	//TODO implement me
	panic("implement me")
}

func (r *Keio) getVolumes(sUrl string, jar *cookiejar.Jar) (volumes []string, err error) {
	//TODO implement me
	panic("implement me")
}

func (r *Keio) getCanvases(sUrl string, jar *cookiejar.Jar) (canvases []string, err error) {
	//TODO implement me
	panic("implement me")
}

func (r *Keio) getBody(apiUrl string, jar *cookiejar.Jar) ([]byte, error) {
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
