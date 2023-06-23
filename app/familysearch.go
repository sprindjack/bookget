package app

import (
	"bookget/config"
	"bookget/lib/curl"
	"bookget/lib/gohttp"
	xhash "bookget/lib/hash"
	"bookget/lib/util"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"
)

type Familysearch struct {
	dt          *DownloadTask
	apiUrl      string
	urlType     int
	dziTemplate string
	userAgent   string
	baseUrl     string
}
type FamilysearchImageData struct {
	DgsNum      string
	WaypointURL string
	ImageURL    string
}
type FamilysearchResultError struct {
	Error struct {
		Message     string   `json:"message"`
		FailedRoles []string `json:"failedRoles"`
		StatusCode  int      `json:"statusCode"`
	} `json:"error"`
}

func (r *Familysearch) Init(iTask int, sUrl string) (msg string, err error) {
	r.dt = new(DownloadTask)
	r.dt.UrlParsed, err = url.Parse(sUrl)
	r.dt.Url = sUrl
	r.dt.Index = iTask
	r.dt.BookId = r.getBookId(r.dt.Url)
	if r.dt.BookId == "" {
		return "requested URL was not found.", err
	}
	r.baseUrl, _ = r.getBaseUrl(r.dt.Url)
	r.dt.Jar, _ = cookiejar.New(nil)
	//  "https://www.familysearch.org/search/filmdata/filmdatainfo"
	r.apiUrl = r.dt.UrlParsed.Scheme + "://" + r.dt.UrlParsed.Host + "/search/filmdata/filmdatainfo"
	return r.download()
}

func (r *Familysearch) getBookId(sUrl string) (bookId string) {
	m := regexp.MustCompile(`(?i)wc=([^&]+)`).FindStringSubmatch(sUrl)
	if m != nil {
		mh := xhash.NewMultiHasher()
		io.Copy(mh, bytes.NewBuffer([]byte(m[1])))
		bookId, _ = mh.SumString(xhash.CRC32, false)
		r.urlType = 0 //中國族譜收藏 1239-2014年 https://www.familysearch.org/search/collection/1787988
		return bookId
	}
	m = regexp.MustCompile(`(?i)rmsId=([A-z\d-_]+)`).FindStringSubmatch(sUrl)
	if m != nil {
		bookId = m[1]
		r.urlType = 1 //家谱图像 https://www.familysearch.org/records/images/
		return bookId
	}
	m = regexp.MustCompile(`(?i)groupId=([A-z\d-_]+)`).FindStringSubmatch(sUrl)
	if m != nil {
		bookId = m[1]
		r.urlType = 2 //家谱图像 https://www.familysearch.org/ark:/61903/3:1:3QS7-L9S9-WS92?view=explore&groupId=M94X-6HR
		return bookId
	}
	m = regexp.MustCompile(`(?i)ark:/(?:[A-z0-9-_:]+)/([A-z\d-_:]+)`).FindStringSubmatch(sUrl)
	if m != nil {
		bookId = m[1]
		r.urlType = 3 //微卷 https://www.familysearch.org/ark:/61903/3:1:3QSQ-G9MC-ZS8F-4?cat=1101921
	}
	return bookId
}

func (r *Familysearch) getBaseUrl(sUrl string) (baseUrl string, err error) {
	ctx := context.Background()
	cli := gohttp.NewClient(ctx, gohttp.Options{
		CookieFile: config.Conf.CookieFile,
		CookieJar:  r.dt.Jar,
		Headers: map[string]interface{}{
			"User-Agent": config.Conf.UserAgent,
			"Accept":     "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7",
		},
	})
	resp, err := cli.Get(sUrl)
	if err != nil {
		return
	}
	bs, _ := resp.GetBody()
	// SERVER_DATA.sgBaseUrl = "https://sg30p0.familysearch.org"
	m := regexp.MustCompile(`SERVER_DATA.sgBaseUrl\s=\s"([^"]+)"`).FindSubmatch(bs)
	if m != nil {
		return string(m[1]), nil
	}
	return "", err
}

