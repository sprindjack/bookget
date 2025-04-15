package http

import (
	"bookget/pkg/progressbar"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

type DownloadPart struct {
	URL       string
	FilePath  string
	StartByte int64
	EndByte   int64
	PartNum   int
}

func example() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run downloader.go <URL> <output_file> [threads]")
		return
	}

	url := os.Args[1]
	filePath := os.Args[2]
	threads := 4 // 默认4线程

	if len(os.Args) > 3 {
		t, err := strconv.Atoi(os.Args[3])
		if err == nil && t > 0 {
			threads = t
		}
	}

	err := DownloadFile(url, filePath, threads)
	if err != nil {
		fmt.Printf("Download failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("\nDownload completed successfully!")
}

func DownloadFile(url, filePath string, threads int) error {
	// 获取文件大小
	size, err := getFileSize(url)
	if err != nil {
		return fmt.Errorf("failed to get file size: %v", err)
	}

	fmt.Printf("Downloading %s (%.2f MB) with %d threads...\n",
		url, float64(size)/(1024*1024), threads)

	// 创建临时目录
	tmpDir := filePath + "_tmp"
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp directory: %v", err)
	}
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			log.Println("failed to remove temp directory:", err)
		}
	}(tmpDir)

	// 计算每个部分的大小
	partSize := size / int64(threads)
	lastPartSize := partSize + size%int64(threads)

	// 创建进度条
	bar := progressbar.NewOptions64(
		size,
		progressbar.OptionSetDescription("Downloading"),
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetWidth(50),
		progressbar.OptionThrottle(100*time.Millisecond),
		progressbar.OptionShowCount(),
		progressbar.OptionOnCompletion(func() {
			fmt.Fprint(os.Stderr, "\n")
		}),
	)

	var wg sync.WaitGroup
	errors := make(chan error, threads)

	// 启动下载goroutines
	for i := 0; i < threads; i++ {
		wg.Add(1)
		go func(partNum int) {
			defer wg.Done()

			var start, end int64
			if partNum == threads-1 {
				start = int64(partNum) * partSize
				end = start + lastPartSize - 1
			} else {
				start = int64(partNum) * partSize
				end = start + partSize - 1
			}

			partFile := fmt.Sprintf("%s/part%d", tmpDir, partNum)
			part := DownloadPart{
				URL:       url,
				FilePath:  partFile,
				StartByte: start,
				EndByte:   end,
				PartNum:   partNum,
			}

			err := downloadPart(part, bar)
			if err != nil {
				errors <- fmt.Errorf("part %d failed: %v", partNum, err)
			}
		}(i)
	}

	// 等待所有下载完成
	go func() {
		wg.Wait()
		close(errors)
	}()

	// 检查错误
	for err := range errors {
		if err != nil {
			return err
		}
	}

	// 合并文件
	if err := mergeFiles(tmpDir, filePath, threads); err != nil {
		return fmt.Errorf("failed to merge files: %v", err)
	}

	return nil
}

func getFileSize(url string) (int64, error) {
	resp, err := http.Head(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("server returned non-200 status: %d %s", resp.StatusCode, resp.Status)
	}

	size, err := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse Content-Length: %v", err)
	}

	return size, nil
}

func downloadPart(part DownloadPart, bar *progressbar.ProgressBar) error {
	client := &http.Client{}
	req, err := http.NewRequest("GET", part.URL, nil)
	if err != nil {
		return err
	}

	rangeHeader := fmt.Sprintf("bytes=%d-%d", part.StartByte, part.EndByte)
	req.Header.Add("Range", rangeHeader)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("server does not support partial content (status %d)", resp.StatusCode)
	}

	file, err := os.Create(part.FilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	reader := io.TeeReader(resp.Body, bar)
	if _, err := io.Copy(file, reader); err != nil {
		return err
	}

	return nil
}

func mergeFiles(tmpDir, outputFile string, parts int) error {
	outFile, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer outFile.Close()

	for i := 0; i < parts; i++ {
		partFile := fmt.Sprintf("%s/part%d", tmpDir, i)
		inFile, err := os.Open(partFile)
		if err != nil {
			return err
		}

		if _, err := io.Copy(outFile, inFile); err != nil {
			inFile.Close()
			return err
		}
		inFile.Close()
	}

	return nil
}
