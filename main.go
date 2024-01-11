package main

import (
	"bookget/config"
	"bookget/router"
	"bufio"
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"sync"
)

var wg sync.WaitGroup

func main() {
	ctx := context.Background()
	//配置初始化
	if !config.Init(ctx) {
		os.Exit(0)
	}

	//終端運行：单个URL
	if config.Conf.DUrl != "" {
		ExecuteCommand(ctx, 1, config.Conf.DUrl)
		log.Println("Download complete.")
		return
	}
	//終端運行：批量URLs
	if f, err := os.Stat(config.Conf.UrlsFile); err == nil && f.Size() > 0 {
		taskForUrls()
		return
	}
	//雙擊運行
	iCount := 0
	for {
		reader := bufio.NewReader(os.Stdin)
		fmt.Println("Enter an URL:")
		fmt.Print("-> ")
		sUrl, err := reader.ReadString('\n')
		if err != nil {
			//fmt.Printf("Error: %w \n", err)
			break
		}
		iCount++
		sUrl = strings.TrimSpace(sUrl)
		ExecuteCommand(ctx, iCount, sUrl)
	}
	log.Println("Download complete.")
}

func taskForUrls() {
	//加载配置文件
	bs, err := os.ReadFile(config.Conf.UrlsFile)
	if err != nil {
		fmt.Println(err)
		return
	}
	mUrls := strings.Split(string(bs), "\n")
	q := QueueNew(int(config.Conf.Threads))
	sUrls := make([]string, 0, len(mUrls))
	for _, v := range mUrls {
		sUrl := strings.TrimSpace(strings.Trim(v, "\r"))
		if sUrl == "" || !strings.HasPrefix(sUrl, "http") {
			continue
		}
		sUrls = append(sUrls, sUrl)
	}
	if config.Conf.AutoDetect == 1 {
		wg.Add(1)
		q.Go(func() {
			defer wg.Done()
			msg, err := router.FactoryRouter("bookget", sUrls)
			if err != nil {
				fmt.Println(err)
				return
			}
			if msg != nil {
				fmt.Printf("%+v\n", msg)
			}
		})
	} else {
		for _, v := range sUrls {
			u, err := url.Parse(v)
			if err != nil {
				continue
			}
			wg.Add(1)
			sUrl := []string{v}
			q.Go(func() {
				defer wg.Done()
				msg, err := router.FactoryRouter(u.Host, sUrl)
				if err != nil {
					fmt.Println(err)
					return
				}
				if msg != nil {
					fmt.Printf("%+v\n", msg)
				}
			})
		}
	}
	wg.Wait()
	log.Println("Download complete.")
	return
}

func ExecuteCommand(ctx context.Context, i int, sUrl string) {
	if sUrl == "" || !strings.HasPrefix(sUrl, "http") {
		fmt.Println("URL Not found.")
		return
	}
	sUrl = strings.Trim(sUrl, "\r\n")
	u, err := url.Parse(sUrl)
	if err != nil {
		fmt.Printf("URL Error:%+v\n", err)
		return
	}
	siteId := u.Host
	if config.Conf.AutoDetect == 1 {
		siteId = "bookget"
	}
	msg, err := router.FactoryRouter(siteId, []string{sUrl})
	if err != nil {
		fmt.Println(err)
		return
	}
	if msg != nil {
		fmt.Printf("%+v\n", msg)
	}
	return
}
