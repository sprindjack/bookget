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
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type DziCnLib struct {
	dt         *DownloadTask
	ServerHost string
	Extention  string
}

type ResponseServerBase struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Title      string   `json:"title"`
		ServerBase string   `json:"serverBase"`
		Images     []string `json:"images"`
	} `json:"data"`
}

type Item struct {
	Extension   string `json:"extension"`
	Height      int    `json:"height"`
	Resolutions int    `json:"resolutions"`
	TileSize    struct {
		H int `json:"h"`
		W int `json:"w"`
	} `json:"tile_size"`
	TileSize2 struct {
		Height int `json:"height"`
		Width  int `json:"width"`
	} `json:"tileSize"`
	Width int `json:"width"`
}

// 自定义一个排序类型
type strs []string

func (s strs) Len() int           { return len(s) }
func (s strs) Less(i, j int) bool { return s[i] < s[j] }
func (s strs) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

func (r DziCnLib) Init(iTask int, sUrl string) (msg string, err error) {
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

func (r DziCnLib) getBookId(sUrl string) (bookId string) {
	m := regexp.MustCompile(`(?i)bookid=([A-z0-9_-]+)`).FindStringSubmatch(sUrl)
	if m != nil {
		return m[1]
	}
	m = regexp.MustCompile(`(?i)id=([A-z0-9_-]+)`).FindStringSubmatch(sUrl)
	if m != nil {
		bookId = m[1]
	}
	return bookId
}

func (r DziCnLib) download() (msg string, err error) {
	name := util.GenNumberSorted(r.dt.Index)
	log.Printf("Get %s  %s\n", name, r.dt.Url)

	r.ServerHost = r.getServerUri(r.dt.BookId, r.dt.UrlParsed)
	if r.ServerHost == "" {
		return "requested URL was not found.", err
	}
	r.dt.SavePath = config.CreateDirectory(r.dt.Url, r.dt.BookId)
	canvases, err := r.getCanvases(r.dt.Url, r.dt.Jar)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	log.Printf(" %d pages \n", len(canvases))
	return r.do(canvases)
}

func (r DziCnLib) do(dziUrls []string) (msg string, err error) {
	if dziUrls == nil {
		return "", err
	}
	storePath := r.dt.SavePath + string(os.PathSeparator)
	referer := url.QueryEscape(r.dt.Url)
	args := []string{"--dezoomer=deepzoom",
		"-H", "Origin:" + referer,
		"-H", "Referer:" + referer,
		"-H", "User-Agent:" + config.Conf.UserAgent,
	}
	size := len(dziUrls)
	for i, val := range dziUrls {
		if !config.PageRange(i, size) {
			continue
		}
		inputUri := storePath + val
		outfile := storePath + util.GenNumberSorted(i+1) + r.Extention
		if FileExist(outfile) {
			continue
		}
		if ret := util.StartProcess(inputUri, outfile, args); ret == true {
			os.Remove(inputUri)
		}
		util.PrintSleepTime(config.Conf.Speed)
	}
	return "", err
}

func (r DziCnLib) getVolumes(sUrl string, jar *cookiejar.Jar) (volumes []string, err error) {
	//TODO implement me
	panic("implement me")
}

func (r DziCnLib) getCanvases(sUrl string, jar *cookiejar.Jar) (canvases []string, err error) {
	apiUrl := fmt.Sprintf("%s/tiles/infos.json", r.ServerHost)
	bs, err := r.getBody(apiUrl, jar)
	if err != nil {
		return
	}
	type ResponseBody struct {
		Tiles map[string]Item `json:"tiles"`
	}
	var result ResponseBody
	if err = json.Unmarshal(bs, &result); err != nil {
		return
	}
	if result.Tiles == nil {
		return
	}

	text := `{
    "Image": {
    "xmlns":    "http://schemas.microsoft.com/deepzoom/2009",
    "Url":      "%s",
    "Format":   "%s",
    "Overlap":  "1", 
	"MaxLevel": "0",
	"Separator": "/",
        "TileSize": "%d",
        "Size": {
            "Height": "%d",
            "Width":  "%d"
        }
    }
}
`

	canvases = make([]string, 0, len(result.Tiles))
	for key, item := range result.Tiles {
		sortId := fmt.Sprintf("%s.json", key)
		dest := config.GetDestPath(r.dt.Url, r.dt.BookId, sortId)
		serverUrl := fmt.Sprintf("%s/tiles/%s/", r.ServerHost, key)
		if r.Extention == "" {
			r.Extention = "." + strings.ToLower(item.Extension)
		}
		jsonText := ""
		if item.TileSize.W == 0 {
			jsonText = fmt.Sprintf(text, serverUrl, item.Extension, item.TileSize2.Width, item.Height, item.Width)
		} else {
			jsonText = fmt.Sprintf(text, serverUrl, item.Extension, item.TileSize.W, item.Height, item.Width)
		}
		util.FileWrite([]byte(jsonText), dest)

		canvases = append(canvases, sortId)
	}
	sort.Sort(strs(canvases))
	return canvases, nil
}

func (r DziCnLib) getServerUri(bookId string, u *url.URL) string {
	m := regexp.MustCompile(`(?i)typeId=([A-z0-9_-]+)`).FindStringSubmatch(u.RawQuery)
	typeId := 80
	if m != nil {
		typeId, _ = strconv.Atoi(m[1])
	}
	apiUrl := fmt.Sprintf("%s://%s/portal/book/view?bookId=%s&typeId=%d", u.Scheme, u.Host, bookId, typeId)
	bs, err := r.getBody(apiUrl, r.dt.Jar)
	if err != nil {
		return ""
	}
	var respServerBase ResponseServerBase
	if err = json.Unmarshal(bs, &respServerBase); err != nil {
		return ""
	}
	return fmt.Sprintf("%s://%s%s", u.Scheme, u.Host, respServerBase.Data.ServerBase)
}

func (r DziCnLib) getBody(apiUrl string, jar *cookiejar.Jar) ([]byte, error) {
	referer := url.QueryEscape(apiUrl)
	ctx := context.Background()
	cli := gohttp.NewClient(ctx, gohttp.Options{
		CookieFile: config.Conf.CookieFile,
		CookieJar:  jar,
		Headers: map[string]interface{}{
			"User-Agent": config.Conf.UserAgent,
			"Referer":    referer,
		},
	})
	resp, err := cli.Get(apiUrl)
	if err != nil {
		return nil, err
	}
	bs, _ := resp.GetBody()
	if resp.GetStatusCode() == 202 || bs == nil {
		return nil, errors.New(fmt.Sprintf("ErrCode:%d, %s", resp.GetStatusCode(), resp.GetReasonPhrase()))
	}
	return bs, nil
}
