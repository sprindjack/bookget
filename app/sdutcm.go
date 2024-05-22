package app

import (
	"bookget/config"
	"bookget/lib/crypt"
	"bookget/lib/gohttp"
	"bookget/lib/util"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strconv"
)

type Sdutcm struct {
	dt    *DownloadTask
	token string
	body  []byte
}

type SdutcmPagePicTxt struct {
	Url       string `json:"url"`
	Text      string `json:"text"`
	Charmax   int    `json:"charmax"`
	ColNum    int    `json:"colNum"`
	PageNum   string `json:"pageNum"`
	ImageList struct {
	} `json:"imageList"`
}
type SdutcmVolumeList struct {
	List []struct {
		ShortTitle string `json:"short_title"`
		ContentId  string `json:"content_id"`
		Lshh       string `json:"lshh"`
		Title      string `json:"title"`
	} `json:"list"`
}

func (p *Sdutcm) Init(iTask int, sUrl string) (msg string, err error) {
	p.dt = new(DownloadTask)
	p.dt.UrlParsed, err = url.Parse(sUrl)
	p.dt.Url = sUrl
	p.dt.Index = iTask
	p.dt.BookId = p.getBookId(p.dt.Url)
	if p.dt.BookId == "" {
		return "requested URL was not found.", err
	}
	p.dt.Jar, _ = cookiejar.New(nil)
	WaitNewCookie()
	return p.download()
}

func (p *Sdutcm) getBookId(sUrl string) (bookId string) {
	m := regexp.MustCompile(`(?i)id=([A-z0-9_-]+)`).FindStringSubmatch(sUrl)
	if m != nil {
		bookId = m[1]
	}
	return bookId
}

func (p *Sdutcm) download() (msg string, err error) {
	name := util.GenNumberSorted(p.dt.Index)
	log.Printf("Get %s  %s\n", name, p.dt.Url)
	p.body, err = p.getPageContent(p.dt.Url)
	if err != nil {
		return "requested URL was not found.", err
	}
	respVolume, err := p.getVolumes(p.dt.Url, p.dt.Jar)
	if err != nil {
		fmt.Println(err)
		return "getVolumes", err
	}
	config.Conf.FileExt = ".pdf"
	for i, vol := range respVolume {
		if !config.VolumeRange(i) {
			continue
		}
		vid := util.GenNumberSorted(i + 1)
		p.dt.SavePath = CreateDirectory(p.dt.UrlParsed.Host, p.dt.BookId, vid)
		canvases, err := p.getCanvases(vol, p.dt.Jar)
		if err != nil || canvases == nil {
			fmt.Println(err)
			continue
		}
		log.Printf(" %d/%d volume, %d pages \n", i+1, len(respVolume), len(canvases))
		p.do(canvases)
	}
	return "", nil
}

func (p *Sdutcm) do(imgUrls []string) (msg string, err error) {
	fmt.Println()
	referer := p.dt.Url
	size := len(imgUrls)
	ctx := context.Background()
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
		log.Printf("Get %d/%d,  URL: %s\n", i+1, size, uri)

		bs, err := getBody(uri, p.dt.Jar)
		var respBody SdutcmPagePicTxt
		if err = json.Unmarshal(bs, &respBody); err != nil {
			break
		}
		csPath := crypt.EncodeURI(respBody.Url)
		pdfUrl := "https://" + p.dt.UrlParsed.Host + "/getencryptFtpPdf.jspx?fileName=" + csPath + p.token
		opts := gohttp.Options{
			DestFile:    dest,
			Overwrite:   false,
			Concurrency: 1,
			CookieFile:  config.Conf.CookieFile,
			CookieJar:   p.dt.Jar,
			Headers: map[string]interface{}{
				"User-Agent": config.Conf.UserAgent,
				"Referer":    referer,
			},
		}
		for k := 0; k < 10; k++ {
			resp, err := gohttp.FastGet(ctx, pdfUrl, opts)
			if err == nil && resp.GetStatusCode() == 200 {
				break
			}
			WaitNewCookieWithMsg(uri)
		}
		util.PrintSleepTime(config.Conf.Speed)
		fmt.Println()
	}
	fmt.Println()
	return "", err
}

func (p *Sdutcm) getVolumes(sUrl string, jar *cookiejar.Jar) (volumes []string, err error) {
	ancientVolume := p.getVolumeId(p.body)
	if err != nil {
		return nil, err
	}
	apiUrl := "https://" + p.dt.UrlParsed.Host + "/sdutcm/ancient/book/getVolume.jspx?lshh=" + ancientVolume
	bs, err := getBody(apiUrl, jar)
	var respBody SdutcmVolumeList
	if err = json.Unmarshal(bs, &respBody); err != nil {
		return nil, err
	}
	for _, m := range respBody.List {
		volUrl := fmt.Sprintf("https://%s/sdutcm/ancient/book/read.jspx?id=%s&pageNum=1", p.dt.UrlParsed.Host, m.ContentId)
		volumes = append(volumes, volUrl)
	}
	return volumes, nil
}

func (p *Sdutcm) getCanvases(sUrl string, jar *cookiejar.Jar) (canvases []string, err error) {
	p.token = p.getToken(p.body)
	size := p.getPageCount(p.body)
	canvases = make([]string, 0, size)
	for i := 1; i <= size; i++ {
		imgUrl := fmt.Sprintf("https://%s/sdutcm/ancient/book/getPagePicTxt.jspx?pageNum=%d&contentId=%s", p.dt.UrlParsed.Host, i, p.dt.BookId)
		canvases = append(canvases, imgUrl)
	}
	return canvases, nil
}

func (p *Sdutcm) getBody(sUrl string, jar *cookiejar.Jar) ([]byte, error) {
	//TODO implement me
	panic("implement me")
}

func (p *Sdutcm) postBody(sUrl string, d []byte) ([]byte, error) {
	//TODO implement me
	panic("implement me")
}

func (p *Sdutcm) getToken(bs []byte) string {
	matches := regexp.MustCompile(`params\s*=\s*["'](\S+)["']`).FindSubmatch(bs)
	if matches != nil {
		return string(matches[1])
	}
	return ""
}

func (p *Sdutcm) getPageCount(bs []byte) int {
	matches := regexp.MustCompile(`pageCount\s+=\s+parseInt\(([0-9]+)\);`).FindSubmatch(bs)
	if matches != nil {
		pageCount, _ := strconv.Atoi(string(matches[1]))
		return pageCount
	}
	return 0
}

func (p *Sdutcm) getVolumeId(bs []byte) string {
	matches := regexp.MustCompile(`ancientVolume\s*=\s*["'](\S+)["'];`).FindSubmatch(bs)
	if matches != nil {
		return string(matches[1])
	}
	return ""
}

func (p *Sdutcm) getPageContent(sUrl string) (bs []byte, err error) {
	p.body, err = getBody(sUrl, p.dt.Jar)
	if err != nil {
		return
	}
	return p.body, err
}
