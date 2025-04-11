package idp

import (
	"net/http/cookiejar"
	"net/url"
)

type DownloadTask struct {
	Index     int
	Url       string
	UrlParsed *url.URL
	BookId    string
	CookieJar *cookiejar.Jar
}

type Canvases struct {
	ImgUrls []string
	Size    int
}
