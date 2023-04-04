package app

import (
	"bookget/config"
	"bookget/lib/file"
	"bookget/lib/gohttp"
	"bookget/lib/util"
	"bookget/lib/zhash"
	"fmt"
	"log"
	"net/http/cookiejar"
	"regexp"
	"strconv"
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
		if pageUrls != nil {
			w.Index = startIndex
			w.BookId = strconv.FormatUint(uint64(zhash.CRC32(v)), 10)
			config.CreateDirectory(v, w.BookId)
			log.Printf("Get %d files.  %s\n", len(pageUrls), v)
			w.do(pageUrls)
		} else {
			config.CreateDirectory(v, "")
			wUrs = append(wUrs, v)
		}
	}
	w.BookId = ""
	w.do(wUrs)
	return "", err
}

func (w Wget) do(wUrls []string) (msg string, err error) {
	size := len(wUrls)
	for _, dUrl := range wUrls {
		sortId := util.GenNumberSorted(w.Index)
		fileName := ""
		ext := file.Extention(dUrl)
		if ext == ".jpg" || ext == ".tif" || ext == ".jp2" || ext == ".png" || ext == ".pdf" {
			fileName = sortId + ext
		} else {
			fileName = file.Name(dUrl)
		}
		log.Printf("Get %d/%d  %s\n", w.Index, size, dUrl)
		w.Index++
		dest := config.GetDestPath(dUrl, w.BookId, fileName)
		cli := gohttp.NewClient()
		_, err = cli.FastGet(dUrl, gohttp.Options{
			DestFile:    dest,
			Concurrency: config.Conf.Threads,
			CookieFile:  config.Conf.CookieFile,
			CookieJar:   w.Jar,
			Headers: map[string]interface{}{
				"User-Agent": config.Conf.UserAgent,
			},
		})
		if err != nil {
			fmt.Println(err)
			break
		}
		fmt.Println()
	}
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
	max, _ := strconv.ParseInt(matches[2], 10, 64)
	iMinLen := len(matches[1])
	startIndex = int(i)

	tmpUrl := regexp.MustCompile(`\((\d+)-(\d+)\)`).ReplaceAllString(sUrl, "%s")
	downloadUrls = make([]string, 0, max)
	for ; i <= max; i++ {
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
