package app

import (
	"bookget/config"
	"bookget/lib/gohttp"
	"bookget/lib/util"
	"context"
	"fmt"
	"log"
	"net/http/cookiejar"
	"net/url"
	"regexp"
)

type Nationaljp struct {
	dt    *DownloadTask
	extId string
}

func (p *Nationaljp) Init(iTask int, sUrl string) (msg string, err error) {
	p.dt = new(DownloadTask)
	p.dt.UrlParsed, err = url.Parse(sUrl)
	p.dt.Url = sUrl
	p.dt.Index = iTask
	p.dt.BookId = p.getBookId(p.dt.Url)
	if p.dt.BookId == "" {
		return "requested URL was not found.", err
	}
	p.dt.Jar, _ = cookiejar.New(nil)
	p.extId = "jp2"
	return p.download()
}

func (p *Nationaljp) getBookId(sUrl string) (bookId string) {
	m := regexp.MustCompile(`(?i)BID=([A-z0-9_-]+)`).FindStringSubmatch(sUrl)
	if m != nil {
		return m[1]
	}
	return ""
}

func (p *Nationaljp) download() (msg string, err error) {
	name := util.GenNumberSorted(p.dt.Index)
	log.Printf("Get %s  %s\n", name, p.dt.Url)

	respVolume, err := p.getVolumes(p.dt.Url, p.dt.Jar)
	if err != nil {
		fmt.Println(err)
		return "getVolumes", err
	}
	p.dt.SavePath = CreateDirectory(p.dt.UrlParsed.Host, p.dt.BookId, "")
	for i, vol := range respVolume {
		if !config.VolumeRange(i) {
			continue
		}
		vid := util.GenNumberSorted(i + 1)
		fileName := vid + ".zip"
		dest := p.dt.SavePath + fileName
		if FileExist(dest) {
			continue
		}
		log.Printf(" %d/%d volume, %s\n", i+1, len(respVolume), p.extId)
		p.do(i+1, vol, dest)
	}
	return msg, err
}

func (p *Nationaljp) do(index int, id, dest string) (msg string, err error) {
	apiUrl := "https://" + p.dt.UrlParsed.Host + "/acv/auto_conversion/download"
	data := fmt.Sprintf("DL_TYPE=%s&id_%d=%s", p.extId, index, id)
	ctx := context.Background()
	opts := gohttp.Options{
		DestFile:    dest,
		Overwrite:   false,
		Concurrency: config.Conf.Threads,
		CookieFile:  config.Conf.CookieFile,
		CookieJar:   p.dt.Jar,
		Headers: map[string]interface{}{
			"User-Agent":   config.Conf.UserAgent,
			"Content-Type": "application/x-www-form-urlencoded",
		},
		Body: []byte(data),
	}
	_, err = gohttp.Post(ctx, apiUrl, opts)
	return "", err
}

func (p *Nationaljp) getVolumes(sUrl string, jar *cookiejar.Jar) (volumes []string, err error) {
	apiUrl := fmt.Sprintf("https://%s/DAS/meta/listPhoto?LANG=default&BID=%s&ID=&NO=&TYPE=dljpeg&DL_TYPE=jpeg", p.dt.UrlParsed.Host, p.dt.BookId)
	bs, err := getBody(apiUrl, nil)
	if err != nil {
		return
	}
	text := string(bs)
	//<input type="checkbox" class="check" name="id_2" posi="2" value="M2016092111023960474"
	//取册数
	matches := regexp.MustCompile(`<input[^>]+posi=["']([0-9]+)["'][^>]+value=["']([A-Za-z0-9]+)["']`).FindAllStringSubmatch(text, -1)
	if matches == nil {
		return
	}
	iLen := len(matches)
	for _, match := range matches {
		//跳过全选复选框
		if iLen > 1 && (match[1] == "0" || match[2] == "") {
			continue
		}
		volumes = append(volumes, match[2])
	}
	return volumes, nil
}

func (p *Nationaljp) getCanvases(sUrl string, jar *cookiejar.Jar) (canvases []string, err error) {
	//TODO implement me
	panic("implement me")
}

func (p *Nationaljp) getBody(sUrl string, jar *cookiejar.Jar) ([]byte, error) {
	//TODO implement me
	panic("implement me")
}

func (p *Nationaljp) postBody(sUrl string, d []byte) ([]byte, error) {
	//TODO implement me
	panic("implement me")
}