func (r *Familysearch) download() (msg string, err error) {
	name := util.GenNumberSorted(r.dt.Index)
	log.Printf("Get %s  %s\n", name, r.dt.Url)
	r.dt.SavePath = config.CreateDirectory(r.dt.Url, r.dt.BookId)
	var canvases []string
	if r.urlType == 1 {
		canvases, err = r.getImageByGroups(r.dt.BookId)
	} else if r.urlType == 2 {
		canvases, err = r.getImageByGroupId(r.dt.BookId)
	} else if r.urlType == 3 {
		imageData, err := r.getImageData(r.dt.Url)
		if err != nil {
			return "", err
		}
		canvases, err = r.getFilmData(r.dt.Url, imageData)
	} else {
		imageData, err := r.getImageData(r.dt.Url)
		if err != nil {
			return "", err
		}
		canvases, err = r.getWaypointData(r.dt.Url, imageData)
	}
	if err != nil {
		return "", err
	}
	size := len(canvases)
	log.Printf(" %d pages.\n", size)

	r.do(canvases)
	return "", nil
}

func (r *Familysearch) do(iiifUrls []string) (msg string, err error) {
	if iiifUrls == nil {
		return
	}
	referer := url.QueryEscape(r.dt.Url)
	header, _ := curl.GetHeaderFile(config.Conf.CookieFile)
	args := []string{"--dezoomer=deepzoom",
		"-H", "authority:www.familysearch.org",
		"-H", "referer:" + referer,
		"-H", "User-Agent:" + header["User-Agent"],
		"-H", "cookie:" + header["Cookie"],
	}
	size := len(iiifUrls)
	for i, uri := range iiifUrls {
		if uri == "" || !config.PageRange(i, size) {
			continue
		}
		sortId := util.GenNumberSorted(i + 1)
		filename := sortId + config.Conf.FileExt
		dest := r.dt.SavePath + string(os.PathSeparator) + filename
		if FileExist(dest) {
			continue
		}
		log.Printf("Get %d/%d  %s\n", i+1, size, uri)
		util.StartProcess(uri, dest, args)
		util.PrintSleepTime(config.Conf.Speed)
	}
	return "", err
}

func (r *Familysearch) getVolumes(sUrl string, jar *cookiejar.Jar) (volumes []string, err error) {
	//TODO implement me
	panic("implement me")
}

func (r *Familysearch) getCanvases(sUrl string, jar *cookiejar.Jar) (canvases []string, err error) {
	panic("implement me")
}

func (r *Familysearch) getImageData(sUrl string) (imageData FamilysearchImageData, err error) {
	type ReqData struct {
		Type string `json:"type"`
		Args struct {
			ImageURL string `json:"imageURL"`
			State    struct {
				ImageOrFilmUrl     string `json:"imageOrFilmUrl"`
				ViewMode           string `json:"viewMode"`
				SelectedImageIndex int    `json:"selectedImageIndex"`
			} `json:"state"`
			Locale string `json:"locale"`
		} `json:"args"`
	}

	type Response struct {
		ImageURL string `json:"imageURL"`
		ArkId    string `json:"arkId"`
		DgsNum   string `json:"dgsNum"`
		Meta     struct {
			SourceDescriptions []struct {
				Id     string `json:"id"`
				About  string `json:"about"`
				Titles []struct {
					Value string `json:"value"`
					Lang  string `json:"lang,omitempty"`
				} `json:"titles"`
				Identifiers struct {
					HttpGedcomxOrgPrimary []string `json:"http://gedcomx.org/Primary"`
				} `json:"identifiers"`
				Descriptor struct {
					Resource string `json:"resource"`
				} `json:"descriptor,omitempty"`
			} `json:"sourceDescriptions"`
		} `json:"meta"`
	}

	var d = ReqData{}
	d.Type = "image-data"
	d.Args.ImageURL = sUrl
	d.Args.State.ImageOrFilmUrl = ""
	d.Args.State.ViewMode = "i"
	d.Args.State.SelectedImageIndex = -1
	d.Args.Locale = "zh"

	bs, err := r.postJson(r.apiUrl, d)
	if err != nil {
		return
	}
	var resultError FamilysearchResultError
	if err = json.Unmarshal(bs, &resultError); resultError.Error.StatusCode != 0 {
		msg := fmt.Sprintf("StatusCode: %d, Message: %s", resultError.Error.StatusCode, resultError.Error.Message)
		err = errors.New(msg)
		return
	}
	resp := Response{}
	if err = json.Unmarshal(bs, &resp); err != nil {
		return
	}
	imageData.DgsNum = resp.DgsNum
	imageData.ImageURL = resp.ImageURL
	for _, description := range resp.Meta.SourceDescriptions {
		if strings.Contains(description.About, "platform/records/waypoints") {
			imageData.WaypointURL = description.About
			break
		}
	}
	return imageData, nil
}

