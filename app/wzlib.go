package app

import (
	"bookget/config"
	"bookget/lib/gohttp"
	"bookget/lib/util"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http/cookiejar"
	"net/url"
	"regexp"
)

type Wzlib struct {
	dt *DownloadTask
}

type WzlibDigital struct {
	ID                  int    `json:"ID"`
	SiteID              int    `json:"SiteID"`
	Title               string `json:"Title"`
	Author              string `json:"author"`
	Source              string `json:"source"`
	Txt                 string `json:"txt"`
	PdfUrl              string `json:"pdf_url"`
	DigitalResourceData []struct {
		Title string `json:"Title"`
		Url   string `json:"Url"`
	} `json:"DigitalResourceData"`
}

type WzlibResult []WzlibItem

type WzlibItem struct {
	Items []struct {
		Id          string `json:"_id"`
		DcPublisher string `json:"dc_publisher"`
		DcTitle     string `json:"dc_title"`
		WzlPdfUrl   string `json:"wzl_pdf_url"`
	} `json:"items"`
	Title string `json:"title"`
}

type WzlibPdfUrls []WzlibPdfUrl
type WzlibPdfUrl struct {
	Url  string
	Name string
}

type WzlibResultPdf struct {
	Data struct {
		Id         string `json:"_id"`
		DcTitle    string `json:"dc_title"`
		ModelId    string `json:"model_id"`
		RelateName string `json:"relate_name"`
		WzlPdfUrl  string `json:"wzl_pdf_url"`
	} `json:"Data"`

	RelateList []interface{} `json:"RelateList"`
}

func (p *Wzlib) Init(iTask int, sUrl string) (msg string, err error) {
	p.dt = new(DownloadTask)
	p.dt.UrlParsed, err = url.Parse(sUrl)
	p.dt.Url = sUrl
	p.dt.Index = iTask
	p.dt.BookId = p.getBookId(p.dt.Url)
	if p.dt.BookId == "" {
		return "requested URL was not found.", err
	}
	p.dt.Jar, _ = cookiejar.New(nil)
	return p.download()
}

func (p *Wzlib) getBookId(sUrl string) (bookId string) {
	m := regexp.MustCompile(`(?i)id=([A-z0-9_-]+)`).FindStringSubmatch(sUrl)
	//m := regexp.MustCompile(`\?id=([A-z\d]+)`).FindStringSubmatch(sUrl)
	if m != nil {
		bookId = m[1]
	}
	return bookId
}

func (p *Wzlib) download() (msg string, err error) {
	name := util.GenNumberSorted(p.dt.Index)
	log.Printf("Get %s  %s\n", name, p.dt.Url)
	p.dt.SavePath = CreateDirectory(p.dt.UrlParsed.Host, p.dt.BookId, "")

	//旧版：瓯越记忆
	if p.dt.UrlParsed.Host == "oyjy.wzlib.cn" {
		canvases, err := p.OyjyGetCanvases(p.dt.BookId)
		if err != nil || canvases == nil {
			fmt.Println(err)
		}
		return p.do(canvases)
	}
	//新版温州图书馆
	canvases, err := p.getCanvases(p.dt.Url, p.dt.Jar)
	if err != nil || canvases == nil {
		fmt.Println(err)
	}
	return p.do(canvases)
}

func (p *Wzlib) do(dUrls []string) (msg string, err error) {
	if dUrls == nil {
		return
	}
	fmt.Println()
	size := len(dUrls)
	log.Printf(" %d PDFs.\n", size)
	ctx := context.Background()
	for i, uri := range dUrls {
		if !config.PageRange(i, size) {
			continue
		}
		if uri == "" {
			continue
		}
		log.Printf("Get %d/%d, URL: %s\n", i+1, size, uri)
		sortId := util.GenNumberSorted(i + 1)
		filename := sortId + ".pdf"
		dest := p.dt.SavePath + filename
		if FileExist(dest) {
			continue
		}
		opts := gohttp.Options{
			DestFile:    dest,
			Overwrite:   false,
			Concurrency: config.Conf.Threads,
			CookieFile:  config.Conf.CookieFile,
			CookieJar:   p.dt.Jar,
		}
		_, err = gohttp.FastGet(ctx, uri, opts)
		if err != nil {
			fmt.Println(err)
			continue
		}
		fmt.Println()
	}
	fmt.Println()
	return "", err
}

func (p *Wzlib) getVolumes(sUrl string, jar *cookiejar.Jar) (volumes []string, err error) {
	//TODO implement me
	panic("implement me")
}

func (p *Wzlib) getCanvases(sUrl string, jar *cookiejar.Jar) (canvases []string, err error) {
	apiUrl := fmt.Sprintf("https://%s/search/juhe_detail/%s/true?Flag=s", p.dt.UrlParsed.Host, p.dt.BookId)
	bs, err := getBody(apiUrl, jar)
	if err != nil {
		return
	}

	var resT = new(WzlibDigital)
	if err = json.Unmarshal(bs, &resT); err != nil {
		log.Printf("json.Unmarshal failed: %s\n", err)
		return
	}
	for _, ret := range resT.DigitalResourceData {
		m := regexp.MustCompile(`file=(\S+)`).FindStringSubmatch(ret.Url)
		if m == nil {
			continue
		}
		pdfUrl := "https://db.wzlib.cn" + m[1]
		canvases = append(canvases, pdfUrl)
	}
	return canvases, nil
}

func (p *Wzlib) getBody(sUrl string, jar *cookiejar.Jar) ([]byte, error) {
	//TODO implement me
	panic("implement me")
}

func (p *Wzlib) postBody(sUrl string, d []byte) ([]byte, error) {
	//TODO implement me
	panic("implement me")
}

func (p *Wzlib) OyjyGetCanvases(bookId string) (canvases []string, err error) {
	//一册
	uri := fmt.Sprintf("https://oyjy.wzlib.cn/api/search/v1/resource/%s", bookId)
	bs, err := getBody(uri, p.dt.Jar)
	if err == nil {
		var result WzlibResultPdf
		if err = json.Unmarshal(bs, &result); err == nil {
			m := regexp.MustCompile(`file=(\S+)`).FindStringSubmatch(result.Data.WzlPdfUrl)
			if m != nil {
				pdfUrl := "https://db.wzlib.cn" + m[1]
				canvases = append(canvases, pdfUrl)
				return canvases, err
			}
		}
	}

	//多册
	relatedUri := fmt.Sprintf("https://oyjy.wzlib.cn/api/search/v1/resource_related/%s", bookId)
	bs, err = getBody(relatedUri, p.dt.Jar)
	if err != nil {
		return
	}
	var result WzlibResult
	if err = json.Unmarshal(bs, &result); err != nil {
		return
	}
	for _, v := range result[0].Items {
		if v.WzlPdfUrl == "" {
			continue
		}
		pdfUrl := "https://db.wzlib.cn" + v.WzlPdfUrl
		canvases = append(canvases, pdfUrl)
	}
	return canvases, err
}
