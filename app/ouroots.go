package app

import (
	"bookget/config"
	"bookget/pkg/gohttp"
	"bookget/pkg/progressbar"
	"bookget/pkg/util"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http/cookiejar"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Ouroots struct {
	dt      *DownloadTask
	Counter int
	bar     *progressbar.ProgressBar
}

type OurootsResponseLoginAnonymousUser struct {
	StatusCode string `json:"statusCode"`
	Msg        string `json:"msg"`
	Token      string `json:"token"`
}

type OurootsResponseCatalogImage struct {
	StatusCode string `json:"statusCode"`
	Msg        string `json:"msg"`
	ImagePath  string `json:"imagePath"`
	ImageSize  int    `json:"imageSize"`
	DocPath    string `json:"docPath"`
}
type OurootsResponseVolume struct {
	StatusCode string `json:"statusCode"`
	Msg        string `json:"msg"`
	Volume     []struct {
		Name     string `json:"name"`
		Pages    int    `json:"pages"`
		VolumeId int    `json:"volumeId"`
	} `json:"volume"`
	Catalogue []struct {
		Key           string `json:"_key"`
		Id            string `json:"_id"`
		Rev           string `json:"_rev"`
		BatchID       string `json:"batchID"`
		PageProp      string `json:"page_prop"`
		BookId        string `json:"book_id"`
		ChapterName   string `json:"chapter_name"`
		SerialNum     string `json:"serial_num"`
		AdminId       string `json:"adminId"`
		CreateTime    int64  `json:"createTime"`
		IsLike        bool   `json:"isLike"`
		IsCollect     bool   `json:"isCollect"`
		ViewNum       int    `json:"viewNum"`
		LikeNum       int    `json:"likeNum"`
		CollectionNum int    `json:"collectionNum"`
		ShareNum      int    `json:"shareNum"`
		VolumeID      int    `json:"volumeID"`
		EndNum        *int   `json:"end_num"`
		VolumeNum     string `json:"volume_num,omitempty"`
		PageNum       string `json:"page_num,omitempty"`
	} `json:"catalogue"`
}

