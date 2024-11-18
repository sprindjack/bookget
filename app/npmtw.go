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
	"sync"
)

type NpmTw struct {
	dt   *DownloadTask
	body []byte
}

func (p *NpmTw) Init(iTask int, sUrl string) (msg string, err error) {
	p.dt = new(DownloadTask)
	p.dt.Jar, _ = cookiejar.New(nil)
	p.body, err = p.getBody(sUrl, p.dt.Jar)
	if err != nil {
		return "", err
	}
	p.dt.UrlParsed, err = url.Parse(sUrl)
	p.dt.Url = sUrl
	p.dt.Index = iTask
	p.dt.BookId = p.getBookId(p.dt.Url)
	if p.dt.BookId == "" {
		return "requested URL was not found.", err
	}
	return p.download()
}

func (p *NpmTw) getBookId(sUrl string) (bookId string) {
	//<tr><th>統一編號</th><td>平圖018255-018299</td></tr>
	//<tr><th>題名</th><td><span class=red>增補六臣注文選</span> 存四十五卷</td></tr>
	m := regexp.MustCompile(`<tr><th>統一編號</th><td>(.+?)</td></tr>`).FindSubmatch(p.body)
	if m == nil {
		m = regexp.MustCompile(`<tr><th>題名</th><td>(.+?)</td></tr>`).FindSubmatch(p.body)
		if m == nil {
			return ""
		}
	}
	re := regexp.MustCompile(`<([^>])*>`)
	bookId = re.ReplaceAllString(string(m[1]), "")
	//bookId = strings.ReplaceAll(strings.ReplaceAll(s, "  ", ""), " ", ".")
	bookId = regexp.MustCompile(`[<>\\/?:|\s$]*`).ReplaceAllString(bookId, "")
	return bookId
}

func (p *NpmTw) download() (msg string, err error) {
	name := util.GenNumberSorted(p.dt.Index)
	log.Printf("Get %s  %s\n", name, p.dt.Url)

	respVolume, err := p.getVolumes(p.dt.Url, p.dt.Jar)
	if err != nil {
		fmt.Println(err)
		return "getVolumes", err
	}
	sizeVol := len(respVolume)
	//ctx := context.Background()
	var wg sync.WaitGroup
	q := QueueNew(int(config.Conf.Threads))
	for i, vol := range respVolume {
		if !config.VolumeRange(i) {
			continue
		}
		vid := util.GenNumberSorted(i + 1)
		if sizeVol == 1 || config.Conf.MergePDFs {
			p.dt.SavePath = CreateDirectory(p.dt.UrlParsed.Host, p.dt.BookId, "")
		} else {
			p.dt.SavePath = CreateDirectory(p.dt.UrlParsed.Host, p.dt.BookId, vid)
		}
		canvases, err := p.getCanvases(vol, p.dt.Jar)
		if err != nil || canvases == nil {
			fmt.Println(err)
			continue
		}
		if config.Conf.MergePDFs {
			filename := vid + ".pdf"
			dest := p.dt.SavePath + filename
			if FileExist(dest) {
				continue
			}
			pdfUrl := canvases[0]
			fmt.Println()
			log.Printf(" %d/%d volume, %s \n", i+1, sizeVol, pdfUrl)
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
				gohttp.FastGet(ctx, pdfUrl, opts)
				fmt.Println()
			})

		} else {
			log.Printf(" %d/%d volume, %d PDFs \n", i+1, sizeVol, len(canvases))
			p.do(canvases)
		}
	}
	wg.Wait()
	fmt.Println()
	return "", nil
}

func (p *NpmTw) do(imgUrls []string) (msg string, err error) {
	if imgUrls == nil {
		return "", nil
	}
	size := len(imgUrls)
	fmt.Println()
	ctx := context.Background()
	for i, uri := range imgUrls {
		if uri == "" || !config.PageRange(i, size) {
			continue
		}
		sortId := util.GenNumberSorted(i + 1)
		filename := sortId + ".pdf"
		dest := p.dt.SavePath + filename
		if FileExist(dest) {
			continue
		}
		pdfUrl := uri
		fmt.Println()
		log.Printf("Get %d/%d  %s\n", i+1, size, pdfUrl)
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
		gohttp.FastGet(ctx, pdfUrl, opts)
		fmt.Println()
	}
	fmt.Println()
	return "", nil
}

