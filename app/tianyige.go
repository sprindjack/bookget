package app

import (
	"bookget/config"
	"bookget/pkg/gohttp"
	xhash "bookget/pkg/hash"
	"bookget/pkg/util"
	"bytes"
	"context"
	"crypto/aes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/andreburgaud/crypt2go/ecb"
	"github.com/andreburgaud/crypt2go/padding"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
	"io"
	"log"
	"math/rand"
	"net/http/cookiejar"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// APP_ID & APP_KEY form https://gj.tianyige.com.cn/js/2.f75e590e.chunk.js
const TIANYIGE_ID = "4f65a2a8247f400c8c29474bf707d680"
const TIANYIGE_KEY = "G3HT5CX8FTG5GWGUUJX8B5SWJTXS1KRC"

type TyeResponseVolume struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data []TygVolume `json:"data"`
}
type TygPageImage struct {
	Records     []TygImageRecord `json:"records"`
	Total       int              `json:"total"`
	Size        int              `json:"size"`
	Current     int              `json:"current"`
	SearchCount bool             `json:"searchCount"`
	Pages       int              `json:"pages"`
}
type TygImageRecord struct {
	ImageId     string      `json:"imageId"`
	ImageName   string      `json:"imageName"`
	DirectoryId string      `json:"directoryId"`
	FascicleId  string      `json:"fascicleId"`
	CatalogId   string      `json:"catalogId"`
	Sort        int         `json:"sort"`
	Type        int         `json:"type"`
	IsParse     interface{} `json:"isParse"`
	Description interface{} `json:"description"`
	Creator     string      `json:"creator"`
	CreateTime  string      `json:"createTime"`
	Updator     string      `json:"updator"`
	UpdateTime  string      `json:"updateTime"`
	IsDeleted   int         `json:"isDeleted"`
	OcrInfo     interface{} `json:"ocrInfo"`
	File        interface{} `json:"file"`
}

// 页面
type TygResponsePage struct {
	Code int          `json:"code"`
	Msg  string       `json:"msg"`
	Data TygPageImage `json:"data"`
}

type TygResponseFile struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		File []struct {
			FileName    string `json:"fileName"`
			FileSuffix  string `json:"fileSuffix"`
			FilePath    string `json:"filePath"`
			UpdateTime  string `json:"updateTime"`
			Sort        string `json:"sort"`
			CreateTime  string `json:"createTime"`
			FileSize    int    `json:"fileSize"`
			FileOldname string `json:"fileOldname"`
			FileInfoId  string `json:"fileInfoId"`
		} `json:"file"`
	} `json:"data"`
}

type TygVolume struct {
	FascicleId   string      `json:"fascicleId"`
	CatalogId    string      `json:"catalogId"`
	Name         string      `json:"name"`
	Introduction interface{} `json:"introduction"`
	GradeId      string      `json:"gradeId"`
	Sort         int         `json:"sort"`
	Creator      interface{} `json:"creator"`
	CreateTime   string      `json:"createTime"`
	Updator      interface{} `json:"updator"`
	UpdateTime   string      `json:"updateTime"`
	IsDeleted    int         `json:"isDeleted"`
	FilePath     interface{} `json:"filePath"`
	ImageCount   interface{} `json:"imageCount"`
}

type TygCatalog struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		Records []struct {
			DirectoryId string      `json:"directoryId"`
			FascicleId  string      `json:"fascicleId"`
			CatalogId   string      `json:"catalogId"`
			Name        string      `json:"name"`
			Description interface{} `json:"description"`
			PageId      string      `json:"pageId"`
			GradeId     string      `json:"gradeId"`
			Region      string      `json:"region"`
			Sort        int         `json:"sort"`
			Creator     interface{} `json:"creator"`
			CreateTime  string      `json:"createTime"`
			Updator     interface{} `json:"updator"`
			UpdateTime  *string     `json:"updateTime"`
			IsDeleted   int         `json:"isDeleted"`
		} `json:"records"`
		Total       int  `json:"total"`
		Size        int  `json:"size"`
		Current     int  `json:"current"`
		SearchCount bool `json:"searchCount"`
		Pages       int  `json:"pages"`
	} `json:"data"`
}

type TygParts map[string][]TygImageRecord

type Tianyige struct {
	dt           *DownloadTask
	index        int
	localStorage struct {
		authorization  string
		authorizationu string
	}
}