func (r *Familysearch) getWaypointData(sUrl string, imageData FamilysearchImageData) (canvases []string, err error) {
	type ReqData struct {
		Type string `json:"type"`
		Args struct {
			WaypointURL string `json:"waypointURL"`
			DgsNum      string `json:"dgsNum"`
			FilmData    string `json:"filmData"`
			State       struct {
				ImageOrFilmUrl     string `json:"imageOrFilmUrl"`
				ViewMode           string `json:"viewMode"`
				SelectedImageIndex int    `json:"selectedImageIndex"`
			} `json:"state"`
			Locale string `json:"locale"`
		} `json:"args"`
	}

	type Response struct {
		DgsNum      string   `json:"dgsNum"`
		Images      []string `json:"images"`
		Type        string   `json:"type"`
		WaypointURL string   `json:"waypointURL"`
		Templates   struct {
			DzTemplate  string `json:"dzTemplate"`
			DasTemplate string `json:"dasTemplate"`
		} `json:"templates"`
	}

	var d = ReqData{}
	d.Type = "waypoint-data"
	d.Args.WaypointURL = imageData.WaypointURL
	d.Args.State.ImageOrFilmUrl = ""
	d.Args.State.ViewMode = "i"
	d.Args.State.SelectedImageIndex = -1
	d.Args.Locale = "zh"
	bs, err := r.postJson(r.apiUrl, d)
	if err != nil {
		return
	}
	var resultError FamilysearchResultError
	if err = json.Unmarshal(bs, &resultError); resultError.Error.StatusCode != 0 {
		msg := fmt.Sprintf("StatusCode: %d, Message: %s", resultError.Error.StatusCode, resultError.Error.Message)
		err = errors.New(msg)
		return
	}
	resp := Response{}
	if err = json.Unmarshal(bs, &resp); err != nil {
		return
	}
	//https://sg30p0.familysearch.org/service/records/storage/deepzoomcloud/dz/v1/{id}/{image}
	r.dziTemplate = regexp.MustCompile(`\{[A-z]+\}`).ReplaceAllString(resp.Templates.DzTemplate, "%s")
	for _, image := range resp.Images {
		m, err := url.Parse(image)
		if err != nil {
			break
		}
		id := path.Base(m.Path)
		xmlUrl := fmt.Sprintf(r.dziTemplate, id, "image.xml")
		canvases = append(canvases, xmlUrl)

	}
	return canvases, nil
}

