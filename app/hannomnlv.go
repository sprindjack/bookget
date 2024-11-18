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
	"sync"
)

type HannomNlv struct {
	dt   *DownloadTask
	body []byte
}

func (p *HannomNlv) Init(iTask int, sUrl string) (msg string, err error) {
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

func (p *HannomNlv) getBookId(sUrl string) (bookId string) {
	var err error
	p.body, err = getBody(sUrl, p.dt.Jar)
	if err != nil {
		return ""
	}
	m := regexp.MustCompile(`var[\s+]documentOID[\s+]=[\s+]['"]([^“]+?)['"];`).FindSubmatch(p.body)
	if m != nil {
		return string(m[1])
	}
	return ""
}

func (p *HannomNlv) download() (msg string, err error) {
	name := util.GenNumberSorted(p.dt.Index)
	log.Printf("Get %s  %s\n", name, p.dt.Url)
	p.dt.SavePath = CreateDirectory(p.dt.UrlParsed.Host, p.dt.BookId, "")
	canvases, err := p.getCanvases(p.dt.Url, p.dt.Jar)
	if err != nil || canvases == nil {
		fmt.Println(err)
	}
	log.Printf(" %d pages \n", len(canvases))
	p.do(canvases)
	return "", nil
}

func (p *HannomNlv) do(imgUrls []string) (msg string, err error) {
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
		dest := p.dt.SavePath + filename
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
	return "", nil
}

func (p *HannomNlv) getVolumes(sUrl string, jar *cookiejar.Jar) (volumes []string, err error) {
	//TODO implement me
	panic("implement me")
}

func (p *HannomNlv) getCanvases(sUrl string, jar *cookiejar.Jar) (canvases []string, err error) {
	matches := regexp.MustCompile(`'([^']+)':\{'w':([0-9]+),'h':([0-9]+)\}`).FindAllSubmatch(p.body, -1)
	if matches == nil {
		return nil, errors.New("No image")
	}
	apiUrl := p.dt.UrlParsed.Scheme + "://" + p.dt.UrlParsed.Host
	match := regexp.MustCompile(`imageserverPageTileImageRequest[\s+]=[\s+]['"]([^;]+)['"];`).FindSubmatch(p.body)
	if match != nil {
		apiUrl += string(match[1])
	} else {
		apiUrl += "/hannom/cgi-bin/imageserver/imageserver.pl?color=all&ext=jpg"
	}
	for _, m := range matches {
		imgUrl := apiUrl + fmt.Sprintf("&oid=%s.%s&key=&width=%s&crop=0,0,%s,%s", p.dt.BookId, m[1], m[2], m[2], m[3])
		canvases = append(canvases, imgUrl)
	}
	return canvases, err
}

func (p *HannomNlv) getBody(sUrl string, jar *cookiejar.Jar) ([]byte, error) {
	//TODO implement me
	panic("implement me")
}

func (p *HannomNlv) postBody(sUrl string, d []byte) ([]byte, error) {
	//TODO implement me
	panic("implement me")
}
