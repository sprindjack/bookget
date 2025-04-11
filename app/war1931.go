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
	"os"
	"regexp"
)

type War1931 struct {
	dt              *DownloadTask
	docType         string
	fileCode        string
	jsonUrlTemplate string
}

type War1931DetailsInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Result  struct {
		CollectNum string      `json:"collectNum"`
		Info       War1931Info `json:"info"`
	} `json:"result"`
}

type War1931Info struct {
	Notes                   string      `json:"notes"`
	SecondResponsibleNation interface{} `json:"secondResponsibleNation"`
	Language                []string    `json:"language"`
	Title                   string      `json:"title"`
	OriginalPlace           []string    `json:"originalPlace"`
	Duration                string      `json:"duration"`
	SeriesVolume            string      `json:"seriesVolume"`
	SeriesSubName           string      `json:"seriesSubName"`
	PublishEvolution        interface{} `json:"publishEvolution"`
	ImageUrl                string      `json:"imageUrl"`
	StartPageId             string      `json:"startPageId"`
	PageAmount              string      `json:"pageAmount"`
	Id                      string      `json:"id"`
	Place                   []string    `json:"place"`
	CreateTimeStr           interface{} `json:"createTimeStr"`
	RedFlag                 string      `json:"redFlag"`
	TimeRange               string      `json:"timeRange"`
	FirstResponsible        []string    `json:"firstResponsible"`
	PublishTimeAll          string      `json:"publishTimeAll"`
	PublishName             string      `json:"publishName"`
	KeyWords                []string    `json:"keyWords"`
	PublishTime             string      `json:"publishTime"`
	Amount                  interface{} `json:"amount"`
	OrgName                 string      `json:"orgName"`
	DocFormat               string      `json:"docFormat"`
	DocType                 string      `json:"docType"` //ts=图书，qk=期刊，bz=报纸
	SeriesName              string      `json:"seriesName"`
	IsResearch              string      `json:"isResearch"`
	IiifObj                 struct {
		FileCode    string      `json:"fileCode"`
		UniqTag     interface{} `json:"uniqTag"`
		VolumeInfo  interface{} `json:"volumeInfo"`
		DirName     string      `json:"dirName"`
		DirCode     string      `json:"dirCode"`
		CurrentPage string      `json:"currentPage"`
		StartPageId string      `json:"startPageId"`
		ImgUrl      string      `json:"imgUrl"`
		Content     string      `json:"content"`
		JsonUrl     string      `json:"jsonUrl"`
		IsUp        interface{} `json:"isUp"`
	} `json:"iiifObj"`
	FileCode               string      `json:"fileCode"`
	FirstCreationWay       []string    `json:"firstCreationWay"`
	ContentDesc            string      `json:"contentDesc"`
	DownloadSum            string      `json:"downloadSum"`
	Version                string      `json:"version"`
	Url                    string      `json:"url"`
	FirstResponsibleNation interface{} `json:"firstResponsibleNation"`
	CreateTime             interface{} `json:"createTime"`
	PublishCycle           []string    `json:"publishCycle"`
	OriginalTitle          string      `json:"originalTitle"`
	Publisher              []string    `json:"publisher"`
	VolumeInfoAllStr       string      `json:"volumeInfoAllStr"`
	SecondCreationWay      interface{} `json:"secondCreationWay"`
	Roundup                string      `json:"roundup"`
	SecondResponsible      []string    `json:"secondResponsible"`
	Remarks                string      `json:"remarks"`
}

type ResponseWar1931Manifest struct {
	Sequences []struct {
		Canvases []struct {
			Height string `json:"height"`
			Images []struct {
				Id         string `json:"@id"`
				Type       string `json:"@type"`
				Motivation string `json:"motivation"`
				On         string `json:"on"`
				Resource   struct {
					Format  string `json:"format"`
					Height  string `json:"height"`
					Id      string `json:"@id"`
					Type    string `json:"@type"`
					Service struct {
						Protocol string `json:"protocol"`
						Profile  string `json:"profile"`
						Width    int    `json:"width"`
						Id       string `json:"@id"`
						Context  string `json:"@context"`
						Height   int    `json:"height"`
					} `json:"service"`
					Width string `json:"width"`
				} `json:"resource"`
			} `json:"images"`
			Id    string `json:"@id"`
			Type  string `json:"@type"`
			Label string `json:"label"`
			Width string `json:"width"`
		} `json:"canvases"`
		Id               string `json:"@id"`
		Type             string `json:"@type"`
		Label            string `json:"label"`
		ViewingDirection string `json:"viewingDirection"`
		ViewingHint      string `json:"viewingHint"`
	} `json:"sequences"`
}