func (p *Tianyige) Init(iTask int, sUrl string) (msg string, err error) {
	p.dt = new(DownloadTask)
	p.dt.UrlParsed, err = url.Parse(sUrl)
	p.dt.Url = sUrl
	p.dt.Index = iTask
	p.dt.BookId = p.getBookId(p.dt.Url)
	if p.dt.BookId == "" {
		return "requested URL was not found.", err
	}
	p.dt.Jar, _ = cookiejar.New(nil)
	//OpenWebBrowser(sUrl, []string{})
	p.localStorage.authorization, p.localStorage.authorizationu, err = p.getLocalStorage()
	if err != nil {
		return "Invalid file localStorage.txt", err
	}
	return p.download()
}

func (p *Tianyige) getBookId(sUrl string) (bookId string) {
	m := regexp.MustCompile(`(?i)searchpage/([A-z0-9_-]+)`).FindStringSubmatch(sUrl)
	if m != nil {
		bookId = m[1]
	} else {
		m = regexp.MustCompile(`(?i)catalogid=([A-z0-9_-]+)`).FindStringSubmatch(sUrl)
		if m != nil {
			bookId = m[1]
		}
	}
	return bookId
}

func (p *Tianyige) download() (msg string, err error) {
	respVolume, err := p.getVolumes(p.dt.BookId, p.dt.Jar)
	if err != nil {
		fmt.Println(err)
		return "getVolumes", err
	}
	canvases, err := p.getCanvases(p.dt.BookId, p.dt.Jar)
	if err != nil {
		log.Println(err)
		return
	}
	log.Printf(" %d volumes,  %d pages.\n", len(respVolume), len(canvases))
	parts := make(TygParts)
	for _, record := range canvases {
		parts[record.FascicleId] = append(parts[record.FascicleId], record)
	}
	var bookmark = "#版本=1.0\r\n"
	sizeVol := len(respVolume)
	for i, vol := range respVolume {
		if !config.VolumeRange(i) {
			continue
		}
		vid := util.GenNumberSorted(i + 1)
		p.dt.SavePath = CreateDirectory(p.dt.UrlParsed.Host, p.dt.BookId, vid)
		sizePage := len(parts[vol.FascicleId])
		log.Printf(" %d/%d volume, %d pages \n", i+1, sizeVol, sizePage)
		text, err := p.getCatalogById(vol.CatalogId, vol.FascicleId, p.index)
		if err == nil {
			bookmark += text
		}
		p.do(parts[vol.FascicleId])
	}

	savePath := CreateDirectory(p.dt.UrlParsed.Host, p.dt.BookId, "")
	data, _ := io.ReadAll(transform.NewReader(bytes.NewReader([]byte(bookmark)), simplifiedchinese.GBK.NewEncoder()))
	_ = os.WriteFile(savePath+"bookmark.txt", []byte(bookmark), os.ModePerm)
	_ = os.WriteFile(savePath+"bookmark_gbk.txt", data, os.ModePerm)
	return msg, err
}

func (p *Tianyige) do(records []TygImageRecord) (msg string, err error) {
	if records == nil {
		return "", nil
	}
	size := len(records)
	fmt.Println()
	var wg sync.WaitGroup
	idDict := make(map[string]string, 1000)
	i := 0
	for _, record := range records {
		uri, _, err := p.getImageById(record.ImageId)
		if err != nil || uri == "" || !config.PageRange(i, size) {
			continue
		}
		i++
		p.index++
		sortId := util.GenNumberSorted(i)
		filename := sortId + config.Conf.FileExt
		dest := p.dt.SavePath + filename
		if config.Conf.Bookmark || FileExist(dest) {
			continue
		}
		log.Printf("Get %d/%d  %s\n", i, size, uri)
		//下载时有验证码
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
		for k := 0; k < 10; k++ {
			_, err = gohttp.FastGet(ctx, uri, opts)
			if err == nil && FileExist(dest) {
				break
			}
			WaitNewCookieWithMsg(uri)
		}

		bs, _ := os.ReadFile(dest)
		mh := xhash.NewMultiHasher()
		_, _ = io.Copy(mh, bytes.NewBuffer(bs))
		kId, _ := mh.SumString(xhash.MD5, false)
		_, ok := idDict[kId]
		if ok {
			fmt.Println()
			continue
		} else {
			idDict[kId] = uri
		}
		util.PrintSleepTime(config.Conf.Speed)
		fmt.Println()
	}
	wg.Wait()
	fmt.Println()
	return "", err
}

