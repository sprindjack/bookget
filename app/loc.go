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
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"sync"
)

type Loc struct {
	dt         *DownloadTask
	xmlContent []byte
}
type LocManifestsJson struct {
	Resources []struct {
		Caption string           `json:"caption"`
		Files   [][]LocImageFile `json:"files"`
		Image   string           `json:"image"`
		Url     string           `json:"url"`
	} `json:"resources"`
}
type LocImageFile struct {
	Height   *int   `json:"height"`
	Levels   int    `json:"levels"`
	Mimetype string `json:"mimetype"`
	Url      string `json:"url"`
	Width    *int   `json:"width"`
	Info     string `json:"info,omitempty"`
	Size     int    `json:"size,omitempty"`
}

func (r *Loc) Init(iTask int, sUrl string) (msg string, err error) {
	r.dt = new(DownloadTask)
	r.dt.UrlParsed, err = url.Parse(sUrl)
	r.dt.Url = sUrl
	r.dt.Index = iTask
	r.dt.BookId = r.getBookId(r.dt.Url)
	if r.dt.BookId == "" {
		return "requested URL was not found.", err
	}
	r.dt.Jar, _ = cookiejar.New(nil)
	return r.download()
}

func (r *Loc) getBookId(sUrl string) (bookId string) {
	m := regexp.MustCompile(`item/([A-Za-z0-9]+)`).FindStringSubmatch(sUrl)
	if m != nil {
		bookId = m[1]
	}
	return bookId
}

func (r *Loc) download() (msg string, err error) {
	//for China
	if r.isChinaIP() {
		name := util.GenNumberSorted(r.dt.Index)
		log.Printf("Get %s  %s\n", name, r.dt.Url)
		r.dt.VolumeId = r.dt.BookId
		r.dt.SavePath = config.CreateDirectory(r.dt.Url, r.dt.BookId)
		canvases, err := r.getCanvasesJPG2000(r.dt.Url)
		if err != nil || canvases == nil {
			return "requested URL was not found.", err
		}
		log.Printf(" %d pages \n", len(canvases))
		config.Conf.FileExt = ".jp2" //强制jpg2000
		return r.do(canvases)
	}

	//for other
	apiUrl := fmt.Sprintf("https://www.loc.gov/item/%s/?fo=json", r.dt.BookId)
	r.xmlContent, err = r.getBody(apiUrl, r.dt.Jar)
	if err != nil || r.xmlContent == nil {
		return "requested URL was not found.", err
	}
	name := util.GenNumberSorted(r.dt.Index)
	log.Printf("Get %s  %s\n", name, r.dt.Url)

	respVolume, err := r.getVolumes(r.dt.Url, r.dt.Jar)
	if err != nil {
		fmt.Println(err)
		return "getVolumes", err
	}
	for i, vol := range respVolume {
		if config.Conf.Volume > 0 && config.Conf.Volume != i+1 {
			continue
		}
		vid := util.GenNumberSorted(i + 1)
		r.dt.VolumeId = r.dt.BookId + "_vol." + vid
		r.dt.SavePath = config.CreateDirectory(r.dt.Url, r.dt.VolumeId)
		canvases, err := r.getCanvases(vol, r.dt.Jar)
		if err != nil || canvases == nil {
			fmt.Println(err)
			continue
		}
		log.Printf(" %d/%d volume, %d pages \n", i+1, len(respVolume), len(canvases))
		r.do(canvases)
	}
	return "", nil
}

func (r *Loc) do(imgUrls []string) (msg string, err error) {
	if imgUrls == nil {
		return
	}
	fmt.Println()
	referer := url.QueryEscape(r.dt.Url)
	size := len(imgUrls)

	var wg sync.WaitGroup
	q := QueueNew(int(config.Conf.Threads))
	for i, uri := range imgUrls {
		if uri == "" || !config.PageRange(i, size) {
			continue
		}
		sortId := util.GenNumberSorted(i + 1)
		filename := sortId + config.Conf.FileExt
		dest := config.GetDestPath(r.dt.Url, r.dt.VolumeId, filename)
		if FileExist(dest) {
			continue
		}
		imgUrl := uri
		log.Printf("Get %d/%d, URL: %s\n", i+1, size, imgUrl)
		wg.Add(1)
		q.Go(func() {
			defer wg.Done()
			ctx := context.Background()
			opts := gohttp.Options{
				DestFile:    dest,
				Overwrite:   false,
				Concurrency: 1,
				CookieFile:  config.Conf.CookieFile,
				CookieJar:   r.dt.Jar,
				Headers: map[string]interface{}{
					"User-Agent": config.Conf.UserAgent,
					"Referer":    referer,
				},
			}
			for k := 0; k < config.Conf.Retry; k++ {
				_, err := gohttp.FastGet(ctx, imgUrl, opts)
				if err == nil {
					break
				}
			}
			util.PrintSleepTime(config.Conf.Speed)
			fmt.Println()
		})
	}
	wg.Wait()
	fmt.Println()
	return "", err
}

