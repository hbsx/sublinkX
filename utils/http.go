package utils

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var HTTPClient = &http.Client{Timeout: 15 * time.Second}

// Fetch bounds network waits and memory usage, without logging secret URLs.
func Fetch(ctx context.Context, address string, limit int64) ([]byte, error) {
	u, err := url.Parse(address)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, fmt.Errorf("远程地址必须使用 HTTP 或 HTTPS")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return nil, fmt.Errorf("远程地址无效")
	}
	response, err := HTTPClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("远程请求失败或超时")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("远程服务返回 HTTP %d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil {
		return nil, fmt.Errorf("读取远程内容失败或超时")
	}
	if int64(len(body)) > limit {
		return nil, fmt.Errorf("远程内容超过大小限制")
	}
	return body, nil
}

// Explicit choices take precedence; old entries retain automatic detection.
func IsRemoteSubscription(raw, sourceType string) bool {
	if sourceType == "proxy" {
		return false
	}
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return false
	}
	if sourceType == "subscription" {
		return true
	}
	if u.User != nil || u.Fragment != "" {
		return false
	}
	if u.Port() != "" && (u.Path == "" || u.Path == "/") && u.RawQuery == "" {
		return false
	}
	return true
}