func (p *Ouroots) Init(iTask int, sUrl string) (msg string, err error) {
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

func (p *Ouroots) getBookId(sUrl string) (bookId string) {
	m := regexp.MustCompile(`(?i)\.html\?([A-z0-9]+)`).FindStringSubmatch(sUrl)
	if m != nil {
		bookId = m[1]
	}
	return bookId
}

func (p *Ouroots) download() (msg string, err error) {
	name := util.GenNumberSorted(p.dt.Index)
	log.Printf("Get %s  %s\n", name, p.dt.Url)

	respVolume, err := p.getVolumes(p.dt.BookId, p.dt.Jar)
	if err != nil || respVolume.StatusCode != "200" {
		fmt.Println(err)
		return "getVolumes", err
	}
	//不按卷下载，所有图片存一个目录
	p.dt.SavePath = CreateDirectory(p.dt.UrlParsed.Host, p.dt.BookId, "")
	macCounter := 0
	for i, vol := range respVolume.Volume {
		if !config.VolumeRange(i) {
			continue
		}
		macCounter += vol.Pages
	}
	fmt.Println()
	p.bar = progressbar.Default(int64(macCounter), "downloading")
	for i, vol := range respVolume.Volume {
		if !config.VolumeRange(i) {
			continue
		}
		p.do(vol.Pages, vol.VolumeId)
	}
	return "", nil
}

func (p *Ouroots) do(pageTotal int, volumeId int) (msg string, err error) {
	token, err := p.getToken()
	if err != nil {
		p.bar.Clear()
		return "token not found.", err
	}
	for i := 1; i <= pageTotal; i++ {
		sortId := fmt.Sprintf("%s.jpg", util.GenNumberSorted(p.Counter+1))
		dest := p.dt.SavePath + sortId
		if util.FileExist(dest) {
			p.Counter++
			p.bar.Add(1)
			time.Sleep(40 * time.Millisecond)
			continue
		}
		respImage, err := p.getBase64Image(p.dt.BookId, volumeId, i, "", token)
		if err != nil || respImage.StatusCode != "200" {
			continue
		}
		if pos := strings.Index(respImage.ImagePath, "data:image/jpeg;base64,"); pos != -1 {
			data := respImage.ImagePath[pos+len("data:image/jpeg;base64,"):]
			bs, err := base64.StdEncoding.DecodeString(data)
			if err != nil || bs == nil {
				//log.Println(err)
				continue
			}
			_ = os.WriteFile(dest, bs, os.ModePerm)
			p.Counter++
			p.bar.Add(1)
			time.Sleep(40 * time.Millisecond)
		}
	}
	return "", nil
}

func (p *Ouroots) getVolumes(catalogKey string, jar *cookiejar.Jar) (OurootsResponseVolume, error) {
	ctx := context.Background()
	cli := gohttp.NewClient(ctx, gohttp.Options{
		CookieFile: config.Conf.CookieFile,
		CookieJar:  jar,
		Headers: map[string]interface{}{
			"User-Agent": config.Conf.UserAgent,
		},
		Query: map[string]interface{}{
			"catalogKey": catalogKey,
			"bookid":     "", //目录索引，不重要
		},
	})
	resp, err := cli.Get("http://dsnode.ouroots.nlc.cn/gtService/data/catalogVolume")
	bs, _ := resp.GetBody()
	if bs == nil {
		return OurootsResponseVolume{}, errors.New(resp.GetReasonPhrase())
	}

	var respVolume OurootsResponseVolume
	if err = json.Unmarshal(bs, &respVolume); err != nil {
		log.Printf("json.Unmarshal failed: %s\n", err)
		return respVolume, errors.New(resp.GetReasonPhrase())
	}
	return respVolume, nil
}

func (p *Ouroots) getCanvases(sUrl string, jar *cookiejar.Jar) (canvases []string, err error) {
	//TODO implement me
	panic("implement me")
}

func (p *Ouroots) getBody(sUrl string, jar *cookiejar.Jar) ([]byte, error) {
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

func (p *Ouroots) postBody(sUrl string, d []byte) ([]byte, error) {
	//TODO implement me
	panic("implement me")
}

func (p *Ouroots) getToken() (string, error) {
	bs, err := p.getBody("http://dsNode.ouroots.nlc.cn/loginAnonymousUser", p.dt.Jar)
	if err != nil {
		return "", err
	}
	var respLoginAnonymousUser OurootsResponseLoginAnonymousUser
	if err = json.Unmarshal(bs, &respLoginAnonymousUser); err != nil {
		return "", err
	}
	return respLoginAnonymousUser.Token, nil
}
func (p *Ouroots) getBase64Image(catalogKey string, volumeId, page int, userKey, token string) (respImage OurootsResponseCatalogImage, err error) {
	jar, _ := cookiejar.New(nil)
	ctx := context.Background()
	cli := gohttp.NewClient(ctx, gohttp.Options{
		CookieFile: config.Conf.CookieFile,
		CookieJar:  jar,
		Headers: map[string]interface{}{
			"User-Agent": config.Conf.UserAgent,
		},
		Query: map[string]interface{}{
			"catalogKey": catalogKey,
			"volumeId":   strconv.FormatInt(int64(volumeId), 10),
			"page":       strconv.FormatInt(int64(page), 10),
			"userKey":    userKey,
			"token":      token,
		},
	})
	resp, err := cli.Get("http://dsnode.ouroots.nlc.cn/data/catalogImage")
	bs, _ := resp.GetBody()
	if bs == nil {
		err = errors.New(resp.GetReasonPhrase())
		return
	}
	if err = json.Unmarshal(bs, &respImage); err != nil {
		return
	}
	return respImage, nil
}
