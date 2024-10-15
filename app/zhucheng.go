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
	"strconv"
	"sync"
)

type ZhuCheng struct {
	dt *DownloadTask
}

func (p *ZhuCheng) Init(iTask int, sUrl string) (msg string, err error) {
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

func (p *ZhuCheng) getBookId(sUrl string) (bookId string) {
	if m := regexp.MustCompile(`&id=(\d+)`).FindStringSubmatch(sUrl); m != nil {
		bookId = m[1]
	}
	return bookId
}

func (p *ZhuCheng) download() (msg string, err error) {
	name := util.GenNumberSorted(p.dt.Index)
	log.Printf("Get %s  %s\n", name, p.dt.Url)
	respVolume, err := p.getVolumes(p.dt.BookId, p.dt.Jar)
	if err != nil {
		fmt.Println(err)
		return "getVolumes", err
	}
	p.dt.SavePath = CreateDirectory("zhucheng", p.dt.BookId, "")
	sizeVol := len(respVolume)
	for i, vol := range respVolume {
		if !config.VolumeRange(i) {
			continue
		}
		if sizeVol == 1 {
			p.dt.SavePath = CreateDirectory("zhucheng", p.dt.BookId, "")
		} else {
			vid := util.GenNumberSorted(i + 1)
			p.dt.SavePath = CreateDirectory("zhucheng", p.dt.BookId, vid)
		}

		canvases, err := p.getCanvases(vol, p.dt.Jar)
		if err != nil || canvases == nil {
			fmt.Println(err)
			continue
		}
		log.Printf(" %d/%d volume, %d pages \n", i+1, sizeVol, len(canvases))
		p.do(canvases)
	}
	return msg, err
}

func (p *ZhuCheng) do(imgUrls []string) (msg string, err error) {
	if imgUrls == nil {
		return
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
	return "", err
}

func (p *ZhuCheng) getVolumes(bookId string, jar *cookiejar.Jar) (volumes []string, err error) {
	hostUrl := p.dt.UrlParsed.Scheme + "://" + p.dt.UrlParsed.Host
	apiUrl := hostUrl + "/index.php?ac=catalog&id=" + bookId
	bs, err := getBody(apiUrl, jar)
	if err != nil {
		return
	}

	//取册数
	matches := regexp.MustCompile(`href="./reader.php([^"]+?)"`).FindAllStringSubmatch(string(bs), -1)
	if matches == nil {
		return
	}
	ids := make([]string, 0, len(matches))
	for _, match := range matches {
		ids = append(ids, match[1])
	}
	volumes = make([]string, 0, len(ids))
	for _, v := range ids {
		sUrl := hostUrl + "/reader.php" + v
		volumes = append(volumes, sUrl)
	}
	return volumes, nil
}

func (p *ZhuCheng) getCanvases(sUrl string, jar *cookiejar.Jar) (canvases []string, err error) {
	bs, err := getBody(sUrl, jar)
	if err != nil {
		return
	}
	bid, err := p.getBid(bs)
	cid, err := p.getCID(bs)
	ext, err := p.getImgType(bs)
	pageSize, err := p.getPageNumber(bs)
	hostUrl := p.dt.UrlParsed.Scheme + "://" + p.dt.UrlParsed.Host + "/images/book/" + bid + "/" + cid + "/"
	for i := 1; i <= pageSize; i++ {
		imgUrl := hostUrl + fmt.Sprintf("%d", i) + ext
		canvases = append(canvases, imgUrl)
	}
	return canvases, err
}

func (p *ZhuCheng) getBody(sUrl string, jar *cookiejar.Jar) ([]byte, error) {
	//TODO implement me
	panic("implement me")
}

func (p *ZhuCheng) postBody(sUrl string, d []byte) ([]byte, error) {
	//TODO implement me
	panic("implement me")
}

func (p *ZhuCheng) getBid(bs []byte) (string, error) {
	match := regexp.MustCompile(`var\s+BID\s+=\s+'([A-z0-9]+)'`).FindSubmatch(bs)
	if match != nil {
		return string(match[1]), nil
	}
	return "", errors.New("not found bid")
}

func (p *ZhuCheng) getCID(bs []byte) (string, error) {
	match := regexp.MustCompile(`var\s+CID\s+=\s+'([A-z0-9]+)'`).FindSubmatch(bs)
	if match != nil {
		return string(match[1]), nil
	}
	return "", errors.New("not found cid")
}
func (p *ZhuCheng) getImgType(bs []byte) (string, error) {
	match := regexp.MustCompile(`var\s+imgtype\s+=\s+'([A-z.]+)'`).FindSubmatch(bs)
	if match != nil {
		return string(match[1]), nil
	}
	return "", errors.New("not found ImgType")
}

func (p *ZhuCheng) getPageNumber(bs []byte) (int, error) {
	match := regexp.MustCompile(`var\s+PAGES\s+=\s+([0-9]+)`).FindSubmatch(bs)
	if match != nil {
		size, _ := strconv.Atoi(string(match[1]))
		return size, nil
	}
	return 0, errors.New("not found PAGES")
}
