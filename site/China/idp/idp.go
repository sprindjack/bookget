package idp

import (
	"bookget/app"
	"bookget/config"
	"bookget/lib/gohttp"
	util "bookget/lib/util"
	"context"
	"fmt"
	"log"
	"net/http/cookiejar"
	"net/url"
	"regexp"
)

func Init(iTask int, sUrl string) (msg string, err error) {
	dt := new(DownloadTask)
	dt.CookieJar, _ = cookiejar.New(nil)
	dt.UrlParsed, _ = url.Parse(sUrl)
	dt.Url = sUrl
	dt.Index = iTask
	dt.BookId = getBookId(sUrl)
	return StartDownload(dt)
}

func getBookId(sUrl string) string {
	bookId := ""
	m := regexp.MustCompile(`uid=([A-Za-z0-9]+)`).FindStringSubmatch(sUrl)
	if m != nil {
		bookId = m[1]
	}
	return bookId
}

func StartDownload(dt *DownloadTask) (msg string, err error) {
	canvases := getCanvases(dt.Url, dt)
	if canvases.Size == 0 {
		return
	}
	log.Printf(" %d pages.\n", canvases.Size)

	savePath := app.CreateDirectory(dt.UrlParsed.Host, dt.BookId, "")
	ext := ".jpg"
	ctx := context.Background()
	for i, dUrl := range canvases.ImgUrls {
		if !config.PageRange(i, canvases.Size) {
			continue
		}
		if dUrl == "" {
			continue
		}
		sortId := util.GenNumberSorted(i + 1)
		log.Printf("Get %s  %s\n", sortId, dUrl)
		dest := savePath + sortId + ext
		cli := gohttp.NewClient(ctx, gohttp.Options{
			DestFile:   dest,
			CookieJar:  dt.CookieJar,
			CookieFile: config.Conf.CookieFile,
			Headers: map[string]interface{}{
				"User-Agent": config.Conf.UserAgent,
			},
		})
		_, err = cli.Get(dUrl)
		if err != nil {
			fmt.Println(err)
		}
	}
	return "", nil
}

func getCanvases(sUrl string, dt *DownloadTask) (canvases Canvases) {
	ctx := context.Background()
	cli := gohttp.NewClient(ctx, gohttp.Options{
		Timeout:    0,
		CookieFile: config.Conf.CookieFile,
		CookieJar:  dt.CookieJar,
		Headers: map[string]interface{}{
			"User-Agent": config.Conf.UserAgent,
		},
	})
	resp, err := cli.Get(sUrl)
	if err != nil {
		log.Fatalln(err)
	}
	bs, _ := resp.GetBody()
	//imageUrls[0] = "/image_IDP.a4d?type=loadRotatedMainImage;recnum=31305;rotate=0;imageType=_M";
	//imageRecnum[0] = "31305";
	m := regexp.MustCompile(`imageRecnum\[\d+\][ \S]?=[ \S]?"(\d+)";`).FindAllSubmatch(bs, -1)
	if m == nil {
		return
	}
	canvases.ImgUrls = make([]string, 0, len(m))
	for _, v := range m {
		id := string(v[1])
		imgUrl := fmt.Sprintf("%s://%s/image_IDP.a4d?type=loadRotatedMainImage;recnum=%s;rotate=0;imageType=_L",
			dt.UrlParsed.Scheme, dt.UrlParsed.Host, id)
		canvases.ImgUrls = append(canvases.ImgUrls, imgUrl)
	}
	canvases.Size = len(canvases.ImgUrls)
	return
}
