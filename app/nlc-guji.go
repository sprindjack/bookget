package app

import (
	"bookget/config"
	"bookget/model/nlc"
	"bookget/pkg/downloader"
	"bookget/pkg/util"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"time"
)

type NlcGuji struct {
	dm     *downloader.DownloadManager
	ctx    context.Context
	cancel context.CancelFunc
	client *http.Client

	rawUrl    string
	parsedUrl *url.URL
	savePath  string
	bookId    string
}

func NewNlcGuji(sUrl string) *NlcGuji {
	ctx, cancel := context.WithCancel(context.Background())

	dm := downloader.NewDownloadManager(config.Conf.MaxConcurrent)
	parsedUrl, _ := url.Parse(sUrl)

	// 创建自定义 Transport 忽略 SSL 验证
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	jar, _ := cookiejar.New(nil)

	return &NlcGuji{
		dm:        dm,
		rawUrl:    sUrl,
		parsedUrl: parsedUrl,
		client:    &http.Client{Timeout: 30 * time.Second, Jar: jar, Transport: tr},
		ctx:       ctx,
		cancel:    cancel,
	}
}

func (s *NlcGuji) getBookId() (bookId string) {
	const (
		metadataIdPattern = `(?i)metadataId=([A-Za-z0-9_-]+)`
		idPattern         = `(?i)\?id=([A-Za-z0-9_-]+)`
	)

	// 预编译正则表达式
	var (
		metadataIdRe = regexp.MustCompile(metadataIdPattern)
		idRe         = regexp.MustCompile(idPattern)
	)

	// 优先尝试匹配 metadataId
	if matches := metadataIdRe.FindStringSubmatch(s.rawUrl); matches != nil && len(matches) > 1 {
		return matches[1]
	}

	// 然后尝试匹配 id
	if matches := idRe.FindStringSubmatch(s.rawUrl); matches != nil && len(matches) > 1 {
		return matches[1]
	}

	return "" // 明确返回空字符串表示未找到
}

func (s *NlcGuji) Start() (msg string, err error) {
	s.bookId = s.getBookId()
	if s.bookId == "" {
		return "[err=getBookId]", err
	}

	canvases, err := s.getCanvases()
	if err != nil || canvases == nil {
		return "[err=getCanvases]", err

	}
	s.letsGo(canvases)
	return "", nil
}

func (s *NlcGuji) letsGo(canvases []nlc.DataItem) (msg string, err error) {
	sizeVol := len(canvases)
	if sizeVol <= 0 {
		return "[err=letsGo]", err
	}
	s.savePath = CreateDirectory(s.parsedUrl.Host, s.bookId, "")

	imgServer := fmt.Sprintf("https://%s/api/common/jpgViewer?ftpId=1&filePathName=", s.parsedUrl.Host)

	s.dm.SetBar(sizeVol)
	for i, item := range canvases {
		//https://guji.nlc.cn/api/anc/ancImageAndContent?metadataId=1001165&structureId=1014544&imageId=2075393
		apiUrl := fmt.Sprintf("https://%s/api/anc/ancImageAndContent?metadataId=%s&structureId=%d&imageId=%s",
			s.parsedUrl.Host, s.bookId, item.StructureId, item.ImageId)
		//metadataId=1001165&structureId=1014544&imageId=2075393
		rawData := []byte(fmt.Sprintf("metadataId=%s&structureId=%d&imageId=%s", s.bookId, item.StructureId, item.ImageId))
		bs, err := s.postBody(apiUrl, rawData)
		if err != nil {
			return "[err=letsGo]", err
		}
		var resp nlc.ImageData
		if err = json.Unmarshal(bs, &resp); err != nil {
			return "[err=letsGo::Unmarshal]", err
		}
		i++
		imgUrl := imgServer + resp.Data.FilePath
		//fileName := util.GenNumberSorted(item.PageNum) + config.Conf.FileExt
		sortId := util.GenNumberSorted(i)
		fileName := sortId + config.Conf.FileExt

		//跳过存在的文件
		if FileExist(s.savePath + fileName) {
			continue
		}

		fmt.Printf("准备中 %d/%d\r", i, sizeVol)

		// 添加GET下载任务
		s.dm.AddTask(
			imgUrl,
			"GET",
			map[string]string{"User-Agent": config.Conf.UserAgent},
			nil,
			s.savePath,
			fileName,
			int(config.Conf.Threads),
		)
	}
	fmt.Println()
	s.dm.Start()
	return "", nil
}

func (s *NlcGuji) getCanvases() (canvases []nlc.DataItem, err error) {

	apiUrl := fmt.Sprintf("https://%s/api/anc/ancImageIdListWithPageNum?metadataId=%s", s.parsedUrl.Host, s.bookId)
	rawData := []byte("metadataId=" + s.bookId)
	bs, err := s.postBody(apiUrl, rawData)
	if err != nil {
		return canvases, err
	}
	resp := new(nlc.BaseResponse)
	if err = json.Unmarshal(bs, &resp); err != nil {
		return canvases, err
	}
	// 打印结果
	for _, item := range resp.Data.ImageIdList {
		canvases = append(canvases, item)
	}
	return canvases, nil
}

func (s *NlcGuji) getBody(sUrl string) ([]byte, error) {
	req, err := http.NewRequest("GET", sUrl, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", config.Conf.UserAgent)
	resp, err := s.client.Do(req.WithContext(s.ctx))
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("close body err=%v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		err = fmt.Errorf("服务器返回错误状态码: %d", resp.StatusCode)
		return nil, err
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func (s *NlcGuji) postBody(sUrl string, postData []byte) ([]byte, error) {
	req, err := http.NewRequest("POST", sUrl, bytes.NewBuffer(postData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", config.Conf.UserAgent)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req.WithContext(s.ctx))
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("close body err=%v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		err = fmt.Errorf("服务器返回错误状态码: %d", resp.StatusCode)
		return nil, err
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}