type War1931Qk struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Result  []struct {
		Title string `json:"title"`
		List  []struct {
			Year     string `json:"year"`
			DataList []struct {
				Id        string `json:"id"`
				Directory string `json:"directory"`
			} `json:"dataList"`
		} `json:"list"`
	} `json:"result"`
}

type War1931findDirectoryByMonth struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Result  []struct {
		Date    string `json:"date"`
		IiifObj struct {
			FileCode    string      `json:"fileCode"`
			UniqTag     string      `json:"uniqTag"`
			VolumeInfo  interface{} `json:"volumeInfo"`
			DirName     string      `json:"dirName"`
			DirCode     string      `json:"dirCode"`
			CurrentPage string      `json:"currentPage"`
			StartPageId string      `json:"startPageId"`
			ImgUrl      string      `json:"imgUrl"`
			Content     string      `json:"content"`
			JsonUrl     string      `json:"jsonUrl"`
			IsUp        interface{} `json:"isUp"`
		} `json:"iiifObj"`
		DirCode     string `json:"dirCode"`
		StartPageId string `json:"startPageId"`
		Id          string `json:"id"`
	} `json:"result"`
}

func (p *War1931) Init(iTask int, sUrl string) (msg string, err error) {
	p.dt = new(DownloadTask)
	p.dt.UrlParsed, err = url.Parse(sUrl)
	p.dt.Url = sUrl
	p.dt.Index = iTask
	p.dt.Jar, _ = cookiejar.New(nil)
	p.dt.BookId = p.getBookId(p.dt.Url)
	if p.dt.BookId == "" {
		return "requested URL was not found.", err
	}
	return p.download()
}

func (p *War1931) getBookId(sUrl string) (bookId string) {
	m := regexp.MustCompile(`fileCode=([A-z0-9_-]+)`).FindStringSubmatch(sUrl)
	if m != nil {
		return m[1]
	}
	return ""
}

func (p *War1931) mkdirAll(directory, vid string) (dirPath string) {
	switch p.docType {
	case "ts":
		p.dt.VolumeId = p.dt.UrlParsed.Host + "_" + p.dt.BookId + string(os.PathSeparator) + directory + "_vol." + vid
		break
	case "bz":
		p.dt.VolumeId = p.dt.UrlParsed.Host + "_" + p.dt.BookId + string(os.PathSeparator) + directory
		break
	case "qk":
		p.dt.VolumeId = p.dt.UrlParsed.Host + "_" + p.dt.BookId + string(os.PathSeparator) + directory + "_vol." + vid
		break
	default:
	}
	p.dt.SavePath = config.Conf.SaveFolder + string(os.PathSeparator) + p.dt.VolumeId
	_ = os.MkdirAll(p.dt.SavePath, os.ModePerm)
	return p.dt.SavePath
}

func (p *War1931) download() (msg string, err error) {
	name := util.GenNumberSorted(p.dt.Index)
	log.Printf("Get %s  %s\n", name, p.dt.Url)
	apiUrl := "https://" + p.dt.UrlParsed.Host + "/backend-prod/esBook/findDetailsInfo/" + p.dt.BookId
	partialVolumes, err := p.getVolumes(apiUrl, p.dt.Jar)
	if err != nil {
		fmt.Println(err)
		return "getVolumes", err
	}
	for k, parts := range partialVolumes {
		if !config.VolumeRange(k) {
			continue
		}
		log.Printf(" %d/%d, %d volumes \n", k+1, len(partialVolumes), len(parts.volumes))
		for i, vol := range parts.volumes {
			vid := util.GenNumberSorted(i + 1)
			p.mkdirAll(parts.directory, vid)
			canvases, err := p.getCanvases(vol, p.dt.Jar)
			if err != nil || canvases == nil {
				fmt.Println(err)
				continue
			}
			log.Printf(" %d/%d volume, %d pages \n", i+1, len(parts.volumes), len(canvases))
			p.do(canvases)
		}
	}
	return "", err
}

