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
	"sync"
)

type SillokGoKr struct {
	dt       *DownloadTask
	bookMark string
	index    int
	canvases []SillokGoKrImage
	apiUrl   string
}

type SillokGoKrImage struct {
	KingCode   string `json:"kingCode"`
	ImageId    string `json:"imageId"`
	Previous   string `json:"previous"`
	Firstchild string `json:"firstchild"`
	Title      string `json:"title"`
	PageId     string `json:"pageId"`
	Parent     string `json:"parent"`
	Type       string `json:"type"`
	Level      string `json:"level"`
	Next       string `json:"next"`
	Seq        string `json:"seq"`
}

type SillokGoKrResponse struct {
	TreeList struct {
		List      []SillokGoKrImage `json:"list"`
		ListCount int               `json:"listCount"`
	} `json:"treeList"`
}

type SillokGoKrBook struct {
	Title      string
	Seq        int
	Id         string
	KingCode   string
	Level      int
	Type       string
	Next       string
	Firstchild string
}

type ByImageIdSort []SillokGoKrImage

func (ni ByImageIdSort) Len() int      { return len(ni) }
func (ni ByImageIdSort) Swap(i, j int) { ni[i], ni[j] = ni[j], ni[i] }
func (ni ByImageIdSort) Less(i, j int) bool {
	idA := ni[i].ImageId
	idB := ni[j].ImageId
	a, err1 := strconv.ParseInt(idA[strings.LastIndex(idA, "_"):], 10, 64)
	b, err2 := strconv.ParseInt(idB[strings.LastIndex(idB, "_"):], 10, 64)
	if err1 != nil || err2 != nil {
		return idA < idB
	}
	return a < b
}

func (r *SillokGoKr) Init(iTask int, sUrl string) (msg string, err error) {
	r.dt = new(DownloadTask)
	r.dt.UrlParsed, err = url.Parse(sUrl)
	r.dt.Url = sUrl
	r.dt.Index = iTask
	r.dt.BookId = r.getBookId(r.dt.Url)
	if r.dt.BookId == "" {
		return "requested URL was not found.", err
	}
	r.apiUrl = r.dt.UrlParsed.Scheme + "://" + r.dt.UrlParsed.Host + r.dt.UrlParsed.Path
	r.dt.Jar, _ = cookiejar.New(nil)
	return r.download()
}

func (r *SillokGoKr) getBookId(sUrl string) (bookId string) {
	m := regexp.MustCompile(`levelId=([A-z0-9_-]+)`).FindStringSubmatch(sUrl)
	if m != nil {
		bookId = m[1]
	}
	return bookId
}

func (r *SillokGoKr) download() (msg string, err error) {
	name := util.GenNumberSorted(r.dt.Index)
	log.Printf("Get %s  %s\n", name, r.dt.Url)

	respBook, err := r.getBooks(r.dt.Url, r.dt.Jar)
	if err != nil {
		fmt.Println(err)
		return "getVolumes", err
	}
	log.Printf(" %d books\n", len(respBook))
	for k, book := range respBook {
		volumes, err := r.getVolumesByBookId(book)
		if err != nil {
			fmt.Printf("Error: +%v\n", err)
			continue
		}
		log.Printf("book %d/%d, %d volume \n", k+1, len(respBook), len(volumes))
		for i, v := range volumes {
			if !config.VolumeRange(i) {
				continue
			}
			r.bookMark = ""
			r.index = 0
			r.canvases = make([]SillokGoKrImage, 0, 1000)
			_seq, _ := strconv.Atoi(v.Seq)
			_level, _ := strconv.Atoi(v.Level)
			volume := SillokGoKrBook{
				Title:      v.Title,
				Seq:        _seq,
				Id:         v.Seq,
				KingCode:   v.KingCode,
				Level:      _level,
				Type:       v.Type,
				Next:       v.Next,
				Firstchild: v.Firstchild,
			}
			err = r.getImagesByVolumeId(volume, &r.canvases)
			if err != nil {
				fmt.Printf("Error: +%v\n", err)
				continue
			}
			fmt.Println()

			vid := util.GenNumberSorted(i + 1)
			r.dt.VolumeId = r.dt.UrlParsed.Host + "_book." + book.Id + "." + book.Title + string(os.PathSeparator) + "vol." + vid + "." + v.Title
			r.dt.SavePath = config.Conf.SaveFolder + string(os.PathSeparator) + r.dt.VolumeId
			_ = os.MkdirAll(r.dt.SavePath, os.ModePerm)
			r.do(nil)
		}
	}

	return "", nil
}

func (r *SillokGoKr) do(imgUrls []string) (msg string, err error) {
	if r.canvases == nil {
		return
	}
	fmt.Println()

	canvasesM := make(map[string]SillokGoKrImage)
	for _, v := range r.canvases {
		canvasesM[v.ImageId] = v
	}
	canvases := make([]SillokGoKrImage, 0, len(canvasesM))
	for _, v := range canvasesM {
		canvases = append(canvases, v)
	}
	sort.Sort(ByImageIdSort(canvases))
	referer := r.dt.Url
	size := len(canvases)
	var wg sync.WaitGroup
	q := QueueNew(int(config.Conf.Threads))
	for i, v := range canvases {
		if v.ImageId == "" || !config.PageRange(i, size) {
			continue
		}
		imgUrl := r.imageIdUrl(v.KingCode, v.ImageId)
		sortId := util.GenNumberSorted(i + 1)
		filename := sortId + ".jpg"
		dest := r.dt.SavePath + filename
		if FileExist(dest) {
			continue
		}
		log.Printf("Get %d/%d,  URL: %s\n", i+1, size, imgUrl)

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
		})
		fmt.Println()
	}
	wg.Wait()
	return "", err
}

