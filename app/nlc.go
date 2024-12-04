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
	"strconv"
	"strings"
	"sync"
)

type ChinaNlc struct {
	dt          *DownloadTask
	body        []byte
	dataType    int //0=pdf,1=pic
	aid         string
	vectorBooks []string
}

func (r *ChinaNlc) Init(iTask int, sUrl string) (msg string, err error) {
	r.dt = new(DownloadTask)
	r.dt.UrlParsed, err = url.Parse(sUrl)
	r.dt.Url = sUrl
	r.dt.Index = iTask
	r.dt.Jar, _ = cookiejar.New(nil)
	if strings.Contains(sUrl, "OutOpenBook/Open") {
		r.body, _ = r.getBody(sUrl, r.dt.Jar)
		r.dt.BookId = r.getBookId(string(r.body))
	} else {
		r.dt.BookId = r.getBookId(r.dt.Url)
	}
	if r.dt.BookId == "" {
		return "requested URL was not found.", err
	}
	return r.download()
}

func (r *ChinaNlc) getBookId(sUrl string) (bookId string) {
	m := regexp.MustCompile(`identifier\s+=\s+["']([^"']+)["']`).FindStringSubmatch(sUrl)
	if m != nil {
		bookId = m[1]
		return
	}
	m = regexp.MustCompile(`fid=([A-z0-9]+)`).FindStringSubmatch(sUrl)
	if m != nil {
		bookId = m[1]
		return
	}
	return bookId
}

func (r *ChinaNlc) download() (msg string, err error) {
	name := util.GenNumberSorted(r.dt.Index)
	log.Printf("Get %s  %s\n", name, r.dt.Url)
	//单册PDF
	if strings.Contains(r.dt.Url, "OutOpenBook/OpenObjectBook") {
		//PDF
		r.dt.SavePath = CreateDirectory(r.dt.UrlParsed.Host, r.dt.BookId, "")
		v, _ := r.identifier(r.dt.Url)
		filename := v.Get("bid") + ".pdf"
		err = r.doPdfUrl(r.dt.Url, filename)
		return "", err
	}
	//单张图
	if strings.Contains(r.dt.Url, "OutOpenBook/OpenObjectPic") {
		r.dt.SavePath = CreateDirectory(r.dt.UrlParsed.Host, r.dt.BookId, "")
		canvases, err := r.getCanvases(r.dt.Url, r.dt.Jar)
		if err != nil || canvases == nil {
			return "", err
		}
		log.Printf("  %d pages \n", len(canvases))
		r.do(canvases)
		return "", err
	}
	//对照阅读单册
	if strings.Contains(r.dt.Url, "OpenTwoObjectBook") {
		r.dt.SavePath = CreateDirectory(r.dt.UrlParsed.Host, r.dt.BookId, "")
		v, _ := r.identifier(r.dt.Url)
		filename := v.Get("bid") + ".pdf"
		pageUrl := fmt.Sprintf("%s://%s/OutOpenBook/OpenObjectBook?aid=%s&bid=%s", r.dt.UrlParsed.Scheme, r.dt.UrlParsed.Host,
			v.Get("aid"), v.Get("bid"))
		err = r.doPdfUrl(pageUrl, filename)

		filename = v.Get("cid") + ".pdf"
		pageUrl = fmt.Sprintf("%s://%s/OutOpenBook/OpenObjectBook?aid=%s&bid=%s", r.dt.UrlParsed.Scheme, r.dt.UrlParsed.Host,
			v.Get("aid"), v.Get("cid"))
		err = r.doPdfUrl(pageUrl, filename)
		return "", err
	}
	//多册/多图
	err = r.downloadForPDFs()
	if err != nil {
		fmt.Println(err)
		return "getVolumes", err
	}
	//矢量多册PDF
	r.downloadForOCR()
	return "", nil
}

func (r *ChinaNlc) do(imgUrls []string) (msg string, err error) {
	if imgUrls == nil {
		return
	}
	fmt.Println()
	referer := url.QueryEscape(r.dt.Url)
	size := len(imgUrls)
	var wg sync.WaitGroup
	q := QueueNew(int(config.Conf.Threads))
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
		imgUrl := uri
		log.Printf("Get %d/%d page, URL: %s\n", i+1, size, imgUrl)
		wg.Add(1)
		q.Go(func() {
			defer wg.Done()
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
			gohttp.FastGet(ctx, imgUrl, opts)
			util.PrintSleepTime(config.Conf.Speed)
			fmt.Println()
		})
	}
	wg.Wait()
	fmt.Println()
	return "", err
}

