package app

import (
	"bookget/config"
	"bookget/lib/gohttp"
	"bookget/lib/util"
	"context"
	"encoding/base64"
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

type NclTw struct {
	dt                       *DownloadTask
	requestVerificationToken string
	imageKey                 string
	enCodeType               int
	body                     []byte
}
type NclTwResponseToken struct {
	Token string `json:"token"`
}

func (r *NclTw) Init(iTask int, sUrl string) (msg string, err error) {
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

func (r *NclTw) getBookId(sUrl string) (bookId string) {
	m := regexp.MustCompile(`item=([A-Za-z0-9]+)`).FindStringSubmatch(sUrl)
	if m != nil {
		bookId = m[1]
	}
	return bookId
}

func (r *NclTw) download() (msg string, err error) {
	name := util.GenNumberSorted(r.dt.Index)
	log.Printf("Get %s  %s\n", name, r.dt.Url)
	r.dt.SavePath = config.CreateDirectory(r.dt.Url, r.dt.BookId)
	canvases, err := r.getCanvases(r.dt.Url, r.dt.Jar)
	if err != nil || canvases == nil {
		return "requested URL was not found.", err
	}
	log.Printf(" %d pages \n", len(canvases))
	r.do(canvases)
	return "", err
}

func (r *NclTw) do(imgUrls []string) (msg string, err error) {
	if imgUrls == nil {
		return
	}
	fmt.Println()
	size := len(imgUrls)
	ctx := context.Background()
	for i, uri := range imgUrls {
		if uri == "" || !config.PageRange(i, size) {
			continue
		}
		sortId := util.GenNumberSorted(i + 1)
		filename := sortId + config.Conf.FileExt
		dest := config.GetDestPath(r.dt.Url, r.dt.BookId, filename)
		if FileExist(dest) {
			continue
		}
		log.Printf("Get %d/%d page, URL: %s\n", i+1, size, uri)
		token, err := r.getToken()
		if err != nil {
			fmt.Println(err)
			break
		}
		imgUrl := uri + "&token=" + token
		opts := gohttp.Options{
			DestFile:    dest,
			Overwrite:   false,
			Concurrency: 1,
			CookieFile:  config.Conf.CookieFile,
			CookieJar:   r.dt.Jar,
			Headers: map[string]interface{}{
				"User-Agent": config.Conf.UserAgent,
				"Referer":    r.dt.Url,
				"authority":  "rbook.ncl.edu.tw",
				"origin":     "https://rbook.ncl.edu.tw",
			},
		}
		_, err = gohttp.FastGet(ctx, imgUrl, opts)
		if err != nil {
			fmt.Println(err)
			continue
		}
		util.PrintSleepTime(config.Conf.Speed)
	}
	fmt.Println()
	return "", err
}

func (r *NclTw) getVolumes(sUrl string, jar *cookiejar.Jar) (volumes []string, err error) {
	//TODO implement me
	panic("implement me")
}

func (r *NclTw) getCanvases(sUrl string, jar *cookiejar.Jar) (canvases []string, err error) {
	r.body, err = r.getBody(sUrl, jar)
	if err != nil {
		return nil, err
	}
	//取页数
	matches := regexp.MustCompile(`name="ImageCheck" value="([^>]+)"`).FindAllSubmatch(r.body, -1)
	if matches == nil {
		return
	}
	canvases = make([]string, 0, len(matches))
	for _, v := range matches {
		href := strings.Replace(string(v[1]), "&amp;", "&", -1)
		imgUrl := r.dt.UrlParsed.Scheme + "://" + r.dt.UrlParsed.Host + href
		canvases = append(canvases, imgUrl)
	}
	return canvases, nil
}

func (r *NclTw) getBody(apiUrl string, jar *cookiejar.Jar) ([]byte, error) {
	ctx := context.Background()
	cli := gohttp.NewClient(ctx, gohttp.Options{
		CookieFile: config.Conf.CookieFile,
		CookieJar:  jar,
		Headers: map[string]interface{}{
			"User-Agent": config.Conf.UserAgent,
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
func (r *NclTw) getRequestToken() string {
	//取请求token
	// <input name="__RequestVerificationToken" type="hidden" value="ayk-lqrk1RrbJb1xB6FM2-cALjxxYUHAapQoPBSLuVQFSmJQQ-DQSAhzcE7IciaEw3GZBs_irf71OGFXZxUctQeJaSBfU2V1TvI5vijRjMA1" />
	m := regexp.MustCompile(`name="__RequestVerificationToken(?:.+)value="(\S+)"`).FindSubmatch(r.body)
	if m == nil {
		return ""
	}
	r.requestVerificationToken = string(m[1])
	return r.requestVerificationToken
}

func (r *NclTw) getToken() (token string, err error) {
	apiUrl := "https://rbook.ncl.edu.tw/NCLSearch/Watermark/getToken"
	requestVerificationToken := r.getRequestToken()
	imgKey, err := r.getImageKey()
	ctx := context.Background()
	cli := gohttp.NewClient(ctx, gohttp.Options{
		CookieFile: config.Conf.CookieFile,
		CookieJar:  r.dt.Jar,
		Headers: map[string]interface{}{
			"User-Agent":       config.Conf.UserAgent,
			"authority":        "rbook.ncl.edu.tw",
			"content-type":     "application/x-www-form-urlencoded; charset=UTF-8",
			"x-requested-with": "XMLHttpRequest",
			"origin":           "https://rbook.ncl.edu.tw",
			"referer":          r.dt.Url,
		},
		FormParams: map[string]interface{}{
			imgKey: requestVerificationToken,
		},
	})
	resp, err := cli.Post(apiUrl)
	if err != nil {
		return "", err
	}
	bs, _ := resp.GetBody()
	resToken := new(NclTwResponseToken)
	if err = json.Unmarshal(bs, resToken); err != nil {
		log.Printf("json.Unmarshal failed: %s\n", err)
		return "", err
	}
	m := strings.Split(resToken.Token, ":")
	if len(m) != 2 {
		return "", err
	}
	token = m[0]
	r.imageKey = m[1]
	return token, nil
}

func (r *NclTw) getImageKey() (key string, err error) {
	if r.imageKey == "" {
		for i := 0; i < 100; i++ {
			r.body, err = r.getBody(r.dt.Url, r.dt.Jar)
			if err != nil {
				return "", errors.New("not found")
			}
			//base64
			m := regexp.MustCompile(`btoa\("([A-z0-9-]+)"\)`).FindSubmatch(r.body)
			if m != nil {
				r.imageKey = string(m[1])
				r.enCodeType = 1
				break
			}
			//haveaniceday
			m = regexp.MustCompile(`var i="([A-z0-9-]+)"`).FindSubmatch(r.body)
			if m != nil {
				r.imageKey = string(m[1])
				r.enCodeType = 2
				break
			}
			//Array[3][$lenght]
			m = regexp.MustCompile(`var u="([A-z0-9-]+)"`).FindSubmatch(r.body)
			if m != nil {
				r.imageKey = string(m[1])
				r.enCodeType = 3
				break
			}
		}
	}
	switch r.enCodeType {
	case 1:
		key = base64.StdEncoding.EncodeToString([]byte(r.imageKey))
		break
	case 2:
		key = r.getImageKey2(r.imageKey)
		break
	case 3:
		key = r.getImageKey3(r.imageKey)
		break
	}
	//fmt.Printf("\nenCodeType=%d\n", r.enCodeType)
	return key, nil
}

func (r *NclTw) getImageKey2(key string) string {
	var sss = "haveaniceday"
	var out = ""
	var f = 0
	size := len(key)
	for i := 0; i < size; i++ {
		var e = r.charCodeAt(key, i)
		k := f % len(sss)
		str := sss[k : k+1]
		o := r.charCodeAt(str, 0)
		s := fmt.Sprintf("%c", e^o)
		out += s
		f++
	}
	return out
}

func (r *NclTw) getImageKey3(key string) string {
	size := len(key)
	var max = 3
	mArr := make([][]string, max)
	i := 0
	c := 1
	var out string
	for j := 0; j < size; j++ {
		s := key[j : j+1]
		mArr[i] = append(mArr[i], s)
		if i == 0 {
			c = 1
		} else if i == (max - 1) {
			c = -1
		}
		i += c
	}
	for _, item := range mArr {
		for _, s := range item {
			if s != "" {
				out += s
			}
		}
	}
	return out
}

func (r *NclTw) charCodeAt(text string, n int) rune {
	i := 0
	for _, code := range text {
		if i == n {
			return code
		}
		i++
	}
	return 0
}