func (p *War1931) do(canvases []string) (msg string, err error) {
	if canvases == nil {
		return "", nil
	}
	referer := url.QueryEscape(p.dt.Url)
	args := []string{
		"-H", "Origin:" + referer,
		"-H", "Referer:" + referer,
		"-H", "User-Agent:" + config.Conf.UserAgent,
	}
	size := len(canvases)
	for i, uri := range canvases {
		if uri == "" || !config.PageRange(i, size) {
			continue
		}
		sortId := util.GenNumberSorted(i + 1)
		filename := sortId + config.Conf.FileExt
		inputUri := p.dt.SavePath + string(os.PathSeparator) + sortId + "_info.json"
		bs, err := p.getBody(uri, p.dt.Jar)
		if err != nil {
			continue
		}
		bsNew := regexp.MustCompile(`profile":\[([^{]+)\{"formats":([^\]]+)\],`).ReplaceAll(bs, []byte(`profile":[{"formats":["jpg"],`))
		err = os.WriteFile(inputUri, bsNew, os.ModePerm)
		if err != nil {
			return "", err
		}
		dest := p.dt.SavePath + string(os.PathSeparator) + filename
		if FileExist(dest) {
			continue
		}
		log.Printf("Get %s  %s\n", sortId, uri)
		if ret := util.StartProcess(inputUri, dest, args); ret == true {
			os.Remove(inputUri)
		}
	}
	return "", err
}

func (p *War1931) getVolumes(apiUrl string, jar *cookiejar.Jar) (volumes []PartialVolumes, err error) {
	bs, err := p.getBody(apiUrl, jar)
	if err != nil {
		return nil, err
	}
	var resp War1931DetailsInfo
	if err = json.Unmarshal(bs, &resp); err != nil {
		return nil, err
	}
	p.docType = resp.Result.Info.DocType
	p.fileCode = resp.Result.Info.FileCode
	jsonUrl := resp.Result.Info.IiifObj.JsonUrl
	p.jsonUrlTemplate, _ = p.getJsonUrlTemplate(jsonUrl, p.fileCode, p.docType)
	switch p.docType {
	case "ts":
		partVol := PartialVolumes{
			directory: p.fileCode,
			Title:     resp.Result.Info.Title,
			volumes:   []string{jsonUrl},
		}
		volumes = append(volumes, partVol)
		break
	case "bz":
		volumes, err = p.getVolumesForBz(jsonUrl, p.dt.Jar)
		break
	case "qk":
		volumes, err = p.getVolumesForQk(jsonUrl, p.dt.Jar)
		break
	default:
	}
	return volumes, nil
}

func (p *War1931) getJsonUrlTemplate(jsonUrl, fileCode, docType string) (jsonUrlTemplate string, err error) {
	if jsonUrl == "" {
		return "", err
	}
	u, err := url.Parse(jsonUrl)
	if err != nil {
		return "", err
	}
	if docType == "ts" {
		jsonUrlTemplate = u.Scheme + "://" + u.Host + "/" + fileCode + "/%s.json"
	} else {
		jsonUrlTemplate = u.Scheme + "://" + u.Host + "/" + fileCode + "/%s/%s.json"
	}
	return jsonUrlTemplate, err
}

func (p *War1931) getVolumesForBz(sUrl string, jar *cookiejar.Jar) (volumes []PartialVolumes, err error) {
	years, err := p.findBzYear(p.fileCode)
	if err != nil {
		return nil, err
	}
	fmt.Println()
	for _, year := range years {
		months, err := p.findBzMonth(year)
		if err != nil {
			continue
		}
		for _, month := range months {
			if len(month) == 1 {
				fmt.Printf("Test %s-0%s\r", year, month)
			} else {
				fmt.Printf("Test %s-%s\r", year, month)
			}
			apiUrl := "https://" + p.dt.UrlParsed.Host + "/backend-prod/esBook/findDirectoryByMonth?fileCode=" + p.fileCode + "&year=" + year + "&month=" + month
			bs, err := p.getBody(apiUrl, jar)
			if err != nil {
				break
			}
			var resp = new(War1931findDirectoryByMonth)
			if err := json.Unmarshal(bs, resp); err != nil {
				log.Printf("json.Unmarshal failed: %s\n", err)
				break
			}
			for _, item := range resp.Result {
				partVol := PartialVolumes{
					directory: year + "/" + item.Date,
					Title:     item.Date,
					volumes:   []string{item.IiifObj.JsonUrl},
				}
				volumes = append(volumes, partVol)
			}
		}
	}
	fmt.Println()
	return volumes, err
}

