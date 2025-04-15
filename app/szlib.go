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
	"strings"
	"sync"
)

type SzLib struct {
	dt *DownloadTask
}
type SzlibResultVolumes struct {
	Meta struct {
		Topic  string `json:"topic"`
		Uri856 struct {
			Image struct {
				Brief     []interface{} `json:"brief"`
				Publish   []interface{} `json:"publish"`
				IndexPage []interface{} `json:"indexPage"`
			} `json:"image"`
			Pdf struct {
				FullText []interface{} `json:"fullText"`
			} `json:"pdf"`
		} `json:"uri_856"`
		UAbstract         string `json:"U_Abstract"`
		USubject          string `json:"U_Subject"`
		UContributor      string `json:"U_Contributor"`
		UAuthor           string `json:"U_Author"`
		UTitle            string `json:"U_Title"`
		UPublishYear      string `json:"U_PublishYear"`
		UZuozheFangshi    string `json:"U_Zuozhe_Fangshi"`
		ULibrary          string `json:"U_Library"`
		UZhuangzhenxinshi string `json:"U_Zhuangzhenxinshi"`
		UPublisher        string `json:"U_Publisher"`
		UPlace            string `json:"U_Place"`
		UCunjuan          string `json:"U_Cunjuan"`
		UZuozheShijian    string `json:"U_Zuozhe_Shijian"`
		USeries           string `json:"U_Series"`
		UZiliaojibie      string `json:"U_Ziliaojibie"`
		UGuest            string `json:"U_Guest"`
		UTimingJianti     string `json:"U_Timing_Jianti"`
		UKeywords         string `json:"U_Keywords"`
		ULibCallno        string `json:"U_LibCallno"`
		UVersionLeixin    string `json:"U_Version_Leixin"`
		USubjectRegular2  string `json:"U_Subject_Regular2"`
		USubjectRegular   string `json:"U_Subject_Regular"`
		UExpectDate       string `json:"U_ExpectDate"`
		UPage             string `json:"U_Page"`
		UVersion          string `json:"U_Version"`
		UPuchabianhao     string `json:"U_Puchabianhao"`
		UCallno           string `json:"U_Callno"`
		UPublish          string `json:"U_Publish"`
		UFence            string `json:"U_Fence"`
		Volume            string `json:"volume"`
	} `json:"meta"`
	Volumes []SzlibDirectory `json:"directory"`
}

type SzlibDirectory struct {
	Name     string `json:"name"`
	Volume   string `json:"volume"`
	Page     string `json:"page"`
	HasText  string `json:"has_text"`
	Children []struct {
		Volume   string        `json:"volume"`
		Children []interface{} `json:"children"`
		Page     string        `json:"page"`
	} `json:"children"`
}

type SzlibResultPage struct {
	TextInfo struct {
	} `json:"text_info"`
	PicInfo struct {
		Description string `json:"description"`
		Tolvol      string `json:"tolvol"`
		Period      string `json:"period"`
		Topic       string `json:"topic"`
		Title       string `json:"title"`
		Path        string `json:"path"`
	} `json:"pic_info"`
	BookImageUrl string `json:"book_image_url"`
}

func (p *SzLib) Init(iTask int, sUrl string) (msg string, err error) {
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

func (p *SzLib) getBookId(sUrl string) (bookId string) {
	m := regexp.MustCompile(`book_id=([A-z\d]+)`).FindStringSubmatch(sUrl)
	if m != nil {
		bookId = m[1]
	}
	return bookId
}

func (p *SzLib) download() (msg string, err error) {
	name := util.GenNumberSorted(p.dt.Index)
	log.Printf("Get %s  %s\n", name, p.dt.Url)

	respVolume, err := p.getVolumes(p.dt.Url, p.dt.Jar)
	if err != nil {
		fmt.Println(err)
		return "getVolumes", err
	}
	sizeVol := len(respVolume.Volumes)
	for i, vol := range respVolume.Volumes {
		if !config.VolumeRange(i) {
			continue
		}
		fmt.Printf("\r Test volume %d ... ", i+1)
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

func (p *SzLib) do(imgUrls []string) (msg string, err error) {
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

func (p *SzLib) getVolumes(sUrl string, jar *cookiejar.Jar) (*SzlibResultVolumes, error) {
	apiUrl := fmt.Sprintf("https://%s/stgj2021/book_view/%s", p.dt.UrlParsed.Host, p.dt.BookId)
	bs, err := p.getBody(apiUrl, jar)
	if err != nil {
		return nil, err
	}
	var rstVolumes = new(SzlibResultVolumes)
	if err = json.Unmarshal(bs, rstVolumes); err != nil {
		log.Printf("json.Unmarshal failed: %s\n", err)
		return nil, err
	}
	return rstVolumes, err
}

func (p *SzLib) getCanvases(vol SzlibDirectory, jar *cookiejar.Jar) ([]string, error) {
	p1, err := p.getSinglePage(p.dt.BookId, vol.Volume, vol.Children[0].Page)
	pos := strings.LastIndex(p1, "/")
	urlPre := p1[:pos]
	ext := util.FileExt(p1)
	canvases := make([]string, 0, len(vol.Children))
	for _, child := range vol.Children {
		imgUrl := fmt.Sprintf("%s/%s%s", urlPre, child.Page, ext)
		canvases = append(canvases, imgUrl)
	}
	return canvases, err
}

func (p *SzLib) getBody(sUrl string, jar *cookiejar.Jar) ([]byte, error) {
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

func (p *SzLib) postBody(sUrl string, d []byte) ([]byte, error) {
	//TODO implement me
	panic("implement me")
}

func (p *SzLib) getSinglePage(bookId string, volumeId string, page string) (string, error) {
	sUrl := fmt.Sprintf("https://%s/stgj2021/book_page/%s/%s/%s", p.dt.UrlParsed.Host, bookId, volumeId, page)
	bs, err := p.getBody(sUrl, p.dt.Jar)
	if err != nil {
		return "", err
	}
	rstPage := new(SzlibResultPage)
	if err = json.Unmarshal(bs, rstPage); err != nil {
		return "", err
	}
	return rstPage.BookImageUrl + rstPage.PicInfo.Path, nil
}
