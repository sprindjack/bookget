package app

import (
	"bookget/config"
	"bookget/pkg/gohttp"
	"bookget/pkg/progressbar"
	"bookget/pkg/util"
	"context"
	"errors"
	"fmt"
	"log"
	"net/http/cookiejar"
	"net/url"
	"regexp"
)

type Idp struct {
	dt  *DownloadTask
	bar *progressbar.ProgressBar
}

func (p *Idp) Init(iTask int, sUrl string) (msg string, err error) {
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

func (p *Idp) getBookId(sUrl string) (bookId string) {
	m := regexp.MustCompile(`uid=([A-Za-z0-9]+)`).FindStringSubmatch(sUrl)
	if m != nil {
		bookId = m[1]
	}
	return bookId
}

func (p *Idp) download() (msg string, err error) {
	name := util.GenNumberSorted(p.dt.Index)
	log.Printf("Get %s  %s\n", name, p.dt.Url)

	canvases, err := p.getCanvases(p.dt.BookId, p.dt.Jar)
	if err != nil || canvases == nil {
		fmt.Println(err)
		return "requested URL was not found.", err
	}
	//不按卷下载，所有图片存一个目录
	p.dt.SavePath = CreateDirectory(p.dt.UrlParsed.Host, p.dt.BookId, "")
	sizeCanvases := len(canvases)
	fmt.Println()
	ext := ".jpg"
	p.bar = progressbar.Default(int64(sizeCanvases), "downloading")
	ctx := context.Background()
	for i, imgUrl := range canvases {
		if !config.PageRange(i, sizeCanvases) || imgUrl == "" {
			continue
		}
		sortId := util.GenNumberSorted(i + 1)
		dest := p.dt.SavePath + sortId + ext
		cli := gohttp.NewClient(ctx, gohttp.Options{
			DestFile:   dest,
			CookieJar:  p.dt.Jar,
			CookieFile: config.Conf.CookieFile,
			Headers: map[string]interface{}{
				"User-Agent": config.Conf.UserAgent,
			},
		})
		_, err = cli.Get(imgUrl)
		if err != nil {
			log.Println(err)
			break
		}
		p.bar.Add(1)
	}
	return "", nil
}

func (p *Idp) do(imgUrls []string) (msg string, err error) {
	//TODO implement me
	panic("implement me")
}

func (p *Idp) getVolumes(sUrl string, jar *cookiejar.Jar) (volumes []string, err error) {
	//TODO implement me
	panic("implement me")
}

func (p *Idp) getCanvases(sUrl string, jar *cookiejar.Jar) ([]string, error) {
	bs, err := p.getBody(sUrl, jar)
	if err != nil {
		return nil, err
	}
	//imageUrls[0] = "/image_IDP.a4d?type=loadRotatedMainImage;recnum=31305;rotate=0;imageType=_M";
	//imageRecnum[0] = "31305";
	m := regexp.MustCompile(`imageRecnum\[\d+\][ \S]?=[ \S]?"(\d+)";`).FindAllSubmatch(bs, -1)
	if m == nil {
		return []string{}, nil
	}
	canvases := make([]string, 0, len(m))
	for _, v := range m {
		id := string(v[1])
		imgUrl := fmt.Sprintf("%s://%s/image_IDP.a4d?type=loadRotatedMainImage;recnum=%s;rotate=0;imageType=_L",
			p.dt.UrlParsed.Scheme, p.dt.UrlParsed.Host, id)
		canvases = append(canvases, imgUrl)
	}
	return canvases, nil
}

func (p *Idp) getBody(sUrl string, jar *cookiejar.Jar) ([]byte, error) {
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
	if resp.GetStatusCode() != 200 || bs == nil {
		return nil, errors.New(fmt.Sprintf("ErrCode:%d, %s", resp.GetStatusCode(), resp.GetReasonPhrase()))
	}
	return bs, nil
}

func (p *Idp) postBody(sUrl string, d []byte) ([]byte, error) {
	//TODO implement me
	panic("implement me")
}