func (p *Tianyige) getVolumes(catalogId string, jar *cookiejar.Jar) (volumes []TygVolume, err error) {
	apiUrl := fmt.Sprintf("https://%s/g/sw-anb/api/getFasciclesByCataId?catalogId=%s", p.dt.UrlParsed.Host, catalogId)
	bs, err := p.getBody(apiUrl, jar)
	if bs == nil || err != nil {
		return nil, err
	}
	resObj := new(TyeResponseVolume)
	if err = json.Unmarshal(bs, resObj); err != nil {
		return nil, err
	}
	volumes = make([]TygVolume, len(resObj.Data))
	copy(volumes, resObj.Data)
	return volumes, err
}

func (p *Tianyige) getCanvases(bookId string, jar *cookiejar.Jar) (canvases []TygImageRecord, err error) {
	for i := 1; i < 100; i++ {
		apiUrl := fmt.Sprintf("https://%s/g/sw-anb/api/queryImageByCatalog?catalogId=%s", p.dt.UrlParsed.Host, bookId)
		d := fmt.Sprintf(`{"param":{"pageNum":%d,"pageSize":999}}`, i)
		bs, err := p.postBody(apiUrl, []byte(d), jar)
		if bs == nil || err != nil {
			break
		}
		var resObj TygResponsePage
		if err = json.Unmarshal(bs, &resObj); resObj.Code != 200 {
			break
		}
		if resObj.Data.Total == len(canvases) {
			break
		}
		records := make([]TygImageRecord, len(resObj.Data.Records))
		copy(records, resObj.Data.Records)
		canvases = append(canvases, records...)
	}
	return canvases, err
}

func (p *Tianyige) getImageById(imageId string) (imgUrl, ocrUrl string, err error) {
	apiUrl := fmt.Sprintf("https://%s/g/sw-anb/api/queryOcrFileByimageId?imageId=%s", p.dt.UrlParsed.Host, imageId)
	var bs []byte
	for i := 0; i < 3; i++ {
		bs, err = p.getBody(apiUrl, p.dt.Jar)
		if bs != nil {
			break
		}
	}
	if err != nil {
		return
	}
	var resObj TygResponseFile
	if err = json.Unmarshal(bs, &resObj); err != nil {
		fmt.Println(err)
		return
	}

	for _, ossFile := range resObj.Data.File {
		if strings.Contains(ossFile.FileOldname, "_c") {
			ocrUrl = fmt.Sprintf("https://%s/fileUpload/%s/%s", p.dt.UrlParsed.Host, ossFile.FilePath, ossFile.FileName)
		} else {
			imgUrl = fmt.Sprintf("https://%s/fileUpload/%s/%s", p.dt.UrlParsed.Host, ossFile.FilePath, ossFile.FileName)
		}
	}
	return
}

func (p *Tianyige) getCatalogById(catalogId, fascicleId string, indexStart int) (string, error) {
	apiUrl := fmt.Sprintf("https://%s/g/sw-anb/api/getDirectorys?catalogId=%s&fascicleId=%s&directoryName=", p.dt.UrlParsed.Host, catalogId, fascicleId)
	bs, err := p.getBody(apiUrl, p.dt.Jar)
	if err != nil {
		return "", err
	}
	var resp TygCatalog
	if err = json.Unmarshal(bs, &resp); err != nil {
		fmt.Println(err)
		return "", err
	}
	var bookmark string
	for _, record := range resp.Data.Records {
		m := regexp.MustCompile(`(\d+).jpg`).FindStringSubmatch(record.PageId)
		if m != nil {
			page, _ := strconv.Atoi(m[1])
			page = indexStart + page
			if os.PathSeparator == '\\' {
				bookmark += fmt.Sprintf("%s......%d\r\n", record.Name, page)
			} else {
				bookmark += fmt.Sprintf("%s......%d\n", record.Name, page)
			}
		}
	}
	return bookmark, err
}

