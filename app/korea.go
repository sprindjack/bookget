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
	"sync"
)

type Korea struct {
	dt   *DownloadTask
	body []byte
}

type KoreaResponse struct {
	ImgInfos []struct {
		BookNum  string `json:"bookNum"`
		Num      string `json:"num"`
		BookPath string `json:"bookPath"`
		ImgNum   string `json:"imgNum"`
		Fname    string `json:"fname"`
	} `json:"imgInfos"`
	BookNum string `json:"bookNum"`
}

func (p *Korea) Init(iTask int, sUrl string) (msg string, err error) {
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

func (p *Korea) getBookId(sUrl string) (bookId string) {
	m := regexp.MustCompile(`uci=([^&]+)`).FindStringSubmatch(sUrl)
	if m != nil {
		return m[1]
	}
	return ""
}

func (p *Korea) download() (msg string, err error) {
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
		if err != nil || vol.Canvases == nil {
			continue
		}
		log.Printf(" %d/%d volume, %d pages \n", i+1, sizeVol, len(vol.Canvases))
		p.do(vol.Canvases)
	}
	return "", nil
}

func (p *Korea) do(imgUrls []string) (msg string, err error) {
	if imgUrls == nil {
		return
	}
	fmt.Println()
	size := len(imgUrls)

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
		log.Printf("Get %d/%d, %s\n", i+1, size, imgUrl)
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
			util.PrintSleepTime(config.Conf.Speed)
			fmt.Println()
		})
	}
	wg.Wait()
	fmt.Println()
	return "", err
}

func (p *Korea) getVolumes(sUrl string, jar *cookiejar.Jar) (volumes []PartialCanvases, err error) {
	bs, err := getBody(sUrl, jar)
	if err != nil {
		return nil, err
	}
	matches := regexp.MustCompile(`var[\s+]bookInfos[\s+]=[\s+]([^;]+);`).FindSubmatch(bs)
	if matches == nil {
		return
	}
	resp := make([]KoreaResponse, 0, 100)
	if err = json.Unmarshal(matches[1], &resp); err != nil {
		return nil, err
	}
	ossHost := fmt.Sprintf("%s://%s/data/des/%s/IMG/", p.dt.UrlParsed.Scheme, p.dt.UrlParsed.Host, p.dt.BookId)
	for _, match := range resp {
		vol := PartialCanvases{
			directory: "",
			Title:     "",
			Canvases:  make([]string, 0, len(match.ImgInfos)),
		}
		for _, m := range match.ImgInfos {
			imgUrl := ossHost + m.BookPath + "/" + m.Fname
			vol.Canvases = append(vol.Canvases, imgUrl)
		}
		volumes = append(volumes, vol)
	}
	return
}

func (p *Korea) getCanvases(sUrl string, jar *cookiejar.Jar) (canvases []string, err error) {
	//TODO implement me
	panic("implement me")
}

func (p *Korea) getBody(sUrl string, jar *cookiejar.Jar) ([]byte, error) {
	//TODO implement me
	panic("implement me")
}

func (p *Korea) postBody(sUrl string, d []byte) ([]byte, error) {
	//TODO implement me
	panic("implement me")
}