func (p *War1931) findBzYear(fileCode string) (years []string, err error) {
	apiUrl := "https://" + p.dt.UrlParsed.Host + "/backend-prod/esBook/findYear/" + fileCode
	bs, err := p.getBody(apiUrl, p.dt.Jar)
	if err != nil {
		return nil, err
	}
	type Response struct {
		Code    string   `json:"code"`
		Message string   `json:"message"`
		Result  []string `json:"result"`
	}
	var resp = new(Response)
	if err = json.Unmarshal(bs, resp); err != nil {
		log.Printf("json.Unmarshal failed: %s\n", err)
		return
	}
	return resp.Result, err
}

func (p *War1931) findBzMonth(year string) (years []string, err error) {
	apiUrl := "https://" + p.dt.UrlParsed.Host + "/backend-prod/esBook/findMonth?fileCode=" + p.fileCode + "&year=" + year
	bs, err := p.getBody(apiUrl, p.dt.Jar)
	if err != nil {
		return nil, err
	}
	type Response struct {
		Code    string   `json:"code"`
		Message string   `json:"message"`
		Result  []string `json:"result"`
	}
	var resp = new(Response)
	if err = json.Unmarshal(bs, resp); err != nil {
		log.Printf("json.Unmarshal failed: %s\n", err)
		return
	}
	return resp.Result, err
}

func (p *War1931) getVolumesForQk(sUrl string, jar *cookiejar.Jar) (volumes []PartialVolumes, err error) {
	apiUrl := "https://" + p.dt.UrlParsed.Host + "/backend-prod/esBook/findDirectoryByYear/" + p.fileCode
	bs, err := p.getBody(apiUrl, jar)
	if err != nil {
		return nil, err
	}
	var resp = new(War1931Qk)
	if err = json.Unmarshal(bs, resp); err != nil {
		log.Printf("json.Unmarshal failed: %s\n", err)
		return
	}
	for _, items := range resp.Result {
		for _, item := range items.List {
			partVol := PartialVolumes{
				directory: item.Year,
				Title:     items.Title,
				volumes:   nil,
			}
			partVol.volumes = make([]string, 0, len(item.DataList))
			for _, v := range item.DataList {
				jsonUrl := fmt.Sprintf(p.jsonUrlTemplate, v.Id, v.Id)
				partVol.volumes = append(partVol.volumes, jsonUrl)
			}
			volumes = append(volumes, partVol)
		}
	}
	return volumes, err
}

func (p *War1931) getCanvases(sUrl string, jar *cookiejar.Jar) (canvases []string, err error) {
	bs, err := p.getBody(sUrl, jar)
	if err != nil {
		return nil, err
	}
	var manifest = new(ResponseWar1931Manifest)
	if err = json.Unmarshal(bs, manifest); err != nil {
		log.Printf("json.Unmarshal failed: %s\n", err)
		return
	}
	if len(manifest.Sequences) == 0 {
		return
	}
	size := len(manifest.Sequences[0].Canvases)
	canvases = make([]string, 0, size)
	for _, canvase := range manifest.Sequences[0].Canvases {
		for _, image := range canvase.Images {
			iiiInfo := fmt.Sprintf("%s/info.json", image.Resource.Service.Id)
			canvases = append(canvases, iiiInfo)
		}
	}
	return canvases, nil
}

func (p *War1931) getBody(sUrl string, jar *cookiejar.Jar) ([]byte, error) {
	ctx := context.Background()
	cli := gohttp.NewClient(ctx, gohttp.Options{
		CookieFile: config.Conf.CookieFile,
		CookieJar:  jar,
		Headers: map[string]interface{}{
			"User-Agent":      config.Conf.UserAgent,
			"Accept-Language": "zh-CN,zh;q=0.8,zh-TW;q=0.7,zh-HK;q=0.5,en-US;q=0.3,en;q=0.2",
			"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8",
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
func (p *War1931) postBody(sUrl string, d []byte) ([]byte, error) {
	//TODO implement me
	panic("implement me")
}
