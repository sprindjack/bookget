package app

import (
	"bookget/config"
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
	"sync"
)

type Berlin struct {
	dt *DownloadTask
}

func (p *Berlin) Init(iTask int, sUrl string) (msg string, err error) {
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

func (p *Berlin) getBookId(sUrl string) (bookId string) {
	m := regexp.MustCompile(`PPN=([A-z0-9_-]+)`).FindStringSubmatch(sUrl)
	if m != nil {
		bookId = m[1]
	}
	return bookId
}

func (p *Berlin) download() (msg string, err error) {
	name := util.GenNumberSorted(p.dt.Index)
	log.Printf("Get %s  %s\n", name, p.dt.Url)

	respVolume, err := p.getVolumes(p.dt.Url, p.dt.Jar)
	if err != nil {
		fmt.Println(err)
		return "getVolumes", err
	}
	sizeVol := len(respVolume)
	for i, vol := range respVolume {
		if config.Conf.Volume > 0 && config.Conf.Volume != i+1 {
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

func (p *Berlin) do(imgUrls []string) (msg string, err error) {
	if config.Conf.UseDziRs {
		p.doDezoomifyRs(imgUrls)
	} else {
		p.doNormal(imgUrls)
	}
	return "", err
}

func (p *Berlin) getVolumes(sUrl string, jar *cookiejar.Jar) (volumes []string, err error) {
	jsonUrl := fmt.Sprintf("https://content.staatsbibliothek-berlin.de/dc/%s/manifest", p.dt.BookId)
	volumes = append(volumes, jsonUrl)
	return volumes, nil
}

func (p *Berlin) getCanvases(sUrl string, jar *cookiejar.Jar) (canvases []string, err error) {
	bs, err := p.getBody(sUrl, jar)
	if err != nil {
		return
	}
	var manifest = new(ResponseManifest)
	if err = json.Unmarshal(bs, manifest); err != nil {
		log.Printf("json.Unmarshal failed: %s\n", err)
		return
	}
	if len(manifest.Sequences) == 0 {
		return
	}
	size := len(manifest.Sequences[0].Canvases)
	canvases = make([]string, 0, size)
	for _, canvase := range manifest.Sequences[0].Canvases {
		for _, image := range canvase.Images {
			if config.Conf.UseDziRs {
				//dezoomify-rs URL
				//iiiInfo := fmt.Sprintf("%s/info.json", image.Resource.Service.Id)
				//https://ngcs-core.staatsbibliothek-berlin.de/dzi/PPN3303598630/PHYS_0001.dzi
				m := regexp.MustCompile("/dc/([A-z0-9]+)-([A-z0-9]+)/full").FindStringSubmatch(image.Resource.Id)
				iiiInfo := fmt.Sprintf("https://ngcs-core.staatsbibliothek-berlin.de/dzi/%s/PHYS_%s.dzi", p.dt.BookId, m[2])
				canvases = append(canvases, iiiInfo)
			} else {
				//JPEG URL
				//imgUrl := image.Resource.Service.Id + "/" + config.Conf.Format
				//https://content.staatsbibliothek-berlin.de/dc/3303598630-0001/full/full/0/default.jpg
				canvases = append(canvases, image.Resource.Id)
			}
		}
	}
	return canvases, nil

}

func (p *Berlin) getBody(sUrl string, jar *cookiejar.Jar) ([]byte, error) {
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

func (p *Berlin) postBody(sUrl string, d []byte) ([]byte, error) {
	//TODO implement me
	panic("implement me")
}

func (p *Berlin) doDezoomifyRs(iiifUrls []string) bool {
	if iiifUrls == nil {
		return false
	}
	referer := url.QueryEscape(p.dt.Url)
	args := []string{
		"-H", "Origin:" + referer,
		"-H", "Referer:" + referer,
		"-H", "User-Agent:" + config.Conf.UserAgent,
	}
	size := len(iiifUrls)
	for i, uri := range iiifUrls {
		if uri == "" || !config.PageRange(i, size) {
			continue
		}
		sortId := util.GenNumberSorted(i + 1)
		filename := sortId + config.Conf.FileExt
		dest := p.dt.SavePath + filename
		if FileExist(dest) {
			continue
		}
		log.Printf("Get %d/%d  %s\n", i+1, size, uri)
		util.StartProcess(uri, dest, args)
	}
	return true
}

func (p *Berlin) doNormal(imgUrls []string) bool {
	if imgUrls == nil {
		return false
	}
	size := len(imgUrls)
	fmt.Println()
	var wg sync.WaitGroup
	q := QueueNew(int(config.Conf.Threads))
	for i, uri := range imgUrls {
		if uri == "" || !config.PageRange(i, size) {
			continue
		}
		ext := util.FileExt(uri)
		sortId := util.GenNumberSorted(i + 1)
		filename := sortId + ext
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
	return true
}
