package main

import (
	"bookget/config"
	"bookget/pkg/queue"
	"bookget/pkg/version"
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

var (
	wg             sync.WaitGroup
	versionChecker = version.NewChecker(
		config.Version,
		"deweizhu", // GitHub仓库所有者
		"bookget",  // GitHub仓库名称
	)
)

func main() {
	ctx := context.Background()

	// 初始化配置
	if !initializeConfig(ctx) {
		return
	}

	// 检查更新
	checkForUpdates()

	// 根据运行模式执行相应操作
	executeByRunMode(ctx)
}

// initializeConfig 处理配置初始化
func initializeConfig(ctx context.Context) bool {
	if !config.Init(ctx) {
		log.Println("配置初始化失败")
		return false
	}
	return true
}

// executeByRunMode 根据运行模式执行相应操作
func executeByRunMode(ctx context.Context) {
	switch determineRunMode() {
	case RunModeSingleURL:
		executeSingleURL(ctx, config.Conf.DUrl)
	case RunModeBatchURLs:
		executeBatchURLs()
	case RunModeInteractive:
		runInteractiveMode(ctx)
	}
	log.Println("Download complete.")
}

type RunMode int

const (
	RunModeSingleURL RunMode = iota
	RunModeBatchURLs
	RunModeInteractive
)

// determineRunMode 确定运行模式
func determineRunMode() RunMode {
	if config.Conf.DUrl != "" {
		return RunModeSingleURL
	}
	if hasValidURLsFile() {
		return RunModeBatchURLs
	}
	return RunModeInteractive
}

// hasValidURLsFile 检查是否有有效的URLs文件
func hasValidURLsFile() bool {
	f, err := os.Stat(config.Conf.UrlsFile)
	return err == nil && f.Size() > 0
}

// executeSingleURL 处理单个URL模式
func executeSingleURL(ctx context.Context, url string) {
	if err := processURL(ctx, 1, url); err != nil {
		log.Println(err)
	}
}

// executeBatchURLs 处理批量URLs模式
func executeBatchURLs() {
	urls, err := loadAndFilterURLs(config.Conf.UrlsFile)
	if err != nil {
		log.Println(err)
		return
	}

	q := queue.NewConcurrentQueue(int(config.Conf.Threads))
	if config.Conf.AutoDetect == 1 {
		processURLsAutoDetect(q, urls)
	} else {
		processURLsManual(q, urls)
	}
	wg.Wait()
}

// runInteractiveMode 运行交互模式
func runInteractiveMode(ctx context.Context) {
	cleanupCookieFile()

	counter := 0
	for {
		sUrl, err := readURLFromInput()
		if err != nil {
			break
		}
		counter++

		if err := processURL(ctx, counter, sUrl); err != nil {
			log.Println(err)
		}
	}
}

// loadAndFilterURLs 加载并过滤URLs
func loadAndFilterURLs(filename string) ([]string, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("无法读取URL文件: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	var urls []string
	for _, line := range lines {
		sUrl := strings.TrimSpace(strings.Trim(line, "\r"))
		if isValidURL(sUrl) {
			urls = append(urls, sUrl)
		}
	}

	if len(urls) == 0 {
		return nil, fmt.Errorf("URL文件中没有有效的URL")
	}

	return urls, nil
}

// isValidURL 验证URL是否有效
func isValidURL(url string) bool {
	return url != "" && strings.HasPrefix(url, "http")
}

// processURLsAutoDetect 自动检测模式处理URLs
func processURLsAutoDetect(q *queue.ConcurrentQueue, urls []string) {
	wg.Add(1)
	q.Go(func() {
		defer wg.Done()
		processURLSet("bookget", urls)
	})
}

// processURLsManual 手动模式处理URLs
func processURLsManual(q *queue.ConcurrentQueue, urls []string) {
	for _, rawURL := range urls {
		u, err := url.Parse(rawURL)
		if err != nil {
			log.Printf("URL解析失败: %s, 错误: %v\n", rawURL, err)
			continue
		}

		wg.Add(1)
		sUrl := rawURL // 创建局部变量供闭包使用
		q.Go(func() {
			defer wg.Done()
			processURLSet(u.Host, []string{sUrl})
		})
	}
}

// processURLSet 处理一组URLs
func processURLSet(siteID string, urls []string) {
	msg, err := router.FactoryRouter(siteID, urls)
	if err != nil {
		log.Println(err)
		return
	}
	if msg != nil {
		log.Printf("%+v\n", msg)
	}
}

// readURLFromInput 从用户输入读取URL
func readURLFromInput() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Enter an URL:")
	fmt.Print("-> ")
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("读取输入失败: %w", err)
	}
	return strings.TrimSpace(input), nil
}

// processURL 处理单个URL
func processURL(ctx context.Context, id int, rawURL string) error {
	rawURL = strings.TrimSpace(rawURL)
	if !isValidURL(rawURL) {
		return fmt.Errorf("无效的URL: %s", rawURL)
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("URL解析失败: %w", err)
	}

	msg, err := router.FactoryRouter(u.Host, []string{rawURL})
	if err != nil {
		return fmt.Errorf("处理URL失败: %w", err)
	}

	if msg != nil {
		log.Printf("%d %+v\n", id, msg)
	}

	return nil
}

// cleanupCookieFile 清理cookie文件
func cleanupCookieFile() {
	if err := os.Remove(config.Conf.CookieFile); err != nil && !os.IsNotExist(err) {
		log.Printf("清理cookie文件失败: %v\n", err)
	}
}

// checkForUpdates 检查版本更新
func checkForUpdates() {
	latestVersion, updateAvailable, err := versionChecker.CheckForUpdate()
	if err != nil {
		log.Printf("版本检查失败: %v\n", err)
		return
	}

	if updateAvailable {
		fmt.Printf("\n新版本可用: %s (当前版本: %s)\n", latestVersion, versionChecker.CurrentVersion)
		fmt.Printf("请访问 https://github.com/deweizhu/bookget/releases/latest 升级。\n\n")
	} else if latestVersion != "" {
		fmt.Printf("当前已是最新版本: %s\n", versionChecker.CurrentVersion)
	}
}