func (r *SillokGoKr) getBooks(sUrl string, jar *cookiejar.Jar) (books []SillokGoKrBook, err error) {
	bs, err := r.getBody(sUrl, jar)
	if err != nil {
		return
	}
	text := string(bs)
	matches := regexp.MustCompile(`<li\s+id="([A-z0-9_]+)"\s+kingCode="([^"]+)"\s+level="([A-z0-9_]+)"\s+type="([A-z0-9_]+)"\s+next="([A-z0-9_]*)"\s+firstchild="([A-z0-9_]+)">`).FindAllStringSubmatch(text, -1)
	if matches == nil {
		return
	}
	books = make([]SillokGoKrBook, 0, len(matches))
	mTitle := regexp.MustCompile(`<span class="folder(?:[^>]+)><a href=(?:[^>]+)>([^<]+)</a></span>`).FindAllStringSubmatch(text, -1)
	if mTitle == nil {
		return
	}
	for i, m := range matches {
		id, _ := strconv.Atoi(m[1])
		level, _ := strconv.Atoi(m[3])
		book := SillokGoKrBook{
			Title:      mTitle[i][1],
			Seq:        id,
			Id:         m[1],
			KingCode:   m[2],
			Level:      level,
			Type:       m[4],
			Next:       m[5],
			Firstchild: m[6],
		}
		if strings.Contains(book.KingCode, "$") {
			continue
		}
		books = append(books, book)
	}
	return books, nil
}

func (r *SillokGoKr) getVolumesByBookId(book SillokGoKrBook) (volumes []SillokGoKrImage, err error) {
	level := strconv.Itoa(book.Level)
	q := url.Values{}
	q.Add("seq", book.Id)
	q.Add("kingCode", book.KingCode)
	q.Add("level", level)
	q.Add("treeType", "IMAGE")
	q.Add("next", book.Next)
	q.Add("firstchild", book.Firstchild)
	apiUrl := r.apiUrl + "search/ajaxExpandImageTree.do?" + q.Encode()

	fmt.Printf("book id=%s, title=%s\n", book.Id, book.Title)
	bs, err := r.getBody(apiUrl, r.dt.Jar)
	if err != nil {
		return
	}
	resp := SillokGoKrResponse{}
	if err = json.Unmarshal(bs, &resp); err != nil {
		return
	}
	for _, v := range resp.TreeList.List {
		//folder
		if v.Type == "L" {
			volumes = append(volumes, v)
		}
	}
	return
}

func (r *SillokGoKr) getImagesByVolumeId(book SillokGoKrBook, canvases *[]SillokGoKrImage) error {
	fmt.Printf("\r volume id=%s, title=%s                                          ", book.Id, book.Title)
	level := strconv.Itoa(book.Level)
	q := url.Values{}
	q.Add("seq", book.Id)
	q.Add("kingCode", book.KingCode)
	q.Add("level", level)
	q.Add("treeType", "IMAGE")
	q.Add("next", book.Next)
	q.Add("firstchild", book.Firstchild)
	apiUrl := r.apiUrl + "search/ajaxExpandImageTree.do?" + q.Encode()

	bs, err := r.getBody(apiUrl, r.dt.Jar)
	if err != nil {
		return err
	}
	resp := SillokGoKrResponse{}
	if err = json.Unmarshal(bs, &resp); err != nil {
		return err
	}
	for i, v := range resp.TreeList.List {
		//folder
		if v.Type == "L" {
			_level, _ := strconv.Atoi(v.Level)
			_seq, _ := strconv.Atoi(v.Seq)
			volume := SillokGoKrBook{
				Title:      v.Title,
				Seq:        _seq,
				Id:         v.Seq,
				KingCode:   v.KingCode,
				Level:      _level,
				Type:       v.Type,
				Next:       v.Next,
				Firstchild: v.Firstchild,
			}

			tab := ""
			for k := 0; k < _level; k++ {
				tab += "    "
			}
			r.index += i + 1
			r.bookMark += tab + v.Title + "......" + strconv.Itoa(r.index) + " \n"

			r.getImagesByVolumeId(volume, canvases)
			continue
		}
		//page type == "T"
		*canvases = append(*canvases, v)
	}
	return nil
}

func (r *SillokGoKr) imageIdUrl(kingCode, imageId string) string {
	kingArr := strings.Split(kingCode, "_")
	if strings.Index(kingArr[1], "$") != -1 {
		kingArr[1] = "000"
	}
	//imgUri := kingArr[0] + "/" + kingArr[1] + "/" + v.ImageId
	imgUrl := fmt.Sprintf("%simageProxy.do?filePath=/s_img/SILLOK/%s/%s/%s.jpg", r.apiUrl, kingArr[0], kingArr[1], imageId)
	return imgUrl
}

func (r *SillokGoKr) getBody(apiUrl string, jar *cookiejar.Jar) ([]byte, error) {
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
	if resp.GetStatusCode() != 200 || bs == nil {
		return nil, errors.New(fmt.Sprintf("ErrCode:%d, %s", resp.GetStatusCode(), resp.GetReasonPhrase()))
	}
	return bs, nil
}
