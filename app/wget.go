package app

import (
	"bookget/config"
	"bookget/lib/file"
	"bookget/lib/gohttp"
	"bookget/lib/util"
	"context"
	"fmt"
	"log"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strconv"
	"sync"
)

type Wget struct {
	Index    int
	SavePath string
	Jar      *cookiejar.Jar
	Urls     []string
	BookId   string
}

func (w Wget) Init(iTask int, sUrl string) (msg string, err error) {
	//TODO implement me
	panic("implement me")
}

func (w Wget) getBookId(sUrl string) (bookId string) {
	//TODO implement me
	panic("implement me")
}

func (w Wget) InitMultiple(sUrl []string) (msg string, err error) {
	w.Urls = sUrl
	w.Index = 1
	w.Jar, _ = cookiejar.New(nil)
	return w.download()
}

func (w Wget) download() (msg string, err error) {
	log.Printf(" %d urls.\n", len(w.Urls))
	wUrs := make([]string, 0, len(w.Urls))
	for _, v := range w.Urls {
		//正则匹配
		pageUrls, startIndex := w.getDownloadUrls(v)
		UrlParsed, _ := url.Parse(v)
		w.SavePath = CreateDirectory(UrlParsed.Host, v, "")
		if pageUrls != nil {
			w.Index = startIndex
			log.Printf("Get %d files.  %s\n", len(pageUrls), v)
			w.do(pageUrls)
		} else {
			wUrs = append(wUrs, v)
		}
	}
	w.BookId = ""
	w.do(wUrs)
	return "", err
}

func (w Wget) do(wUrls []string) (msg string, err error) {
	size := len(wUrls)
	var wg sync.WaitGroup
	q := QueueNew(int(config.Conf.Threads))
	for _, dUrl := range wUrls {
		sortId := util.GenNumberSorted(w.Index)
		fileName := ""
		ext := file.Extention(dUrl)
		if len(ext) == 4 && ext[:1] == "." {
			fileName = sortId + ext
		} else {
			fileName = sortId + config.Conf.FileExt
		}
		log.Printf("Get %d/%d  %s\n", w.Index, size, dUrl)
		w.Index++
		dest := w.SavePath + fileName
		imgUrl := dUrl
		wg.Add(1)
		q.Go(func() {
			defer wg.Done()
			ctx := context.Background()
			opts := gohttp.Options{
				DestFile:    dest,
				Overwrite:   false,
				Concurrency: 1,
				CookieFile:  config.Conf.CookieFile,
				CookieJar:   w.Jar,
				Headers: map[string]interface{}{
					"User-Agent": config.Conf.UserAgent,
				},
			}
			gohttp.FastGet(ctx, imgUrl, opts)
		})
		fmt.Println()
		util.PrintSleepTime(config.Conf.Speed)
	}
	wg.Wait()
	return "", err
}

func (w Wget) getVolumes(sUrl string, jar *cookiejar.Jar) (volumes []string, err error) {
	//TODO implement me
	panic("implement me")
}

func (w Wget) getCanvases(sUrl string, jar *cookiejar.Jar) (canvases []string, err error) {
	//TODO implement me
	panic("implement me")
}

func (w Wget) getDownloadUrls(sUrl string) (downloadUrls []string, startIndex int) {
	matches := regexp.MustCompile(`\((\d+)-(\d+)\)`).FindStringSubmatch(sUrl)
	if matches == nil {
		return nil, 0
	}
	i, _ := strconv.ParseInt(matches[1], 10, 64)
	size, _ := strconv.ParseInt(matches[2], 10, 64)
	iMinLen := len(matches[1])
	startIndex = int(i)

	tmpUrl := regexp.MustCompile(`\((\d+)-(\d+)\)`).ReplaceAllString(sUrl, "%s")
	downloadUrls = make([]string, 0, size)
	for ; i <= size; i++ {
		iLen := len(strconv.FormatInt(i, 10))
		if iLen < iMinLen {
			iLen = iMinLen
		}
		sortId := util.GenNumberLimitLen(int(i), iLen)
		dUrl := fmt.Sprintf(tmpUrl, sortId)
		downloadUrls = append(downloadUrls, dUrl)
	}
	return downloadUrls, startIndex
}
