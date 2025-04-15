package app

import (
	"bookget/config"
	"bookget/pkg/gohttp"
	"bookget/pkg/util"
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

type Usthk struct {
	dt *DownloadTask
}
type UsthkResponseFiles struct {
	FileList []string `json:"file_list"`
}

func (p *Usthk) Init(iTask int, sUrl string) (msg string, err error) {
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

func (p *Usthk) getBookId(sUrl string) (bookId string) {
	m := regexp.MustCompile(`bib/([A-z0-9_-]+)`).FindStringSubmatch(sUrl)
	if m != nil {
		bookId = m[1]
	}
	return bookId
}

func (p *Usthk) download() (msg string, err error) {
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

		canvases, err := p.getCanvases(vol, p.dt.Jar)
		if err != nil || canvases == nil {
			fmt.Println(err)
			continue
		}
		fmt.Println()
		log.Printf(" %d/%d volume, %d pages \n", i+1, sizeVol, len(canvases))
		p.do(canvases)
	}
	return "", nil
}

func (p *Usthk) do(imgUrls []string) (msg string, err error) {
	if imgUrls == nil {
		return "图片URLs为空", errors.New("imgUrls is nil")
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
	return "", nil
}

func (p *Usthk) getVolumes(sUrl string, jar *cookiejar.Jar) (volumes []string, err error) {
	volumes = append(volumes, sUrl)
	return volumes, nil
}

func (p *Usthk) getCanvases(sUrl string, jar *cookiejar.Jar) ([]string, error) {
	bs, err := p.getBody(sUrl, jar)
	if err != nil {
		return nil, err
	}
	//view_book('6/o/b1129168/ebook'
	matches := regexp.MustCompile(`view_book\(["'](\S+)["']`).FindAllStringSubmatch(string(bs), -1)
	if matches == nil {
		return nil, fmt.Errorf("Canvas not found")
	}

	canvases := make([]string, 0, len(matches))
	for _, m := range matches {
		sPath := m[1]
		apiUrl := fmt.Sprintf("https://%s/bookreader/getfilelist.php?path=%s", p.dt.UrlParsed.Host, sPath)
		bs, err = p.getBody(apiUrl, p.dt.Jar)
		if err != nil {
			break
		}
		respFiles := new(UsthkResponseFiles)
		if err = json.Unmarshal(bs, respFiles); err != nil {
			log.Printf("json.Unmarshal failed: %s\n", err)
			break
		}
		//imgUrls := make([]string, 0, len(result.FileList))
		for _, v := range respFiles.FileList {
			imgUrl := fmt.Sprintf("https://%s/obj/%s/%s", p.dt.UrlParsed.Host, sPath, v)
			canvases = append(canvases, imgUrl)
		}
	}
	return canvases, nil
}

func (p *Usthk) getBody(sUrl string, jar *cookiejar.Jar) ([]byte, error) {
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

func (p *Usthk) postBody(sUrl string, d []byte) ([]byte, error) {
	//TODO implement me
	panic("implement me")
}
