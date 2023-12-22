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
	"regexp"
	"strings"
	"sync"
)

// Graphql 查manifestUrl
type Graphql struct {
	Data struct {
		ResourcesByOrangelightIds []struct {
			Id        string      `json:"id"`
			Thumbnail interface{} `json:"thumbnail"`
			Url       string      `json:"url"`
			Members   []struct {
				Id       string `json:"id"`
				Typename string `json:"__typename"`
			} `json:"members"`
			Typename      string `json:"__typename"`
			ManifestUrl   string `json:"manifestUrl"`
			OrangelightId string `json:"orangelightId"`
		} `json:"resourcesByOrangelightIds"`
	} `json:"data"`
}

type PrincetonResponseManifest struct {
	Manifests []struct {
		Context   string   `json:"@context"`
		Type      string   `json:"@type"`
		Id        string   `json:"@id"`
		Label     []string `json:"label"`
		Thumbnail struct {
			Id      string `json:"@id"`
			Service struct {
				Context string `json:"@context"`
				Id      string `json:"@id"`
				Profile string `json:"profile"`
			} `json:"service"`
		} `json:"thumbnail"`
	} `json:"manifests"`
}

// Manifest 查info.json
type PrincetonResponseManifest2 struct {
	Sequences []struct {
		Type      string `json:"@type"`
		Id        string `json:"@id"`
		Rendering []struct {
			Id     string `json:"@id"`
			Label  string `json:"label"`
			Format string `json:"format"`
		} `json:"rendering"`
		Canvases []struct {
			Type      string `json:"@type"`
			Id        string `json:"@id"`
			Label     string `json:"label"`
			Rendering []struct {
				Id     string `json:"@id"`
				Label  string `json:"label"`
				Format string `json:"format"`
			} `json:"rendering"`
			Width  int `json:"width"`
			Height int `json:"height"`
			Images []struct {
				Type       string `json:"@type"`
				Motivation string `json:"motivation"`
				Resource   struct {
					Type    string `json:"@type"`
					Id      string `json:"@id"`
					Height  int    `json:"height"`
					Width   int    `json:"width"`
					Format  string `json:"format"`
					Service struct {
						Context string `json:"@context"`
						Id      string `json:"@id"`
						Profile string `json:"profile"`
					} `json:"service"`
				} `json:"resource"`
				Id string `json:"@id"`
				On string `json:"on"`
			} `json:"images"`
		} `json:"canvases"`
		ViewingHint string `json:"viewingHint"`
	} `json:"sequences"`
}

type Princeton struct {
	dt *DownloadTask
}

