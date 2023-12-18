package app

import (
	"bookget/config"
	"bookget/lib/util"
	"encoding/json"
	"fmt"
	"log"
	"net/http/cookiejar"
	"net/url"
	"os"
	"regexp"
)

type Njuedu struct {
	dt     *DownloadTask
	typeId int
}

type NjuCatalog struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    []struct {
		BookId         string        `json:"bookId"`
		BookName       string        `json:"bookName"`
		VolumeNum      string        `json:"volumeNum"`
		ImgDescription interface{}   `json:"imgDescription"`
		Catalogues     []interface{} `json:"catalogues"`
	} `json:"data"`
}
type NjuResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Title      string   `json:"title"`
		ServerBase string   `json:"serverBase"`
		Images     []string `json:"images"`
	} `json:"data"`
}

type NjuDetail struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    []struct {
		Id             int    `json:"id"`
		BookId         string `json:"bookId"`
		Num            int    `json:"num"`
		AttributeId    int    `json:"attributeId"`
		AttributeValue string `json:"attributeValue"`
		Operator       int    `json:"operator"`
		OperatorName   string `json:"operatorName"`
		CreateTime     int    `json:"createTime"`
		UpdateTime     int    `json:"updateTime"`
		TypeId         int    `json:"typeId"`
		Captions       string `json:"captions"`
	} `json:"data"`
}

type ResponseTiles struct {
	Tiles map[string]Item `json:"tiles"`
}

func (p *Njuedu) Init(iTask int, sUrl string) (msg string, err error) {
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

func (p *Njuedu) getBookId(sUrl string) (bookId string) {
	if m := regexp.MustCompile(`(?i)bookId=([A-z0-9_-]+)`).FindStringSubmatch(sUrl); m != nil {
		bookId = m[1]
	}
	return bookId
}

func (p *Njuedu) download() (msg string, err error) {
	name := util.GenNumberSorted(p.dt.Index)
	log.Printf("Get %s  %s\n", name, p.dt.Url)
	p.typeId, err = p.getDetail(p.dt.BookId, p.dt.Jar)
	if err != nil {
		fmt.Println(err)
		return "getDetail", err
	}
	respVolume, err := p.getVolumes(p.dt.BookId, p.dt.Jar)
	if err != nil {
		fmt.Println(err)
		return "getVolumes", err
	}
	for i, vol := range respVolume {
		if config.Conf.Volume > 0 && config.Conf.Volume != i+1 {
			continue
		}
		vid := util.GenNumberSorted(i + 1)
		p.dt.SavePath = CreateDirectory(p.dt.UrlParsed.Host, p.dt.BookId, vid)
		canvases, err := p.getCanvases(vol, p.dt.Jar)
		if err != nil || canvases == nil {
			fmt.Println(err)
			continue
		}
		log.Printf(" %d/%d volume, %d pages \n", i+1, len(respVolume), len(canvases))
		p.do(canvases)
	}
	return msg, err
}

func (p *Njuedu) do(dziUrls []string) (msg string, err error) {
	if dziUrls == nil {
		return "", err
	}
	referer := url.QueryEscape(p.dt.Url)
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
		fileName := util.GenNumberSorted(i+1) + config.Conf.FileExt
		inputUri := p.dt.SavePath + val
		outfile := p.dt.SavePath + fileName
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

func (p *Njuedu) getDetail(bookId string, jar *cookiejar.Jar) (typeId int, err error) {
	apiUrl := "https://" + p.dt.UrlParsed.Host + "/portal/book/getBookById?bookId=" + bookId
	bs, err := getBody(apiUrl, jar)
	if err != nil {
		return 0, err
	}
	var result NjuDetail
	if err = json.Unmarshal(bs, &result); err != nil {
		log.Printf("json.Unmarshal failed: %s\n", err)
		return
	}
	for _, v := range result.Data {
		if v.TypeId > 0 {
			typeId = v.TypeId
			break
		}
	}
	return typeId, err
}

func (p *Njuedu) getVolumes(bookId string, jar *cookiejar.Jar) (volumes []string, err error) {
	apiUrl := fmt.Sprintf("https://%s/portal/book/getMasterSlaveCatalogue?typeId=%d&bookId=%s", p.dt.UrlParsed.Host, p.typeId, bookId)
	bs, err := getBody(apiUrl, jar)
	if err != nil {
		return nil, err
	}
	var result NjuCatalog
	if err = json.Unmarshal(bs, &result); err != nil {
		log.Printf("json.Unmarshal failed: %s\n", err)
		return
	}
	for _, d := range result.Data {
		volUrl := fmt.Sprintf("https://%s/portal/book/view?bookId=%s&typeId=%d", p.dt.UrlParsed.Host, d.BookId, p.typeId)
		volumes = append(volumes, volUrl)
	}
	return volumes, err

}

func (p *Njuedu) getCanvases(sUrl string, jar *cookiejar.Jar) (canvases []string, err error) {
	bs, err := getBody(sUrl, jar)
	if err != nil {
		return nil, err
	}
	var result NjuResponse
	if err = json.Unmarshal(bs, &result); err != nil {
		log.Printf("json.Unmarshal failed: %s\n", err)
		return
	}
	for _, id := range result.Data.Images {
		sortId := fmt.Sprintf("%s.json", id)
		canvases = append(canvases, sortId)
	}

	serverBase := "https://" + p.dt.UrlParsed.Host + result.Data.ServerBase
	jsonUrl := serverBase + "/tiles/infos.json"
	text := `{
    "Image": {
    "xmlns":    "https://schemas.microsoft.com/deepzoom/2009",
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
	bs, err = getBody(jsonUrl, jar)
	if err != nil {
		return nil, err
	}
	var resp ResponseTiles
	if err = json.Unmarshal(bs, &resp); err != nil {
		return
	}
	if resp.Tiles == nil {
		return
	}
	ext := config.Conf.FileExt[1:]
	for key, item := range resp.Tiles {
		sortId := fmt.Sprintf("%s.json", key)
		dest := p.dt.SavePath + sortId
		serverUrl := fmt.Sprintf("%s/tiles/%s/", serverBase, key)
		jsonText := ""
		//ext := strings.ToLower(item.Extension)
		if item.TileSize.W == 0 {
			jsonText = fmt.Sprintf(text, serverUrl, ext, item.TileSize2.Width, item.Height, item.Width)
		} else {
			jsonText = fmt.Sprintf(text, serverUrl, ext, item.TileSize.W, item.Height, item.Width)
		}
		_ = os.WriteFile(dest, []byte(jsonText), os.ModePerm)
	}
	return canvases, nil
}

func (p *Njuedu) getBody(sUrl string, jar *cookiejar.Jar) ([]byte, error) {
	//TODO implement me
	panic("implement me")
}

func (p *Njuedu) postBody(sUrl string, d []byte) ([]byte, error) {
	//TODO implement me
	panic("implement me")
}
