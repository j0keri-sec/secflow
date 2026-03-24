// +build ignore

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/imroc/req/v3"
)

func main() {
	fmt.Println("=== Testing Qianxin APIs ===")

	logDir := "./log"
	os.MkdirAll(logDir, 0755)

	client := req.NewClient().
		SetTimeout(60 * time.Second).
		SetUserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36")

	articleID := 1763

	// Test various API endpoints
	apis := []struct {
		name   string
		method string
		url    string
		body   interface{}
	}{
		{
			name:   "article-hotspot (list)",
			method: "POST",
			url:    "https://ti.qianxin.com/alpha-api/v2/vuln/article-hotspot",
			body: map[string]interface{}{
				"page_no":   1,
				"page_size": 10,
				"category":  "热点周报",
			},
		},
		{
			name:   "article-hotspot/detail",
			method: "POST",
			url:    "https://ti.qianxin.com/alpha-api/v2/vuln/article-hotspot/detail",
			body: map[string]interface{}{
				"id": articleID,
			},
		},
		{
			name:   "notice-detail",
			method: "GET",
			url:    fmt.Sprintf("https://ti.qianxin.com/alpha-api/v2/vuln/notice-detail/%d?type=hot-week", articleID),
			body:   nil,
		},
		{
			name:   "article/detail",
			method: "GET",
			url:    fmt.Sprintf("https://ti.qianxin.com/alpha-api/v2/article/detail/%d", articleID),
			body:   nil,
		},
		{
			name:   "vuln/article",
			method: "POST",
			url:    "https://ti.qianxin.com/alpha-api/v2/vuln/article",
			body: map[string]interface{}{
				"id": articleID,
			},
		},
	}

	for _, api := range apis {
		fmt.Printf("\n--- Testing: %s ---\n", api.name)
		fmt.Printf("URL: %s\n", api.url)

		var resp *req.Response
		var err error

		headers := map[string]string{
			"Accept":       "application/json, text/plain, */*",
			"Accept-Language": "zh-CN,zh;q=0.9,en;q=0.8",
			"Origin":       "https://ti.qianxin.com",
			"Referer":      "https://ti.qianxin.com/vulnerability/notice-list",
		}

		if api.method == "POST" {
			resp, err = client.R().
				SetHeaders(headers).
				SetHeader("Content-Type", "application/json").
				SetBody(api.body).
				Post(api.url)
		} else {
			resp, err = client.R().
				SetHeaders(headers).
				Get(api.url)
		}

		if err != nil {
			fmt.Printf("ERROR: %v\n", err)
			continue
		}
		defer resp.Body.Close()

		fmt.Printf("Status: %d\n", resp.StatusCode)

		if resp.StatusCode == 200 {
			// Pretty print JSON response
			var jsonData interface{}
			if err := json.Unmarshal(resp.Bytes(), &jsonData); err == nil {
				jsonBytes, _ := json.MarshalIndent(jsonData, "", "  ")
				content := string(jsonBytes)

				// Save to file
				filename := filepath.Join(logDir, fmt.Sprintf("api_%s_%d.txt", api.name, time.Now().Unix()))
				os.WriteFile(filename, []byte(content), 0644)
				fmt.Printf("Response (%d chars) saved to: %s\n", len(content), filename)

				// Print first 500 chars
				if len(content) > 500 {
					fmt.Printf("Preview: %s...\n", content[:500])
				} else {
					fmt.Printf("Response: %s\n", content)
				}
			} else {
				fmt.Printf("Response (raw): %s\n", string(resp.Bytes()))
			}
		} else {
			fmt.Printf("Response: %s\n", string(resp.Bytes()))
		}
	}

	fmt.Println("\n=== Done ===")
}