func (p *NpmTw) getVolumes(sUrl string, jar *cookiejar.Jar) (volumes []string, err error) {
	bodyText := string(p.body)
	//matches := regexp.MustCompile(`(?i)href=['"](\S+?)ACTION=TQ(\S+?)["']`).FindAllStringSubmatch(bodyText, -1)
	start := strings.Index(bodyText, "id=tree_title")
	if start == -1 {
		start = strings.Index(bodyText, "id=\"tree_title\"")
		if start == -1 {
			return
		}
	}
	subText1 := bodyText[start:]
	end := strings.Index(subText1, "id=\"footer-wrapper\"")
	if end == -1 {
		return
	}
	subText := subText1[:end]
	matches := regexp.MustCompile(`href=['"](\S+)["']`).FindAllStringSubmatch(subText, -1)

	if matches == nil {
		return nil, errors.New("[getVolumes]no links")
	}
	for i, match := range matches {
		if i == 0 {
			//跳过第一个链接
			continue
		}
		//v := match[1] + "action=TQ" + match[2]
		v := match[1]
		if v[0] != '/' {
			v = "/" + v
		}
		link := "https://" + p.dt.UrlParsed.Host + v
		volumes = append(volumes, link)
	}
	return volumes, nil
}

func (p *NpmTw) getCanvases(sUrl string, jar *cookiejar.Jar) (canvases []string, err error) {
	bs, err := p.getBody(sUrl, jar)
	if err != nil {
		return
	}
	bodyText := string(bs)
	matches := regexp.MustCompile(`(?i)href=['"]?([A-z\d]{56})["']?`).FindAllStringSubmatch(bodyText, -1)
	if matches == nil {
		return nil, errors.New("[getCanvases]no images")
	}
	id, sec, _ := p.getIdSecu(bodyText)

	//整册合并为一个PDF？
	if config.Conf.MergePDFs {
		i := len(matches)
		first := matches[0][1]
		last := matches[i-1][1]
		newId := fmt.Sprintf("%s%s", last[:4], first[4:])
		link := fmt.Sprintf("https://%s/npmtpc/npmtpall?ID=%s&SECU=%s&ACTION=UI,%s", p.dt.UrlParsed.Host, id, sec, newId)
		canvases = append(canvases, link)
		return
	}
	// 按台北故宫官方提供的URL下载若干个PDF
	for _, match := range matches {
		v := match[1] + "&action=TQ" + match[1]
		link := fmt.Sprintf("https://%s/npmtpc/npmtpall?ID=%s&SECU=%s&ACTION=UI,%s", p.dt.UrlParsed.Host, id, sec, v)
		canvases = append(canvases, link)
	}
	return canvases, nil
}

func (p *NpmTw) getBody(sUrl string, jar *cookiejar.Jar) ([]byte, error) {
	ctx := context.Background()
	cli := gohttp.NewClient(ctx, gohttp.Options{
		CookieFile: config.Conf.CookieFile,
		CookieJar:  jar,
		Headers: map[string]interface{}{
			"User-Agent": config.Conf.UserAgent,
		},
	})
	resp, err := cli.Get(sUrl)
	if err != nil {
		return nil, err
	}
	bs, _ := resp.GetBody()
	if bs == nil {
		return nil, errors.New(fmt.Sprintf("ErrCode:%d, %s", resp.GetStatusCode(), resp.GetReasonPhrase()))
	}
	return bs, nil
}

func (p *NpmTw) postBody(sUrl string, d []byte) ([]byte, error) {
	//TODO implement me
	panic("implement me")
}

func (p *NpmTw) getIdSecu(text string) (id string, sec string, tphc string) {
	//<input type=hidden name=ID value=3408><input type=hidden name=SECU value=536673644>
	//<input type=hidden name=TPHC value=1 size=30>
	matches := regexp.MustCompile(`<input\s+type=hidden\s+name=(ID|SECU|TPHC)\s+value=(\d+)`).FindAllStringSubmatch(text, -1)
	if matches == nil {
		return
	}

	for _, v := range matches {
		if v[1] == "ID" {
			id = v[2]
		} else if v[1] == "SECU" {
			sec = v[2]
		} else if v[1] == "TPHC" {
			tphc = v[2]
		}
	}
	return
}