func (r *Familysearch) getImageByGroupId(groupId string) (canvases []string, err error) {
	text := `{"operationName":"GetGroup","variables":{"groupIdOrArkId":"` + groupId + `","detailed":true},"query":"query GetGroup($groupIdOrArkId: String!, $detailed: Boolean!) {\n  group(groupIdOrArkId: $groupIdOrArkId) {\n    volumes @include(if: $detailed)\n    changeHistory @include(if: $detailed) {\n      user {\n        contactName\n        __typename\n      }\n      type\n      opCode\n      date {\n        formatted\n        formattedFullWithTime\n        gedcomX\n        fromTimestamp\n        formattedCompactWithTime\n        formattedCompactWithHourMinute\n        __typename\n      }\n      coverages {\n        place {\n          fullName\n          name\n          id\n          __typename\n        }\n        recordTypes {\n          apolloId\n          id\n          name\n          date {\n            formatted\n            gedcomX\n            __typename\n          }\n          __typename\n        }\n        __typename\n      }\n      metadata {\n        archivalReferenceNumber\n        languages\n        title\n        volumes\n        creator\n        __typename\n      }\n      __typename\n    }\n    childCount @include(if: $detailed)\n    childNumber @include(if: $detailed)\n    childGroups(changeHistoryForFirstChildOnly: true) @include(if: $detailed) {\n      changeHistory {\n        user {\n          contactName\n          __typename\n        }\n        type\n        opCode\n        date {\n          formatted\n          formattedFullWithTime\n          gedcomX\n          fromTimestamp\n          formattedCompactWithTime\n          formattedCompactWithHourMinute\n          __typename\n        }\n        coverages {\n          place {\n            fullName\n            name\n            id\n            __typename\n          }\n          recordTypes {\n            apolloId\n            id\n            name\n            date {\n              formatted\n              gedcomX\n              __typename\n            }\n            __typename\n          }\n          __typename\n        }\n        metadata {\n          archivalReferenceNumber\n          languages\n          title\n          volumes\n          creator\n          __typename\n        }\n        __typename\n      }\n      childCount\n      id\n      imageViewerCoverages {\n        place {\n          fullName\n          name\n          id\n          __typename\n        }\n        recordTypes {\n          apolloId\n          id\n          name\n          date {\n            formatted\n            gedcomX\n            __typename\n          }\n          __typename\n        }\n        __typename\n      }\n      imageApids\n      indexedChildCount\n      internalData {\n        cameraOperator\n        cameraOperatorNumber\n        id\n        originalMediaNumber\n        phoenixAcquisitionIds\n        __typename\n      }\n      metadata {\n        archivalReferenceNumber\n        createdDate {\n          formatted\n          gedcomX\n          __typename\n        }\n        creator\n        currentDate {\n          formatted\n          gedcomX\n          __typename\n        }\n        custodian\n        imageGroupNumber\n        languages\n        place\n        title\n        volume\n        volumes\n        __typename\n      }\n      naturalGroupId\n      __typename\n    }\n    currentWorkflowActions @include(if: $detailed) {\n      name\n      status\n      parameters {\n        name\n        value\n        __typename\n      }\n      __typename\n    }\n    id @include(if: $detailed)\n    indexedChildCount @include(if: $detailed)\n    imageReprocessing @include(if: $detailed) {\n      action\n      date {\n        formattedFullWithTime\n        fromTimestamp\n        __typename\n      }\n      __typename\n    }\n    imageRework @include(if: $detailed) {\n      cameraOperator {\n        id\n        name\n        __typename\n      }\n      status {\n        status\n        type\n        __typename\n      }\n      history {\n        action\n        date {\n          formatted\n          __typename\n        }\n        state\n        username\n        notes\n        __typename\n      }\n      __typename\n    }\n    imageViewerCoverages {\n      place {\n        fullName\n        name\n        id\n        __typename\n      }\n      recordTypes {\n        apolloId\n        id\n        name\n        date {\n          formatted\n          gedcomX\n          __typename\n        }\n        __typename\n      }\n      __typename\n    }\n    imageApids\n    internalData {\n      cameraOperator\n      cameraOperatorNumber\n      id\n      originalMediaNumber\n      phoenixAcquisitionIds\n      __typename\n    }\n    metadata {\n      archivalReferenceNumber\n      createdDate {\n        formatted\n        gedcomX\n        __typename\n      }\n      creator\n      currentDate {\n        formatted\n        gedcomX\n        __typename\n      }\n      custodian\n      imageGroupNumber\n      languages\n      place\n      title\n      volume\n      __typename\n    }\n    naturalGroupId @include(if: $detailed)\n    parentGroupId @include(if: $detailed)\n    properties @include(if: $detailed) {\n      name\n      values\n      type\n      __typename\n    }\n    siblingCount @include(if: $detailed)\n    types @include(if: $detailed)\n    volumes @include(if: $detailed)\n    volumeSetAncestralHall @include(if: $detailed)\n    volumeSetFirstAncestor @include(if: $detailed)\n    volumeSetMigrantAncestor @include(if: $detailed)\n    volumeSetTitle @include(if: $detailed)\n    volumeSetTotalVolumes @include(if: $detailed)\n    __typename\n  }\n  recordTypes @include(if: $detailed) {\n    id\n    name\n    __typename\n  }\n}\n"}`
	type Response struct {
		Data struct {
			Group struct {
				ChildGroups []struct {
					ImageApids []string `json:"imageApids"`
				} `json:"childGroups"`
				ImageApids []interface{} `json:"imageApids"`
			} `json:"group"`
		} `json:"data"`
	}

	apiUrl := fmt.Sprintf("%s://%s/records/images/orchestration/", r.dt.UrlParsed.Scheme, r.dt.UrlParsed.Host)
	bs, err := r.postBody(apiUrl, []byte(text))
	if err != nil {
		return
	}

	resp := Response{}
	if err = json.Unmarshal(bs, &resp); err != nil {
		return
	}
	for _, group := range resp.Data.Group.ChildGroups {
		for _, v := range group.ImageApids {
			dzUrl := r.baseUrl + "/service/records/storage/deepzoomcloud/dz/v1/apid:" + v + "/image.xml"
			canvases = append(canvases, dzUrl)
		}
	}
	return canvases, nil
}