func (r *ChinaNlc) downloadForPDFs() error {
	respVolume, err := r.getVolumes(r.dt.Url, r.dt.Jar)
	if err != nil {
		return err
	}
	size := len(respVolume)
	for i, vol := range respVolume {
		if !config.VolumeRange(i) {
			continue
		}
		vid := util.GenNumberSorted(i + 1)
		//图片
		if strings.Contains(vol, "OpenObjectPic") {
			r.dataType = 1
			r.dt.SavePath = CreateDirectory(r.dt.UrlParsed.Host, r.dt.BookId, vid)
			canvases, err := r.getCanvases(vol, r.dt.Jar)
			if err != nil || canvases == nil {
				fmt.Println(err)
				continue
			}
			log.Printf(" %d/%d volume, %d pages \n", i+1, size, len(canvases))
			r.do(canvases)
		} else {
			//PDF
			r.dt.SavePath = CreateDirectory(r.dt.UrlParsed.Host, r.dt.BookId, "")
			log.Printf("Get %d/%d volume, URL: %s\n", i+1, size, vol)
			filename := vid + ".pdf"
			r.doPdfUrl(vol, filename)
		}
	}
	return nil
}

func (r *ChinaNlc) downloadForOCR() {
	if r.vectorBooks == nil {
		return
	}
	for i, vol := range r.vectorBooks {
		if !config.VolumeRange(i) {
			continue
		}
		vid := util.GenNumberSorted(i + 1)
		r.dt.SavePath = CreateDirectory(r.dt.UrlParsed.Host, r.dt.BookId, "ocr")
		log.Printf("Get %d/%d volume, URL: %s\n", i+1, len(r.vectorBooks), vol)
		filename := vid + ".pdf"
		r.doPdfUrl(vol, filename)
	}
	return
}

func (r *ChinaNlc) getVolumes(sUrl string, jar *cookiejar.Jar) (volumes []string, err error) {
	r.body, err = r.getBody(r.dt.Url, r.dt.Jar)
	if err != nil {
		return nil, err
	}
	text := util.SubText(string(r.body), "<div id=\"multiple\"", "id=\"catalogDiv\">")
	//取册数
	aUrls := regexp.MustCompile(`<a[^>]+class="a1"[^>].+href="/OutOpenBook/([^"]+)"`).FindAllStringSubmatch(text, -1)
	for _, uri := range aUrls {
		pageUrl := fmt.Sprintf("%s://%s/OutOpenBook/%s", r.dt.UrlParsed.Scheme, r.dt.UrlParsed.Host, uri[1])
		volumes = append(volumes, pageUrl)
	}
	//
	aid := ""
	if volumes != nil {
		match := regexp.MustCompile(`aid=([^&]+)`).FindStringSubmatch(volumes[0])
		if match != nil {
			aid = match[1]
		}
	}

	//对照阅读
	twoUrls := regexp.MustCompile(`openTwoBookNew\('([^"']+)','([^"']+)'`).FindAllStringSubmatch(text, -1)
	if twoUrls != nil && aid != "" {
		r.vectorBooks = make([]string, 0, len(twoUrls))
		for _, uri := range twoUrls {
			pageUrl := fmt.Sprintf("%s://%s/OutOpenBook/OpenObjectBook?aid=%s&bid=%s", r.dt.UrlParsed.Scheme, r.dt.UrlParsed.Host, aid, uri[2])
			r.vectorBooks = append(r.vectorBooks, pageUrl)
		}
	}
	return volumes, err
}

