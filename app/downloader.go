package app

import (
	"bookget/config"
	"bookget/lib/gohttp"
	"bookget/lib/util"
	"context"
	"fmt"
	"log"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
)

type Downloader interface {
	Init(iTask int, sUrl string) (msg string, err error)
	getBookId(sUrl string) (bookId string)
	download() (msg string, err error)
	do(imgUrls []string) (msg string, err error)
	getVolumes(sUrl string, jar *cookiejar.Jar) (volumes []string, err error)
	getCanvases(sUrl string, jar *cookiejar.Jar) (canvases []string, err error)
}

type DownloadTask struct {
	Index     int
	Url       string
	UrlParsed *url.URL
	SavePath  string
	BookId    string
	Title     string
	VolumeId  string
	Param     map[string]interface{} //备用参数
	Jar       *cookiejar.Jar
}

type Volume struct {
	Title string
	Url   string
	Seq   int
}

func NormalDownload(pageUrl, bookId string, imgUrls []string, jar *cookiejar.Jar) (err error) {
	if imgUrls == nil {
		return
	}
	if jar == nil {
		jar, err = cookiejar.New(nil)
	}
	threads := config.Conf.Threads
	if strings.Contains(imgUrls[0], "/full/") || strings.HasSuffix(imgUrls[0], "/0/default.jpg") {
		threads = 1
	}
	size := len(imgUrls)
	ctx := context.Background()
	for i, uri := range imgUrls {
		if uri == "" || !config.PageRange(i, size) {
			continue
		}
		ext := util.FileExt(uri)
		sortId := util.GenNumberSorted(i + 1)
		log.Printf("Get %s  %s\n", sortId, uri)
		filename := sortId + ext
		dest := config.GetDestPath(pageUrl, bookId, filename)
		opts := gohttp.Options{
			DestFile:    dest,
			Overwrite:   false,
			Concurrency: threads,
			CookieFile:  config.Conf.CookieFile,
			CookieJar:   jar,
			Headers: map[string]interface{}{
				"User-Agent": config.Conf.UserAgent,
			},
		}
		_, err = gohttp.FastGet(ctx, uri, opts)
		if err != nil {
			fmt.Println(err)
			break
		}
	}
	return err
}

func DziDownload(pageUrl, bookId string, iiifUrls []string) {
	if iiifUrls == nil {
		return
	}
	referer := url.QueryEscape(pageUrl)
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
		dest := config.GetDestPath(pageUrl, bookId, filename)
		if FileExist(dest) {
			continue
		}
		log.Printf("Get %s  %s\n", sortId, uri)
		util.StartProcess(uri, dest, args)
	}
}

func FileExist(path string) bool {
	fi, err := os.Stat(path)
	if err == nil && fi.Size() > 0 {
		return true
	}
	return false
}
