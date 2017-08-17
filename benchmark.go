package MeloyApi

import (
	"net/http"
	"bytes"
	"sync"
	"strings"
	"time"
)

// 基准测试管理器
type BenchmarkManager struct {
}

// 执行基准测试
func (manager *BenchmarkManager) test(uri string, method string, header http.Header, body string, totalRequests int, concurrency int) (successRequests int, failRequests int, requestsPerSecond int, avgMs float32, err error) {
	method = strings.ToUpper(method)

	if totalRequests < 1 {
		totalRequests = 1000
	}

	if concurrency < 1 {
		concurrency = 32
	}

	var client = &http.Client{}
	client.Timeout = 5 * time.Second

	var mutex = sync.Mutex{}
	var wg = sync.WaitGroup{}
	wg.Add(concurrency)

	var sumMs float32 = 0
	var startTime = time.Now().UnixNano()
	for i := 0; i < concurrency; i ++ {
		go func(id int) {
			var count = totalRequests / concurrency
			var mod = totalRequests % concurrency
			if id < mod {
				count ++
			}

			if count == 0 {
				wg.Done()
				return
			}

			for j := 0; j < count; j ++ {
				request, err := http.NewRequest(method, uri, bytes.NewReader([]byte(body)))
				if err != nil {
					continue
				}

				if method == "POST" {
					request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
				}

				request.Header.Set("User-Agent", "MeloyAPI Benchmark")

				for key, values := range header {
					for _, value := range values {
						request.Header.Set(key, value)
					}
				}

				var now = time.Now().UnixNano()
				response, err := client.Do(request)

				if err == nil {
					response.Body.Close()
				}
				var ns = time.Now().UnixNano() - now

				mutex.Lock()

				if err != nil {
					failRequests ++
				} else {
					sumMs += float32(ns) / 1000000
					successRequests ++
				}

				totalRequests --

				mutex.Unlock()

				if j == count-1 {
					wg.Done()
					return
				}
			}
		}(i)
	}

	wg.Wait()

	var totalMs = (time.Now().UnixNano() - startTime) / 1000000
	requestsPerSecond = int(float32(successRequests+failRequests) / (float32(totalMs) / 1000))

	if successRequests > 0 {
		avgMs = sumMs / float32(successRequests)
	}

	return
}
