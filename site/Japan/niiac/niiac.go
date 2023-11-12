package niiac

import (
	"bookget/app"
	"bookget/config"
	curl "bookget/lib/curl"
	util "bookget/lib/util"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
)

func Init(iTask int, taskUrl string) (msg string, err error) {
	bookId := ""
	m := regexp.MustCompile(`toyobunko/([^/]+)/([^/]+)`).FindStringSubmatch(taskUrl)
	if m != nil {
		bookId = fmt.Sprintf("%s.%s", m[1], m[2])
		config.CreateDirectory(taskUrl, bookId)
		StartDownload(iTask, taskUrl, bookId)
	}
	return "", err
}

func StartDownload(iTask int, pageUrl, bookId string) {
	name := util.GenNumberSorted(iTask)
	log.Printf("Get %s  %s\n", name, pageUrl)

	imageUrls, iiifUrls := getImageUrls(bookId, pageUrl)
	if imageUrls == nil || iiifUrls == nil {
		return
	}
	size := len(imageUrls)
	log.Printf(" %d pages.\n", size)

	config.CreateDirectory(pageUrl, bookId)
	if config.Conf.UseDziRs {
		app.DziDownload(pageUrl, bookId, iiifUrls)
	} else {
		app.NormalDownload(pageUrl, bookId, imageUrls, nil)
	}
}

func getImageUrls(bookId, bookUrl string) (imgUrls []string, iiifUrls []string) {
	uri := fmt.Sprintf("%s/manifest.json", bookUrl)
	var manifest = new(Manifest)
	bs, err := curl.Get(uri, nil)
	if err != nil {
		return
	}
	if err = json.Unmarshal(bs, manifest); err != nil {
		log.Printf("json.Unmarshal failed: %s\n", err)
		return
	}
	if len(manifest.Sequences) == 0 {
		return
	}
	i := len(manifest.Sequences[0].Canvases)
	imgUri := make([]string, 0, i)
	iiifUri := make([]string, 0, i)
	for _, canvase := range manifest.Sequences[0].Canvases {
		for _, image := range canvase.Images {
			//dezoomify-rs URL
			iiiInfo := fmt.Sprintf("%s/info.json", image.Resource.Service.Id)
			iiifUri = append(iiifUri, iiiInfo)

			//JPEG URL
			imgUrl := image.Resource.Service.Id + "/" + config.Conf.Format
			imgUri = append(imgUri, imgUrl)
		}
	}
	return imgUri, iiifUri
}
