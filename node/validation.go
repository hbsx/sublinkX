package node

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

var legacyLinkSeparator = regexp.MustCompile(`,\s*([A-Za-z][A-Za-z0-9+.-]*://)`)

// Preserve commas in URL parameters; accept legacy comma-separated complete URLs.
func SplitLinks(raw string) []string {
	raw = legacyLinkSeparator.ReplaceAllString(raw, "\n$1")
	var links []string
	for _, line := range strings.Split(raw, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			links = append(links, line)
		}
	}
	return links
}

func decodeBase64(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	for _, encoding := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding} {
		if data, err := encoding.DecodeString(raw); err == nil {
			return string(data), nil
		}
	}
	return "", fmt.Errorf("Base64 内容无效")
}

func validPort(raw string) (int, error) {
	port, err := strconv.Atoi(raw)
	if err != nil || port < 1 || port > 65535 {
		return 0, fmt.Errorf("节点端口必须为 1 到 65535")
	}
	return port, nil
}

func validateEndpoint(u *url.URL, needUser bool) error {
	if u.Hostname() == "" || (needUser && (u.User == nil || u.User.Username() == "")) {
		return fmt.Errorf("节点缺少服务器或认证信息")
	}
	_, err := validPort(u.Port())
	return err
}

// Validate even when the user supplied a name.
func NodeName(raw string) (string, error) {
	scheme, _, ok := strings.Cut(raw, "://")
	if !ok {
		return "", fmt.Errorf("节点缺少协议头")
	}
	switch strings.ToLower(scheme) {
	case "ss":
		v, e := DecodeSSURL(raw)
		return v.Name, e
	case "ssr":
		v, e := DecodeSSRURL(raw)
		return v.Qurey.Remarks, e
	case "vmess":
		v, e := DecodeVMESSURL(raw)
		return v.Ps, e
	case "vless":
		v, e := DecodeVLESSURL(raw)
		return v.Name, e
	case "trojan":
		v, e := DecodeTrojanURL(raw)
		return v.Name, e
	case "hy", "hysteria":
		v, e := DecodeHYURL(raw)
		return v.Name, e
	case "hy2", "hysteria2":
		v, e := DecodeHY2URL(raw)
		return v.Name, e
	case "tuic":
		v, e := DecodeTuicURL(raw)
		return v.Name, e
	case "anytls":
		v, e := DecodeAnyTLSURL(raw)
		return v.Name, e
	case "socks", "socks5", "http", "https":
		v, e := DecodeStandardProxyURL(raw)
		return v.Name, e
	default:
		return "", fmt.Errorf("不支持的节点协议")
	}
}

func DecodeSubscription(body string) ([]string, error) {
	body = strings.TrimSpace(body)
	if !strings.Contains(body, "://") {
		decoded, err := decodeBase64(body)
		if err != nil {
			return nil, fmt.Errorf("远程订阅格式无效")
		}
		body = decoded
	}
	links := SplitLinks(body)
	if len(links) == 0 {
		return nil, fmt.Errorf("订阅没有有效节点")
	}
	for _, link := range links {
		if _, err := NodeName(link); err != nil {
			return nil, fmt.Errorf("订阅包含无效或不支持的节点")
		}
	}
	return links, nil
}

func uniqueName(name string, used map[string]bool) string {
	base := name
	for i := 2; used[name]; i++ {
		name = fmt.Sprintf("%s (%d)", base, i)
	}
	used[name] = true
	return name
}
