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

func (r Loc) Init(iTask int, sUrl string) (msg string, err error) {
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

func (r Loc) getBookId(sUrl string) (bookId string) {
	m := regexp.MustCompile(`item/([A-Za-z0-9]+)`).FindStringSubmatch(sUrl)
	if m != nil {
		bookId = m[1]
	}
	return bookId
}

func (r Loc) download() (msg string, err error) {
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

func (r Loc) do(imgUrls []string) (msg string, err error) {
	if imgUrls == nil {
		return
	}
	fmt.Println()
	referer := url.QueryEscape(r.dt.Url)
	size := len(imgUrls)
	ctx := context.Background()
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
		log.Printf("Get %d/%d page, URL: %s\n", i+1, size, uri)
		opts := gohttp.Options{
			DestFile:    dest,
			Overwrite:   false,
			Concurrency: config.Conf.Threads,
			CookieFile:  config.Conf.CookieFile,
			CookieJar:   r.dt.Jar,
			Headers: map[string]interface{}{
				"User-Agent": config.Conf.UserAgent,
				"Referer":    referer,
			},
		}
		_, err := gohttp.FastGet(ctx, uri, opts)
		if err != nil {
			fmt.Println(err)
			continue
		}
		//util.PrintSleepTime(config.Conf.Speed)
	}
	fmt.Println()
	return "", err
}

func (r Loc) getVolumes(sUrl string, jar *cookiejar.Jar) (volumes []string, err error) {
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

func (r Loc) getCanvases(sUrl string, jar *cookiejar.Jar) (canvases []string, err error) {
	var manifests = new(LocManifestsJson)
	if err = json.Unmarshal(r.xmlContent, manifests); err != nil {
		log.Printf("json.Unmarshal failed: %s\n", err)
		return
	}
	newWidth := ""
	//限制图片最大宽度
	if config.Conf.FullImageWidth > 6400 {
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
func (r Loc) getBody(apiUrl string, jar *cookiejar.Jar) ([]byte, error) {
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
func (r Loc) getImagePage(fileUrls []LocImageFile, newWidth string) (downloadUrl string, ok bool) {
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