func (r *Loc) getVolumes(sUrl string, jar *cookiejar.Jar) (volumes []string, err error) {
	var manifests = new(LocManifestsJson)
	if err = json.Unmarshal(r.xmlContent, manifests); err != nil {
		log.Printf("json.Unmarshal failed: %s\n", err)
		return
	}
	//一本书有N卷
	for _, resource := range manifests.Resources {
		volumes = append(volumes, resource.Url)
	}
	return volumes, nil
}

func (r *Loc) getCanvases(sUrl string, jar *cookiejar.Jar) (canvases []string, err error) {
	var manifests = new(LocManifestsJson)
	if err = json.Unmarshal(r.xmlContent, manifests); err != nil {
		log.Printf("json.Unmarshal failed: %s\n", err)
		return
	}
	newWidth := ""
	//限制图片最大宽度
	if config.Conf.FullImageWidth > 2400 {
		newWidth = "full/pct:100/"
	} else if config.Conf.FullImageWidth >= 1000 {
		newWidth = fmt.Sprintf("full/%d,/", config.Conf.FullImageWidth)
	}
	for _, resource := range manifests.Resources {
		if resource.Url != sUrl {
			continue
		}
		for _, file := range resource.Files {
			//每页有6种下载方式
			imgUrl, ok := r.getImagePage(file, newWidth)
			if ok {
				canvases = append(canvases, imgUrl)
			}
		}
	}
	return canvases, nil
}
func (r *Loc) getBody(apiUrl string, jar *cookiejar.Jar) ([]byte, error) {
	referer := r.dt.Url
	ctx := context.Background()
	cli := gohttp.NewClient(ctx, gohttp.Options{
		CookieFile: config.Conf.CookieFile,
		CookieJar:  jar,
		Headers: map[string]interface{}{
			"User-Agent": config.Conf.UserAgent,
			"Referer":    referer,
			"authority":  "www.loc.gov",
			"origin":     "https://www.loc.gov",
		},
	})
	resp, err := cli.Get(apiUrl)
	if err != nil {
		return nil, err
	}
	bs, _ := resp.GetBody()
	if resp.GetStatusCode() != http.StatusOK {
		return nil, errors.New(fmt.Sprintf("ErrCode:%d, %s", resp.GetStatusCode(), resp.GetReasonPhrase()))
	}
	return bs, nil
}
func (r *Loc) getImagePage(fileUrls []LocImageFile, newWidth string) (downloadUrl string, ok bool) {
	for _, f := range fileUrls {
		if config.Conf.FileExt == ".jpg" && f.Mimetype == "image/jpeg" {
			if strings.Contains(f.Url, "full/pct:100/") {
				if newWidth != "" && newWidth != "full/pct:100/" {
					downloadUrl = strings.Replace(f.Url, "full/pct:100/", newWidth, 1)
				} else {
					downloadUrl = f.Url
				}
				ok = true
				break
			}
		} else if f.Mimetype != "image/jpeg" {
			downloadUrl = f.Url
			//downloadUrl = strings.Replace(f.Url, "https://tile.loc.gov/storage-services/", "http://140.147.239.202/", 1)
			ok = true
			break
		}
	}
	return
}

func (r *Loc) isChinaIP() bool {
	ctx := context.Background()
	cli := gohttp.NewClient(ctx, gohttp.Options{
		CookieFile: config.Conf.CookieFile,
		CookieJar:  r.dt.Jar,
		Headers: map[string]interface{}{
			"User-Agent": config.Conf.UserAgent,
			"Referer":    "http://ip-api.com/",
		},
	})
	resp, err := cli.Get("http://ip-api.com/json/?lang=zh-CN")
	if err != nil {
		return false
	}
	bs, _ := resp.GetBody()
	text := string(bs)
	if strings.Contains(text, "\"country\":\"中国\"") {
		return true
	}
	return false
}

func (r *Loc) getCanvasesJPG2000(sUrl string) (canvases []string, err error) {
	d := []byte("pid=" + sUrl + "&filetype=jp2&cdn=ncdn&submit=%E8%8E%B7%E5%8F%96")
	ctx := context.Background()
	cli := gohttp.NewClient(ctx, gohttp.Options{
		CookieFile: config.Conf.CookieFile,
		CookieJar:  r.dt.Jar,
		Headers: map[string]interface{}{
			"User-Agent":   config.Conf.UserAgent,
			"Content-Type": "application/x-www-form-urlencoded",
			"Referer":      "https://ok.daoing.com/mggh/index.php?from=bookget",
		},
		Body: d,
	})
	resp, err := cli.Post("https://ok.daoing.com/mggh/index.php")
	if err != nil {
		return nil, err
	}
	bs, _ := resp.GetBody()
	matches := regexp.MustCompile(`http://140.147.239.202/([A-z0-9_/.-])+`).FindAllStringSubmatch(string(bs), -1)
	if matches == nil {
		return
	}
	for _, match := range matches {
		canvases = append(canvases, match[0])
	}
	return canvases, nil
}