func (r *ChinaNlc) doPdfUrl(sUrl, filename string) error {
	dest := r.dt.SavePath + filename
	if FileExist(dest) {
		return nil
	}
	v, err := r.identifier(sUrl)
	if err != nil {
		return err
	}
	tokenKey, timeKey, timeFlag := r.getToken(sUrl)

	//http://read.nlc.cn/menhu/OutOpenBook/getReaderNew
	pdfUrl := fmt.Sprintf("%s://%s/menhu/OutOpenBook/getReaderNew?aid=%s&bid=%s&kime=%s&fime=%s",
		r.dt.UrlParsed.Scheme, r.dt.UrlParsed.Host, v.Get("aid"), v.Get("bid"), timeKey, timeFlag)

	ctx := context.Background()
	opts := gohttp.Options{
		DestFile:    dest,
		Overwrite:   false,
		Concurrency: 1,
		CookieFile:  config.Conf.CookieFile,
		CookieJar:   r.dt.Jar,
		Headers: map[string]interface{}{
			"User-Agent": config.Conf.UserAgent,
			"Referer":    "http://read.nlc.cn/static/webpdf/lib/WebPDFJRWorker.js",
			"Range":      "bytes=0-1",
			"myreader":   tokenKey,
		},
	}
	resp, err := gohttp.FastGet(ctx, pdfUrl, opts)
	if err != nil || resp.GetStatusCode() != 200 {
		fmt.Println(err)
	}
	util.PrintSleepTime(config.Conf.Speed)
	fmt.Println()
	return err
}

func (r *ChinaNlc) getCanvases(sUrl string, jar *cookiejar.Jar) (canvases []string, err error) {
	v, err := r.identifier(sUrl)
	if err != nil {
		return nil, err
	}
	bid, _ := strconv.ParseFloat(v.Get("bid"), 32)
	iBid := int(bid)
	//图片类型检测
	var pageUrl string
	aid := v.Get("aid")
	if aid == "495" || aid == "952" || aid == "467" || aid == "1080" {
		pageUrl = fmt.Sprintf("%s://%s/allSearch/openBookPic?id=%d&l_id=%s&indexName=data_%s",
			r.dt.UrlParsed.Scheme, r.dt.UrlParsed.Host, iBid, v.Get("lid"), aid)
	} else if aid == "022" {
		//中国记忆库图片 不用登录可以查看
		pageUrl = fmt.Sprintf("%s://%s/allSearch/openPic_noUser?id=%d&identifier=%s&indexName=data_%s",
			r.dt.UrlParsed.Scheme, r.dt.UrlParsed.Host, iBid, v.Get("did"), aid)
	} else {
		pageUrl = fmt.Sprintf("%s://%s/allSearch/openPic?id=%d&identifier=%s&indexName=data_%s",
			r.dt.UrlParsed.Scheme, r.dt.UrlParsed.Host, iBid, v.Get("did"), aid)
	}
	//
	bs, err := r.getBody(pageUrl, jar)
	if err != nil {
		return
	}
	matches := regexp.MustCompile(`<img\s+src="(http|https)://(read|mylib).nlc.cn([^"]+)"`).FindAllSubmatch(bs, -1)
	for _, m := range matches {
		imgUrl := r.dt.UrlParsed.Scheme + "://" + r.dt.UrlParsed.Host + string(m[3])
		canvases = append(canvases, imgUrl)
	}
	return canvases, nil
}

func (r *ChinaNlc) identifier(sUrl string) (v url.Values, err error) {
	u, err := url.Parse(sUrl)
	if err != nil {
		return
	}
	m, _ := url.ParseQuery(u.RawQuery)
	if m["aid"] == nil || m["bid"] == nil {
		return nil, errors.New("error aid/bid")
	}
	return m, nil
}

func (r *ChinaNlc) getBody(apiUrl string, jar *cookiejar.Jar) ([]byte, error) {
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

func (r *ChinaNlc) getToken(uri string) (tokenKey, timeKey, timeFlag string) {
	body, err := r.getBody(uri, nil)
	if err != nil {
		log.Printf("Server unavailable: %s", err.Error())
		return
	}
	//<iframe id="myframe" name="myframe" src="" width="100%" height="100%" scrolling="no" frameborder="0" tokenKey="4ADAD4B379874C10864990817734A2BA" timeKey="1648363906519" timeFlag="1648363906519" sflag=""></iframe>
	params := regexp.MustCompile(`(tokenKey|timeKey|timeFlag)="([a-zA-Z0-9]+)"`).FindAllStringSubmatch(string(body), -1)
	//tokenKey := ""
	//timeKey := ""
	//timeFlag := ""
	for _, v := range params {
		if v[1] == "tokenKey" {
			tokenKey = v[2]
		} else if v[1] == "timeKey" {
			timeKey = v[2]
		} else if v[1] == "timeFlag" {
			timeFlag = v[2]
		}
	}
	return
}