func (r *Familysearch) getImageByGroups(rmsId string) (canvases []string, err error) {
	type Response struct {
		Groups []struct {
			ImageUrls []string `json:"imageUrls"`
		} `json:"groups"`
	}

	apiUrl := fmt.Sprintf("%s://%s/records/images/api/imageDetails/groups/%s?properties&changeLog&coverageIndex=null",
		r.dt.UrlParsed.Scheme, r.dt.UrlParsed.Host, rmsId)
	bs, err := r.getBody(apiUrl, r.dt.Jar)
	if err != nil {
		return
	}

	resp := Response{}
	if err = json.Unmarshal(bs, &resp); err != nil {
		return
	}
	for _, group := range resp.Groups {
		for _, v := range group.ImageUrls {
			dzUrl := v + "/image.xml"
			canvases = append(canvases, dzUrl)
		}
	}
	return canvases, nil
}

func (r *Familysearch) getFilmData(sUrl string, imageData FamilysearchImageData) (canvases []string, err error) {
	type ReqData struct {
		Type string `json:"type"`
		Args struct {
			DgsNum string `json:"dgsNum"`
			State  struct {
				I                  string `json:"i"`
				Cat                string `json:"cat"`
				ImageOrFilmUrl     string `json:"imageOrFilmUrl"`
				CatalogContext     string `json:"catalogContext"`
				ViewMode           string `json:"viewMode"`
				SelectedImageIndex int    `json:"selectedImageIndex"`
			} `json:"state"`
			Locale    string `json:"locale"`
			SessionId string `json:"sessionId"`
			LoggedIn  bool   `json:"loggedIn"`
		} `json:"args"`
	}

	u, err := url.Parse(imageData.ImageURL)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	var d = ReqData{}
	d.Type = "film-data"
	d.Args.DgsNum = imageData.DgsNum
	d.Args.State.CatalogContext = q.Get("cat")
	d.Args.State.Cat = q.Get("cat")
	d.Args.State.ImageOrFilmUrl = u.Path
	d.Args.State.ViewMode = "i"
	d.Args.State.SelectedImageIndex = -1
	d.Args.Locale = "zh"
	d.Args.LoggedIn = true
	d.Args.SessionId = r.getSessionId()

	type Response struct {
		DgsNum             string      `json:"dgsNum"`
		Images             []string    `json:"images"`
		PreferredCatalogId string      `json:"preferredCatalogId"`
		Type               string      `json:"type"`
		WaypointCrumbs     interface{} `json:"waypointCrumbs"`
		WaypointURL        interface{} `json:"waypointURL"`
		Templates          struct {
			DasTemplate string `json:"dasTemplate"`
			DzTemplate  string `json:"dzTemplate"`
		} `json:"templates"`
	}
	bs, err := r.postJson(r.apiUrl, d)
	if err != nil {
		return
	}
	var resultError FamilysearchResultError
	if err = json.Unmarshal(bs, &resultError); resultError.Error.StatusCode != 0 {
		msg := fmt.Sprintf("StatusCode: %d, Message: %s", resultError.Error.StatusCode, resultError.Error.Message)
		err = errors.New(msg)
		return
	}
	resp := Response{}
	if err = json.Unmarshal(bs, &resp); err != nil {
		return
	}
	//https://sg30p0.familysearch.org/service/records/storage/deepzoomcloud/dz/v1/{id}/{image}
	r.dziTemplate = regexp.MustCompile(`\{[A-z]+\}`).ReplaceAllString(resp.Templates.DzTemplate, "%s")
	for _, image := range resp.Images {
		//https://familysearch.org/ark:/61903/3:1:3QSQ-G9MC-ZSQ7-3/image.xml
		m := regexp.MustCompile(`(?i)ark:/(?:[A-z0-9-_:]+)/([A-z\d-_:]+)/image.xml`).FindStringSubmatch(image)
		if m == nil {
			continue
		}
		xmlUrl := fmt.Sprintf(r.dziTemplate, m[1], "image.xml")
		canvases = append(canvases, xmlUrl)

	}
	return canvases, err
}

