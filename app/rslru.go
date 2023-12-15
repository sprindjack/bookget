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
	"sync"
)

type RslRu struct {
	dt       *DownloadTask
	response RslRuResponse
}
type RslRuResponse struct {
	IsAvailable                     bool        `json:"isAvailable"`
	IsAuthorizationRequired         bool        `json:"isAuthorizationRequired"`
	IsGosuslugiVerificationRequired bool        `json:"isGosuslugiVerificationRequired"`
	Formats                         []string    `json:"formats"`
	PageCount                       int         `json:"pageCount"`
	IsSearchable                    bool        `json:"isSearchable"`
	HasTextLayer                    bool        `json:"hasTextLayer"`
	OwnershipSystem                 string      `json:"ownershipSystem"`
	AccessLevel                     string      `json:"accessLevel"`
	AccessInformationMessage        interface{} `json:"accessInformationMessage"`
	Description                     struct {
		Author  interface{} `json:"author"`
		Title   string      `json:"title"`
		Imprint string      `json:"imprint"`
	} `json:"description"`
	PrintAccess struct {
		IsPrintable             bool `json:"isPrintable"`
		IsPrintableWhenLoggedIn bool `json:"isPrintableWhenLoggedIn"`
	} `json:"printAccess"`
	ViewAccess struct {
		AvailablePdfPages []struct {
			Min int `json:"min"`
			Max int `json:"max"`
		} `json:"availablePdfPages"`
		AvailableEpubPercent    interface{} `json:"availableEpubPercent"`
		PreviewPdfPages         interface{} `json:"previewPdfPages"`
		OutOfPreviewRangeAction interface{} `json:"outOfPreviewRangeAction"`
	} `json:"viewAccess"`
	DownloadAccess struct {
		IsDownloadable      bool          `json:"isDownloadable"`
		DownloadableFormats []interface{} `json:"downloadableFormats"`
		ForbiddenReasonText interface{}   `json:"forbiddenReasonText"`
	} `json:"downloadAccess"`
	HasAudio            bool   `json:"hasAudio"`
	HasWordCoordinates  bool   `json:"hasWordCoordinates"`
	ReadingSessionId    string `json:"readingSessionId"`
	AllowedAccessTokens struct {
		Pdf bool `json:"pdf"`
	} `json:"allowedAccessTokens"`
}

func (r *RslRu) Init(iTask int, sUrl string) (msg string, err error) {
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

func (r *RslRu) getBookId(sUrl string) (bookId string) {
	m := regexp.MustCompile(`/([A-z\d]+)/([A-z\d]+)`).FindStringSubmatch(sUrl)
	if m != nil {
		bookId = m[2]
	}
	return bookId
}

func (r *RslRu) download() (msg string, err error) {
	name := util.GenNumberSorted(r.dt.Index)
	log.Printf("Get %s  %s\n", name, r.dt.Url)

	r.response, err = r.getJsonResponse()
	if err != nil {
		return "requested URL was not found.", err
	}
	vid := r.response.Description.Title
	r.dt.SavePath = CreateDirectory(r.dt.UrlParsed.Host, r.dt.BookId, vid)
	canvases, err := r.getCanvases(r.dt.Url, r.dt.Jar)
	if err != nil || canvases == nil {
		return "requested URL was not found.", err
	}
	log.Printf(" %d pages \n", len(canvases))
	return r.do(canvases)
}

func (r *RslRu) do(canvases []string) (msg string, err error) {
	if canvases == nil {
		return
	}
	fmt.Println()
	referer := r.dt.Url
	size := len(canvases)
	var wg sync.WaitGroup
	q := QueueNew(int(config.Conf.Threads))
	for i, uri := range canvases {
		if uri == "" || !config.PageRange(i, size) {
			continue
		}
		sortId := util.GenNumberSorted(i + 1)
		filename := sortId + config.Conf.FileExt
		dest := r.dt.SavePath + filename
		if FileExist(dest) {
			continue
		}
		imgUrl := uri
		log.Printf("Get %d/%d page, URL: %s\n", i+1, size, imgUrl)
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
			gohttp.FastGet(ctx, imgUrl, opts)
			fmt.Println()
		})
	}
	wg.Wait()
	fmt.Println()
	return "", err
}

func (r *RslRu) getVolumes(sUrl string, jar *cookiejar.Jar) (volumes []string, err error) {
	//TODO implement me
	panic("implement me")
}

func (r *RslRu) getCanvases(sUrl string, jar *cookiejar.Jar) (canvases []string, err error) {
	for i := 1; i <= r.response.PageCount; i++ {
		imgUrl := fmt.Sprintf("https://viewer.rsl.ru/api/v1/document/%s/page/%d", r.dt.BookId, i)
		canvases = append(canvases, imgUrl)
	}
	return canvases, nil
}

func (r *RslRu) getJsonResponse() (resp RslRuResponse, err error) {
	apiUrl := fmt.Sprintf("https://viewer.rsl.ru/api/v1/document/%s/info", r.dt.BookId)
	bs, err := r.getBody(apiUrl, r.dt.Jar)
	if err != nil {
		return
	}
	if err = json.Unmarshal(bs, &resp); err != nil {
		log.Printf("json.Unmarshal failed: %s\n", err)
	}
	return resp, err
}

func (r *RslRu) getBody(apiUrl string, jar *cookiejar.Jar) ([]byte, error) {
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
