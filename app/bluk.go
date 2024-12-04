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

type Bluk struct {
	dt *DownloadTask
}

func (p *Bluk) Init(iTask int, sUrl string) (msg string, err error) {
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

func (p *Bluk) getBookId(sUrl string) (bookId string) {
	m := regexp.MustCompile(`Viewer.aspx\?ref=([\S]+)`).FindStringSubmatch(sUrl)
	if m != nil {
		bookId = m[1]
	}
	return bookId
}

func (p *Bluk) download() (msg string, err error) {
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
		log.Printf(" %d/%d volume, %d pages \n", i+1, sizeVol, len(canvases))
		p.do(canvases)
	}
	return "", nil
}

func (p *Bluk) do(imgUrls []string) (msg string, err error) {
	if config.Conf.UseDziRs {
		p.doDezoomifyRs(imgUrls)
	} else {
		p.doNormal(imgUrls)
	}
	return "", err
}

func (p *Bluk) getVolumes(sUrl string, jar *cookiejar.Jar) (volumes []string, err error) {
	volumes = append(volumes, sUrl)
	return volumes, nil
}

func (p *Bluk) getCanvases(sUrl string, jar *cookiejar.Jar) (canvases []string, err error) {
	bs, err := p.getBody(sUrl, jar)
	if err != nil {
		return
	}
	//        <input type="hidden" name="PageList" id="PageList" value="##||or_6814!1_fs001r||or_6814!1_fs001v||or_6814!1_f001r||or_6814!1_f001v||or_6814!1_f002r||or_6814!1_f002v||or_6814!1_f003r||or_6814!1_f003v||or_6814!1_f004r||or_6814!1_f004v||or_6814!1_f005r||or_6814!1_f005v||or_6814!1_f006r||or_6814!1_f006v||or_6814!1_f007r||or_6814!1_f007v||or_6814!1_f008r||or_6814!1_f008v||or_6814!1_f009r||or_6814!1_f009v||or_6814!1_f010r||or_6814!1_f010v||or_6814!1_f011r||or_6814!1_f011v||or_6814!1_f012r||or_6814!1_f012v||or_6814!1_f013r||or_6814!1_f013v||or_6814!1_f014r||or_6814!1_f014v||or_6814!1_f015r||or_6814!1_f015v||or_6814!1_f016r||or_6814!1_f016v||or_6814!1_f017r||or_6814!1_f017v||or_6814!1_f018r||or_6814!1_f018v||or_6814!1_f019r||or_6814!1_f019v||or_6814!1_f020r||or_6814!1_f020v||or_6814!1_f021r||or_6814!1_f021v||or_6814!1_f022r||or_6814!1_f022v||or_6814!1_f023r||or_6814!1_f023v||or_6814!1_f024r||or_6814!1_f024v||or_6814!1_f025r||or_6814!1_f025v||or_6814!1_f026r||or_6814!1_f026v||or_6814!1_f027r||or_6814!1_f027v||or_6814!1_f028r||or_6814!1_f028v||or_6814!1_f029r||or_6814!1_f029v||or_6814!1_f030r||or_6814!1_f030v||or_6814!1_f031r||or_6814!1_f031v||or_6814!1_f032r||or_6814!1_f032v||or_6814!1_f033r||or_6814!1_f033v||or_6814!1_f034r||or_6814!1_f034v||or_6814!1_f035r||or_6814!1_f035v||or_6814!1_f036r||or_6814!1_f036v||or_6814!1_f037r||or_6814!1_f037v||##||or_6814!1_fblefv||or_6814!1_fbrigr||##||or_6814!1_fblefr||or_6814!1_fbrigv||or_6814!1_fbspi" />
	match := regexp.MustCompile(`id="PageList"[\s]+value=["']([\S]+)["']`).FindStringSubmatch(string(bs))
	if match == nil {
		return
	}
	m := strings.Split(match[1], "||")
	if len(m) == 0 {
		return
	}
	size := len(m)
	canvases = make([]string, 0, size)
	for _, id := range m {
		if id == "##" {
			continue
		}
		//dezoomify-rs URL
		dziUrl := fmt.Sprintf("http://www.bl.uk/manuscripts/Proxy.ashx?view=%s.xml", id)
		canvases = append(canvases, dziUrl)

	}
	return canvases, nil

}

func (p *Bluk) getBody(sUrl string, jar *cookiejar.Jar) ([]byte, error) {
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

func (p *Bluk) postBody(sUrl string, d []byte) ([]byte, error) {
	//TODO implement me
	panic("implement me")
}

func (p *Bluk) doDezoomifyRs(iiifUrls []string) bool {
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

func (p *Bluk) doNormal(imgUrls []string) bool {
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