func (r *Familysearch) postJson(sUrl string, d interface{}) ([]byte, error) {
	ctx := context.Background()
	cli := gohttp.NewClient(ctx, gohttp.Options{
		CookieFile: config.Conf.CookieFile,
		CookieJar:  r.dt.Jar,
		Headers: map[string]interface{}{
			"User-Agent":   config.Conf.UserAgent,
			"Content-Type": "application/json",
			"authority":    "www.familysearch.org",
			"origin":       "https://www.familysearch.org",
			"referer":      r.dt.Url,
		},
		JSON: d,
	})
	resp, err := cli.Post(sUrl)
	if err != nil {
		return nil, err
	}
	bs, _ := resp.GetBody()
	return bs, err
}

func (r *Familysearch) getSessionId() string {
	header, _ := curl.GetHeaderFile(config.Conf.CookieFile)
	//fssessionid=e10ce618-f7f7-45de-b2c3-d1a31d080d58-prod;
	m := regexp.MustCompile(`fssessionid=([^;]+);`).FindStringSubmatch(header["Cookie"])
	if m != nil {
		return "bearer " + m[1]
	}
	return ""
}

func (r *Familysearch) postBody(sUrl string, d []byte) ([]byte, error) {
	sid := r.getSessionId()
	ctx := context.Background()
	cli := gohttp.NewClient(ctx, gohttp.Options{
		CookieFile: config.Conf.CookieFile,
		CookieJar:  r.dt.Jar,
		Headers: map[string]interface{}{
			"User-Agent":    config.Conf.UserAgent,
			"Content-Type":  "application/json",
			"authority":     "www.familysearch.org",
			"origin":        "https://www.familysearch.org",
			"authorization": sid,
			"referer":       r.dt.Url,
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

func (r *Familysearch) getBody(apiUrl string, jar *cookiejar.Jar) ([]byte, error) {
	ctx := context.Background()
	cli := gohttp.NewClient(ctx, gohttp.Options{
		CookieFile: config.Conf.CookieFile,
		CookieJar:  jar,
		Headers: map[string]interface{}{
			"User-Agent":   config.Conf.UserAgent,
			"Content-Type": "application/json",
			"authority":    "www.familysearch.org",
			"origin":       "https://www.familysearch.org",
			"referer":      r.dt.Url,
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
