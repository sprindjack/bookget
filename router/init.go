package router

import (
	"bookget/config"
	"bookget/pkg/util"
	"crypto/tls"
	"errors"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// RouterInit ...
type RouterInit interface {
	getRouterInit(sUrl []string) (map[string]interface{}, error)
}

var doInit sync.Once
var Router = make(map[string]RouterInit)

// FactoryRouter ...
// siteID: 站点ID
// sUrl: 网址
func FactoryRouter(siteID string, sUrl []string) (map[string]interface{}, error) {
	if config.Conf.AutoDetect == 1 {
		siteID = "bookget"
	} else if config.Conf.AutoDetect == 2 || strings.Contains(sUrl[0], ".json") {
		siteID = "iiif.io"
	}
	if strings.Contains(sUrl[0], "tiles/infos.json") {
		siteID = "dzicnlib"
	}
	doInit.Do(func() {
		//{{{ ---------------中国--------------------------------------------------
		//[中国]国家图书馆
		Router["read.nlc.cn"] = new(ChinaNcl)
		Router["mylib.nlc.cn"] = new(ChinaNcl)
		//[中国]臺灣華文電子書庫
		Router["taiwanebook.ncl.edu.tw"] = new(Huawen)
		//[中国]香港中文大学图书馆cuhk.Init
		Router["repository.pkg.cuhk.edu.hk"] = new(CuHk)
		//[中国]香港科技大学图书馆 usthk.Init
		Router["lbezone.hkust.edu.hk"] = new(UstHk)
		//[中国]洛阳市图书馆
		Router["111.7.82.29:8090"] = new(LuoYang)
		//[中国]温州市图书馆
		Router["oyjy.wzlib.cn"] = new(Wzlib)
		Router["arcgxhpv7cw0.db.wzlib.cn"] = new(Wzlib)
		//[中国]深圳市图书馆-古籍
		Router["yun.szlib.org.cn"] = new(YunSzlib)
		//[中国]广州大典
		Router["gzdd.gzlib.gov.cn"] = new(GzddGzlib)
		Router["gzdd.gzlib.org.cn"] = new(GzddGzlib)

		//[中国]天一阁博物院古籍数字化平台
		Router["gj.tianyige.com.cn"] = new(TianYiGeLib)
		//[中国]江苏高校珍贵古籍数字图书馆
		Router["jsgxgj.nju.edu.cn"] = new(Njuedu)
		//北京故宫博物院-故宫名画记
		//Router["minghuaji.dpm.org.cn"] = new(MinghuajiBjDpm)
		//Router["m-minghuaji.dpm.org.cn"] = new(MinghuajiBjDpm)
		//Router["digicol.dpm.org.cn"] = new(MinghuajiBjDpm)
		//[中国]中华寻根网-国图
		Router["ouroots.nlc.cn"] = new(OurootsNlc)
		//[中国]国家哲学社会科学文献中心
		Router["www.ncpssd.org"] = new(Ncpssd)
		Router["www.ncpssd.cn"] = new(Ncpssd)
		//[中国]山东中医药大学古籍数字图书馆
		Router["gjsztsg.sdutcm.edu.cn"] = new(Sdutcm)
		//[中国]天津图书馆历史文献数字资源库
		Router["lswx.tjl.tj.cn:8001"] = new(Tjliblswx)
		//[中国]云南数字方志馆
		Router["dfz.yn.gov.cn"] = new(Yndfz)
		//[中国]香港大学数字图书
		Router["digitalrepository.pkg.hku.hk"] = new(Hkulib)
		//[中国]山东省诸城市图书馆
		Router["124.134.220.209:8100"] = new(ZhuCheng)
		Router["dlibgate.cafa.edu.cn"] = new(CafaEdu)
		Router["dlib.cafa.edu.cn"] = new(CafaEdu)

		//抗日战争与中日关系文献数据平台
		Router["www.modernhistory.org.cn"] = new(War1931)
		//}}} -----------------------------------------------------------------

		//---------------日本--------------------------------------------------
		//[日本]京都大学图书馆 rmda.kulib.kyoto-u.ac.jp 自动检测

		//[日本]国立国会图书馆
		Router["dl.ndl.go.jp"] = new(NdlGo)
		//[日本]E国宝eMuseum
		Router["emuseum.nich.go.jp"] = new(EmuseumNich)
		//[日本]宫内厅书陵部（汉籍集览）
		Router["db2.sido.keio.ac.jp"] = new(SidoKeio)
		//[日本]东京大学东洋文化研究所（汉籍善本资料库）
		Router["shanben.ioc.u-tokyo.ac.jp"] = new(ShanbenuTokyo)
		//[日本]国立公文书馆（内阁文库）
		Router["www.digital.archives.go.jp"] = new(Nationaljp)
		//[日本]东洋文库
		Router["dsr.nii.ac.jp"] = new(DsrNiiAc)
		//[日本]早稻田大学图书馆
		Router["archive.wul.waseda.ac.jp"] = new(Waseda)
		//[日本]国書数据库（古典籍）
		Router["kokusho.nijl.ac.jp"] = new(KokushoNijlAc)
		//[日本]京都大学人文科学研究所 东方学数字图书博物馆
		Router["kanji.zinbun.kyoto-u.ac.jp"] = new(Kyotou)

		//[日本]駒澤大学 电子贵重书库
		Router["repo.komazawa-u.ac.jp"] = new(NormalIIIF)
		//[日本]关西大学图书馆
		Router["www.iiif.ku-orcas.kansai-u.ac.jp"] = new(NormalIIIF)
		//[日本]庆应义塾大学图书馆
		Router["dcollections.pkg.keio.ac.jp"] = new(NormalIIIF)

		//[日本]大阪府立圖書館 IIIF自動檢測

		//[日本]国立历史民俗博物馆
		Router["khirin-a.rekihaku.ac.jp"] = new(KhirinRekihaku)
		//[日本]市立米泽图书馆
		Router["www.library.yonezawa.yamagata.jp"] = new(LibYonezawa)
		Router["webarchives.tnm.jp"] = new(WebarchivesTnm)
		//[日本]龙谷大学
		Router["da.library.ryukoku.ac.jp"] = new(Ryukoku)
		//}}} -----------------------------------------------------------------

		//{{{---------------美国、欧洲--------------------------------------------------
		//[美国]哈佛大学图书馆
		Router["iiif.pkg.harvard.edu"] = new(Harvard)
		Router["listview.pkg.harvard.edu"] = new(Harvard)
		Router["curiosity.pkg.harvard.edu"] = new(Harvard)
		//[美国]hathitrust 数字图书馆
		Router["babel.hathitrust.org"] = new(Hathitrust)
		//[美国]普林斯顿大学图书馆
		Router["catalog.princeton.edu"] = new(Princeton)
		Router["dpul.princeton.edu"] = new(Princeton)
		//[美国]国会图书馆
		Router["www.loc.gov"] = new(UsLoc)
		//[美国]斯坦福大学图书馆
		Router["searchworks.stanford.edu"] = new(SearchworksStanford)
		//[美国]犹他州家谱
		Router["www.familysearch.org"] = new(FamilySearch)
		//[德国]柏林国立图书馆
		Router["digital.staatsbibliothek-berlin.de"] = new(DigitalBerlin)
		//[德国]巴伐利亞州立圖書館東亞數字資源庫
		Router["ostasien.digitale-sammlungen.de"] = new(Sammlungen)
		Router["www.digitale-sammlungen.de"] = new(Sammlungen)
		//[英国]牛津大学博德利图书馆
		Router["digital.bodleian.ox.ac.uk"] = new(OxacUk)
		//[英国]图书馆文本手稿
		Router["www.bl.uk"] = new(BlUk)
		//Smithsonian Institution
		Router["ids.si.edu"] = new(SiEdu)
		Router["www.si.edu"] = new(SiEdu)
		Router["iiif.si.edu"] = new(SiEdu)
		Router["asia.si.edu"] = new(SiEdu)
		//[美國]柏克萊加州大學東亞圖書館
		Router["digicoll.pkg.berkeley.edu"] = new(Berkeley)
		//奥地利国图
		Router["digital.onb.ac.at"] = new(OnbDigital)

		//}}} -----------------------------------------------------------------

		//{{{---------------其它--------------------------------------------------
		//國際敦煌項目
		Router["idp.nlc.cn"] = new(IDP)
		Router["idp.bl.uk"] = new(IDP)
		Router["idp.orientalstudies.ru"] = new(IDP)
		Router["idp.afc.ryukoku.ac.jp"] = new(IDP)
		Router["idp.bbaw.de"] = new(IDP)
		Router["idp.bnf.fr"] = new(IDP)
		Router["idp.korea.ac.kr"] = new(IDP)

		//[韩国]
		Router["kyudb.snu.ac.kr"] = new(KyudbSnu)
		Router["sillok.history.go.kr"] = new(Sillokgokr)
		Router["lod.nl.go.kr"] = new(DlibGoKr)
		//高丽大学
		Router["kostma.korea.ac.kr"] = new(Korea)

		//俄罗斯图书馆
		Router["viewer.rsl.ru"] = new(RslRu)
		//越南汉喃古籍文献典藏数位计划
		Router["pkg.nomfoundation.org"] = new(Nomfoundation)
		//越南国家图书馆汉农图书馆
		Router["hannom.nlv.gov.vn"] = new(HannomNlv)

		//}}} -----------------------------------------------------------------
		Router["iiif.io"] = new(NormalIIIF)
		Router["bookget"] = new(NormalHttp)
		Router["dzicnlib"] = new(DziCnLib)
	})

	if _, ok := Router[siteID]; !ok {
		urlType := getHeaderContentType(sUrl[0])
		if urlType == "json" {
			siteID = "iiif.io"
		} else if urlType != "html" {
			siteID = "bookget"
		}
		if _, ok2 := Router[siteID]; !ok2 {
			return nil, errors.New("Unsupported URL:" + sUrl[0])
		}
		//return nil, errors.New("Unsupported URL:" + sUrl[0])
	}
	return Router[siteID].getRouterInit(sUrl)
}

func getHeaderContentType(sUrl string) string {
	if strings.Contains(sUrl, ".json") {
		return "json"
	}
	m := regexp.MustCompile(`\((\d+)-(\d+)\)`).FindStringSubmatch(sUrl)
	if m != nil {
		return "octet-stream"
	}

	tr := &http.Transport{
		TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
		DisableKeepAlives: true,
	}
	client := http.Client{
		Timeout:   30 * time.Second,
		Transport: tr,
	}
	req, _ := http.NewRequest("GET", sUrl, nil)
	req.Header.Set("User-Agent", config.Conf.UserAgent)
	req.Header.Set("Range", "bytes=0-0")
	resp, err := client.Do(req)
	defer resp.Body.Close()
	if err != nil {
		log.Fatalln(err)
		return ""
	}
	ret := ""
	//application/ld+json
	bodyType := resp.Header.Get("content-type")
	m = strings.Split(bodyType, ";")
	switch m[0] {
	case "application/ld+json":
		ret = "json"
		break
	case "application/json":
		ret = "json"
		break
	case "text/html":
		ret = "html"
		break
	}
	return ret
}

func ExplanRegexpUrl(taskUrl string) (taskUrls []string) {
	uriMatch, ok := util.GetUriMatch(taskUrl)
	if ok {
		iMinLen := len(uriMatch.Min)
		for i := uriMatch.IMin; i <= uriMatch.IMax; i++ {
			iLen := len(strconv.Itoa(i))
			if iLen < iMinLen {
				iLen = iMinLen
			}
			sortId := util.GenNumberLimitLen(i, iLen)
			dUrl := regexp.MustCompile(`\((\d+)-(\d+)\)`).ReplaceAll([]byte(taskUrl), []byte(sortId))
			sUrl := string(dUrl)
			taskUrls = append(taskUrls, sUrl)
		}
		return
	}
	taskUrls = append(taskUrls, taskUrl)
	return
}