func (p *Princeton) Init(iTask int, sUrl string) (msg string, err error) {
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

func (p *Princeton) getBookId(sUrl string) (bookId string) {
	m := regexp.MustCompile(`catalog/([A-z\d]+)`).FindStringSubmatch(sUrl)
	if m != nil {
		bookId = m[1]
	}
	return bookId
}

func (p *Princeton) download() (msg string, err error) {
	name := util.GenNumberSorted(p.dt.Index)
	log.Printf("Get %s  %s\n", name, p.dt.Url)

	respVolume, err := p.getVolumes(p.dt.Url, p.dt.Jar)
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
	return "", nil
}

func (p *Princeton) do(canvases []string) (msg string, err error) {
	if canvases == nil {
		return
	}
	fmt.Println()
	referer := p.dt.Url
	size := len(canvases)
	var wg sync.WaitGroup
	q := QueueNew(int(config.Conf.Threads))
	for i, uri := range canvases {
		if uri == "" || !config.PageRange(i, size) {
			continue
		}
		sortId := util.GenNumberSorted(i + 1)
		filename := sortId + config.Conf.FileExt
		dest := p.dt.SavePath + filename
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
				CookieJar:   p.dt.Jar,
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

func (p *Princeton) getVolumes(sUrl string, jar *cookiejar.Jar) (volumes []string, err error) {
	var manifestUrl = ""
	//
	if strings.Contains(sUrl, "dpul.princeton.edu") {
		bs, err := p.getBody(sUrl, jar)
		if err != nil {
			return nil, err
		}
		//页面中有https://figgy.princeton.edu/viewer#?manifest=https%3A%2F%2Ffiggy.princeton.edu%2Fconcern%2Fscanned_resources%2F64ee594e-4735-4a8e-b712-73b8c00ec56b%2Fmanifest&config=https%3A%2F%2Ffiggy.princeton.edu%2Fviewer%2Fexhibit%2Fconfig%3Fmanifest%3Dhttps%3A%2F%2Ffiggy.princeton.edu%2Fconcern%2Fscanned_resources%2F64ee594e-4735-4a8e-b712-73b8c00ec56b%2Fmanifest

		m := regexp.MustCompile(`manifest=(.+?)&`).FindStringSubmatch(string(bs))
		if m == nil {
			return nil, err
		}
		manifestUrl, _ = url.QueryUnescape(m[1])
	}

	if strings.Contains(sUrl, "catalog.princeton.edu") {
		//Graphql 查询
		phql := new(Graphql)
		d := fmt.Sprintf(`{"operationName":"GetResourcesByOrangelightIds","variables":{"ids":["%s"]},"query":"query GetResourcesByOrangelightIds($ids: [String!]!) {\n  resourcesByOrangelightIds(ids: $ids) {\n    id\n    thumbnail {\n      iiifServiceUrl\n      thumbnailUrl\n      __typename\n    }\n    url\n    members {\n      id\n      __typename\n    }\n    ... on ScannedResource {\n      manifestUrl\n      orangelightId\n      __typename\n    }\n    ... on ScannedMap {\n      manifestUrl\n      orangelightId\n      __typename\n    }\n    ... on Coin {\n      manifestUrl\n      orangelightId\n      __typename\n    }\n    __typename\n  }\n}\n"}`,
			p.dt.BookId)
		bs, err := p.postBody("https://figgy.princeton.edu/graphql", []byte(d))
		if err != nil {
			return nil, err

		}
		if err = json.Unmarshal(bs, phql); err != nil {
			log.Printf("json.Unmarshal failed: %s\n", err)
			return nil, err
		}
		for _, v := range phql.Data.ResourcesByOrangelightIds {
			manifestUrl = v.ManifestUrl
		}

	}
	if manifestUrl == "" {
		return
	}

	//查全书分卷URL
	var manifest = new(PrincetonResponseManifest)
	body, err := p.getBody(manifestUrl, jar)
	if err != nil {
		return
	}
	if err = json.Unmarshal(body, manifest); err != nil {
		log.Printf("json.Unmarshal failed: %s\n", err)
		return
	}

	if manifest.Manifests == nil {
		volumes = append(volumes, manifestUrl)
	} else {
		//分卷URL处理
		for _, vol := range manifest.Manifests {
			volumes = append(volumes, vol.Id)
		}
	}
	return volumes, nil
}

func (p *Princeton) getCanvases(sUrl string, jar *cookiejar.Jar) (canvases []string, err error) {
	var manifest2 = new(PrincetonResponseManifest2)
	body, err := p.getBody(sUrl, jar)
	if err != nil {
		return
	}
	if err = json.Unmarshal(body, manifest2); err != nil {
		log.Printf("json.Unmarshal failed: %s\n", err)
		return
	}
	i := len(manifest2.Sequences[0].Canvases)
	canvases = make([]string, 0, i)

	//分卷URL处理
	for _, sequences := range manifest2.Sequences {
		for _, canvase := range sequences.Canvases {
			for _, image := range canvase.Images {
				//JPEG URL
				imgUrl := image.Resource.Service.Id + "/" + config.Conf.Format
				canvases = append(canvases, imgUrl)
			}
		}
	}

	return canvases, nil
}

func (p *Princeton) getBody(apiUrl string, jar *cookiejar.Jar) ([]byte, error) {
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

func (p *Princeton) postBody(sUrl string, d []byte) ([]byte, error) {
	ctx := context.Background()
	cli := gohttp.NewClient(ctx, gohttp.Options{
		CookieFile: config.Conf.CookieFile,
		CookieJar:  p.dt.Jar,
		Headers: map[string]interface{}{
			"User-Agent":   config.Conf.UserAgent,
			"Content-Type": "application/json",
			"authority":    "figgy.princeton.edu",
			"referer":      p.dt.Url,
		},
		Body: d,
	})
	resp, err := cli.Post(sUrl)
	if err != nil {
		return nil, err
	}
	bs, _ := resp.GetBody()
	return bs, err
}