func (p *Tianyige) getBody(sUrl string, jar *cookiejar.Jar) ([]byte, error) {
	ctx := context.Background()
	token := p.getToken()
	cli := gohttp.NewClient(ctx, gohttp.Options{
		CookieFile: config.Conf.CookieFile,
		CookieJar:  jar,
		Headers: map[string]interface{}{
			"User-Agent":     config.Conf.UserAgent,
			"Content-Type":   "application/json;charset=UTF-8",
			"token":          token,
			"appId":          TIANYIGE_ID,
			"authorization":  p.localStorage.authorization,
			"authorizationu": p.localStorage.authorizationu,
		},
	})
	resp, err := cli.Get(sUrl)
	if err != nil {
		return nil, err
	}
	bs, _ := resp.GetBody()
	if bs == nil || resp.GetStatusCode() != 200 {
		msg := fmt.Sprintf("Please try again later.[%d %s]\n", resp.GetStatusCode(), resp.GetReasonPhrase())
		fmt.Println(msg)
		return nil, errors.New(msg)
	}
	return bs, err
}

func (p *Tianyige) postBody(sUrl string, d []byte, jar *cookiejar.Jar) ([]byte, error) {
	token := p.getToken()
	ctx := context.Background()
	cli := gohttp.NewClient(ctx, gohttp.Options{
		CookieFile: config.Conf.CookieFile,
		CookieJar:  jar,
		Headers: map[string]interface{}{
			"User-Agent":     config.Conf.UserAgent,
			"Content-Type":   "application/json;charset=UTF-8",
			"token":          token,
			"appId":          TIANYIGE_ID,
			"authorization":  p.localStorage.authorization,
			"authorizationu": p.localStorage.authorizationu,
		},
		Body: d,
	})
	resp, err := cli.Post(sUrl)
	if err != nil {
		return nil, err
	}
	bs, _ := resp.GetBody()
	if bs == nil || resp.GetStatusCode() != 200 {
		msg := fmt.Sprintf("Please try again later.[%d %s]\n", resp.GetStatusCode(), resp.GetReasonPhrase())
		fmt.Println(msg)
		return nil, errors.New(msg)
	}
	return bs, err
}

func (p *Tianyige) encrypt(pt, key []byte) string {
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err.Error())
	}
	mode := ecb.NewECBEncrypter(block)
	padder := padding.NewPkcs7Padding(mode.BlockSize())
	pt, err = padder.Pad(pt) // padd last block of plaintext if block size less than block cipher size
	if err != nil {
		panic(err.Error())
	}
	ct := make([]byte, len(pt))
	mode.CryptBlocks(ct, pt)
	return base64.StdEncoding.EncodeToString(ct)
}

func (p *Tianyige) getToken() string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	//pt := []byte(strconv.Itoa(r.Intn(900000)+100000) + strconv.FormatInt(time.Now().UnixMilli(), 10))
	pt := []byte(fmt.Sprintf("%.6d%d", r.Int31()%10000, time.Now().UnixMilli()))
	// Key size for AES is either: 16 bytes (128 bits), 24 bytes (192 bits) or 32 bytes (256 bits)
	key := []byte(TIANYIGE_KEY)
	return p.encrypt(pt, key)
}

// // 假设 LocalStorage 中已经有 'authorization' 和 'authorizationu' 这两个键
// const authorization = localStorage.getItem('authorization');
// const authorizationu = localStorage.getItem('authorizationu');
func (p *Tianyige) getLocalStorage() (string, string, error) {
	bs, err := os.ReadFile(config.Conf.LocalStorage)
	if bs == nil || err != nil {
		return "", "", err
	}

	// 分割输入字符串为多个部分，以换行符为分隔符
	lines := strings.Split(string(bs), "\n")

	authTokens := make(map[string]string)
	for _, line := range lines {
		// 去除行首和行尾的空格
		line = strings.TrimSpace(line)

		// 如果行是空的，则跳过
		if line == "" {
			continue
		}

		// 分割键和值，以冒号为分隔符，并去除键和值两侧的空格
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			return "", "", fmt.Errorf("invalid line format: %s", line)
		}
		key := strings.TrimSpace(parts[0])
		value := strings.Trim(strings.Trim(strings.Trim(parts[1], "\""), " "), "'")

		// 将键值对存储到 map 中
		authTokens[key] = value
	}

	// 从 map 中提取 authorization 和 authorizationu 的值
	authorization, ok1 := authTokens["authorization"]
	authorizationu, ok2 := authTokens["authorizationu"]

	// 检查是否成功提取到所有需要的值
	if !ok1 || !ok2 {
		return "", "", fmt.Errorf("missing required token")
	}
	return authorization, authorizationu, nil
}
