package app

import (
	"bookget/config"
	"bookget/pkg/gohttp"
	"bookget/pkg/util"
	"context"
	"errors"
	"fmt"
	"log"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

type Kyotou struct {
	dt *DownloadTask
}

func (p *Kyotou) Init(iTask int, sUrl string) (msg string, err error) {
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

func (p *Kyotou) getBookId(sUrl string) (bookId string) {
	if strings.Contains(sUrl, "menu") {
		return getBookId(sUrl)
	}
	return ""
}

func (p *Kyotou) download() (msg string, err error) {
	name := util.GenNumberSorted(p.dt.Index)
	log.Printf("Get %s  %s\n", name, p.dt.Url)

	respVolume, err := p.getVolumes(p.dt.Url, p.dt.Jar)
	if err != nil {
		fmt.Println(err)
		return "getVolumes", err
	}
	sizeVol := len(respVolume)
	for i, vol := range respVolume {
		if !config.VolumeRange(i) {
			continue
		}
		if sizeVol == 1 {
			p.dt.SavePath = CreateDirectory(p.dt.UrlParsed.Host, p.dt.BookId, "")
		} else {
			vid := util.GenNumberSorted(i + 1)
			p.dt.SavePath = CreateDirectory(p.dt.UrlParsed.Host, p.dt.BookId, vid)
		}
		canvases, err := p.getCanvases(vol, p.dt.Jar)
		if err != nil || canvases == nil {
			fmt.Println(err)
			continue
		}
		log.Printf(" %d/%d volume, %d pages \n", i+1, sizeVol, len(canvases))
		p.do(canvases)
	}
	return "", nil
}

func (p *Kyotou) do(imgUrls []string) (msg string, err error) {
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
	return "", err
}

func (p *Kyotou) getVolumes(sUrl string, jar *cookiejar.Jar) (volumes []string, err error) {
	bs, err := p.getBody(sUrl, jar)
	if err != nil {
		return
	}
	//取册数
	matches := regexp.MustCompile(`href=["']?(.+?)\.html["']?`).FindAllSubmatch(bs, -1)
	if matches == nil {
		return
	}
	pos := strings.LastIndex(sUrl, "/")
	hostUrl := sUrl[:pos]
	volumes = make([]string, 0, len(matches))
	for _, v := range matches {
		text := string(v[1])
		if strings.Contains(text, "top") {
			continue
		}
		linkUrl := fmt.Sprintf("%s/%s.html", hostUrl, text)
		volumes = append(volumes, linkUrl)
	}
	return volumes, err
}

func (p *Kyotou) getCanvases(sUrl string, jar *cookiejar.Jar) (canvases []string, err error) {
	bs, err := p.getBody(sUrl, jar)
	if err != nil {
		return
	}
	startPos, ok := p.getVolStartPos(bs)
	if !ok {
		return
	}
	maxPage, ok := p.getVolMaxPage(bs)
	if !ok {
		return
	}
	bookNumber, ok := p.getBookNumber(bs)
	if !ok {
		return
	}
	pos := strings.LastIndex(sUrl, "/")
	pos1 := strings.LastIndex(sUrl[:pos], "/")
	hostUrl := sUrl[:pos1]
	maxPos := startPos + maxPage
	for i := 1; i < maxPos; i++ {
		sortId := util.GenNumberSorted(i)
		imgUrl := fmt.Sprintf("%s/L/%s%s.jpg", hostUrl, bookNumber, sortId)
		canvases = append(canvases, imgUrl)
	}
	return canvases, err
}

func (p *Kyotou) getBody(sUrl string, jar *cookiejar.Jar) ([]byte, error) {
	referer := url.QueryEscape(sUrl)
	ctx := context.Background()
	cli := gohttp.NewClient(ctx, gohttp.Options{
		CookieFile: config.Conf.CookieFile,
		CookieJar:  jar,
		Headers: map[string]interface{}{
			"User-Agent": config.Conf.UserAgent,
			"Referer":    referer,
		},
	})
	resp, err := cli.Get(sUrl)
	if err != nil {
		return nil, err
	}
	bs, _ := resp.GetBody()
	if resp.GetStatusCode() != 200 || bs == nil {
		return nil, errors.New(fmt.Sprintf("ErrCode:%d, %s", resp.GetStatusCode(), resp.GetReasonPhrase()))
	}
	return bs, nil
}

func (p *Kyotou) postBody(sUrl string, d []byte) ([]byte, error) {
	//TODO implement me
	panic("implement me")
}

func (p *Kyotou) getBookNumber(bs []byte) (bookNumber string, ok bool) {
	//当前开始位置
	match := regexp.MustCompile(`var[\s]+bookNum[\s]+=["'\s]*([A-z0-9]+)["'\s]*;`).FindStringSubmatch(string(bs))
	if match == nil {
		return "", false
	}
	return match[1], true
}

func (p *Kyotou) getVolStartPos(bs []byte) (startPos int, ok bool) {
	//当前开始位置
	match := regexp.MustCompile(`var[\s]+volStartPos[\s]*=[\s]*([0-9]+)[\s]*;`).FindStringSubmatch(string(bs))
	if match == nil {
		return 0, false
	}
	startPos, _ = strconv.Atoi(match[1])
	return startPos, true
}

func (p *Kyotou) getVolCurPage(bs []byte) (curPage int, ok bool) {
	//当前开始位置
	match := regexp.MustCompile(`var[\s]+curPage[\s]*=[\s]*([0-9]+)[\s]*;`).FindStringSubmatch(string(bs))
	if match == nil {
		return 0, false
	}
	curPage, _ = strconv.Atoi(match[1])
	return curPage, true
}

func (p *Kyotou) getVolMaxPage(bs []byte) (maxPage int, ok bool) {
	//当前开始位置
	match := regexp.MustCompile(`var[\s]+volMaxPage[\s]*=[\s]*([0-9]+)[\s]*;`).FindStringSubmatch(string(bs))
	if match == nil {
		return 0, false
	}
	maxPage, _ = strconv.Atoi(match[1])
	return maxPage, true
}
