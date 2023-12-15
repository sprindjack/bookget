package sdutcm

import (
	"bookget/app"
	"bookget/config"
	"bookget/lib/crypt"
	"bookget/lib/gohttp"
	"bookget/lib/util"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

func Init(iTask int, sUrl string) (msg string, err error) {
	dt := new(DownloadTask)
	dt.UrlParsed, err = url.Parse(sUrl)
	dt.Url = sUrl
	dt.Index = iTask
	return Download(dt)
}

func Download(dt *DownloadTask) (msg string, err error) {
	dt.BookId = getBookId(dt.Url)
	if dt.BookId == "" {
		return "", err
	}
	dt.SavePath = app.CreateDirectory(dt.UrlParsed.Host, dt.BookId, "")

	name := util.GenNumberSorted(dt.Index)
	log.Printf("Get %s  %s\n", name, dt.Url)
	jar, _ := cookiejar.New(nil)
	canvases, token, err := getCanvases(dt.BookId, jar)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	ctx := context.Background()
	for i := 1; i <= canvases.Size; i++ {
		pUrl := fmt.Sprintf("https://gjsztsg.sdutcm.edu.cn/sdutcm/ancient/book/getPagePicTxt.jspx?pageNum=%d&contentId=%s", i, dt.BookId)
		bs, err := getBody(pUrl, jar)
		var respBody ResponseBody
		if err = json.Unmarshal(bs, &respBody); err != nil {
			break
		}
		csPath := crypt.EncodeURI(respBody.Url)
		pdfUrl := "https://gjsztsg.sdutcm.edu.cn/getFtpPdfFile.jspx?fileName=" + csPath + token
		sortId := util.GenNumberSorted(i)
		log.Printf("Get %d/%d  %s\n", i, canvases.Size, pdfUrl)
		dest := dt.SavePath + sortId + ".pdf"
		opts := gohttp.Options{
			DestFile:    dest,
			Overwrite:   false,
			Concurrency: 1,
			CookieFile:  config.Conf.CookieFile,
			CookieJar:   jar,
			Headers: map[string]interface{}{
				"User-Agent": config.Conf.UserAgent,
				"Referer":    "https://gjsztsg.sdutcm.edu.cn/thirdparty/pdfview/pdf.worker.js",
			},
		}
		_, err = gohttp.FastGet(ctx, pdfUrl, opts)
		if err != nil {
			fmt.Println(err)
			break
		}
		//canvases.ImgUrls = append(canvases.ImgUrls, pdfUrl)
		util.PrintSleepTime(config.Conf.Speed)
	}
	return "", err
}

func getBookId(text string) string {
	text = strings.ToLower(text)
	var bookId string
	m := regexp.MustCompile(`(?i)id=([A-z0-9_-]+)`).FindStringSubmatch(text)
	if m != nil {
		bookId = m[1]
	}
	return bookId
}

func getCanvases(bookId string, jar *cookiejar.Jar) (canvases Canvases, token string, err error) {
	apiUrl := fmt.Sprintf("https://gjsztsg.sdutcm.edu.cn/sdutcm/ancient/book/read.jspx?id=%s&pageNum=1", bookId)
	body, err := getBody(apiUrl, jar)
	if err != nil {
		return
	}
	token = getToken(body)
	canvases.Size = getPageCount(body)
	canvases.ImgUrls = make([]string, 0, canvases.Size)
	return canvases, token, nil
}

func getToken(bs []byte) string {
	matches := regexp.MustCompile(`params[\s]*=[\s]*["'](\S+)["']`).FindSubmatch(bs)
	if matches != nil {
		return string(matches[1])
	}
	return ""
}

func getPageCount(bs []byte) int {
	matches := regexp.MustCompile(`pageCount[\s]+=[\s]+parseInt\(([0-9]+)\);`).FindSubmatch(bs)
	if matches != nil {
		pageCount, _ := strconv.Atoi(string(matches[1]))
		return pageCount
	}
	return 0
}

func getBody(apiUrl string, jar *cookiejar.Jar) ([]byte, error) {
	if jar == nil {
		jar, _ = cookiejar.New(nil)
	}
	ctx := context.Background()
	cli := gohttp.NewClient(ctx, gohttp.Options{
		CookieFile: config.Conf.CookieFile,
		CookieJar:  jar,
		Headers: map[string]interface{}{
			"User-Agent": config.Conf.UserAgent,
		},
	})
	resp, err := cli.Get(apiUrl)
	if err != nil {
		return nil, err
	}
	bs, _ := resp.GetBody()
	if bs == nil {
		err = errors.New(resp.GetReasonPhrase())
		return nil, err
	}
	return bs, nil
}
