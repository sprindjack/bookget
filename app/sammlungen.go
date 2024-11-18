package app

import (
	"bookget/lib/util"
	"fmt"
	"log"
	"net/http/cookiejar"
	"net/url"
	"regexp"
)

type Sammlungen struct {
	dt *DownloadTask
}

func (r *Sammlungen) Init(iTask int, sUrl string) (msg string, err error) {
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

func (r *Sammlungen) getBookId(sUrl string) (bookId string) {
	m := regexp.MustCompile(`/view/([A-z\d]+)`).FindStringSubmatch(sUrl)
	if m != nil {
		bookId = m[1]
	}
	return bookId
}

func (r *Sammlungen) download() (msg string, err error) {
	name := util.GenNumberSorted(r.dt.Index)
	log.Printf("Get %s  %s\n", name, r.dt.Url)
	manifestUrl := fmt.Sprintf("https://api.digitale-sammlungen.de/iiif/presentation/v2/%s/manifest", r.dt.BookId)
	var iiif IIIF
	return iiif.InitWithId(r.dt.Index, manifestUrl, r.dt.BookId)
}

func (r *Sammlungen) do(imgUrls []string) (msg string, err error) {
	//TODO implement me
	panic("implement me")
}

func (r *Sammlungen) getVolumes(sUrl string, jar *cookiejar.Jar) (volumes []string, err error) {
	//TODO implement me
	panic("implement me")
}

func (r *Sammlungen) getCanvases(sUrl string, jar *cookiejar.Jar) (canvases []string, err error) {
	//TODO implement me
	panic("implement me")
}
