// Package httputil provides shared HTTP client helpers used across grabbers
// and notifiers. It is intentionally kept small — no business logic here.
package httputil

import (
	"context"
	"errors"
	"math/rand"
	"os"
	"time"

	"github.com/imroc/req/v3"
)

// NewClient returns a preconfigured req.Client suitable for web scraping.
// It impersonates Chrome, sets a random User-Agent, configures automatic
// retries with jitter, and respects the GO_SKIP_TLS_CHECK environment variable.
func NewClient() *req.Client {
	c := req.C()
	c.ImpersonateChrome().
		SetCommonHeader("User-Agent", randUserAgent()).
		SetTimeout(15 * time.Second).
		SetCommonRetryCount(3).
		SetCommonRetryInterval(func(resp *req.Response, attempt int) time.Duration {
			if errors.Is(resp.Err, context.Canceled) {
				return 0
			}
			return time.Second*5 + time.Duration(rand.Int63n(int64(time.Second*3)))
		}).
		SetCommonRetryHook(func(resp *req.Response, err error) {
			if err != nil && !errors.Is(err, context.Canceled) {
				c.Headers.Set("User-Agent", randUserAgent())
			}
		}).
		SetCommonRetryCondition(func(resp *req.Response, err error) bool {
			if err != nil {
				return !errors.Is(err, context.Canceled)
			}
			return false
		})

	if os.Getenv("GO_SKIP_TLS_CHECK") != "" {
		c.EnableInsecureSkipVerify()
	}
	return c
}

// WrapAPIClient adds common JSON API headers to an existing client.
func WrapAPIClient(c *req.Client) *req.Client {
	return c.SetCommonHeaders(map[string]string{
		"Accept":             "application/json, text/plain, */*",
		"Accept-Language":    "zh-CN,zh;q=0.9,en;q=0.8",
		"Content-Type":       "application/json",
		"Sec-Fetch-Dest":     "empty",
		"Sec-Fetch-Mode":     "cors",
		"Sec-Fetch-Site":     "same-origin",
		"sec-ch-ua":          `"Chromium";v="122", "Not(A:Brand";v="24"`,
		"sec-ch-ua-mobile":   `?0`,
		"sec-ch-ua-platform": `"Windows"`,
	})
}

var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:123.0) Gecko/20100101 Firefox/123.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 14.3; rv:123.0) Gecko/20100101 Firefox/123.0",
}

func randUserAgent() string {
	return userAgents[rand.Intn(len(userAgents))]
}
