package node

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

type StandardProxyURL struct {
	Name     string
	Scheme   string
	Server   string
	Port     int
	Username string
	Password string
	TLS      bool
}

func DecodeStandardProxyURL(raw string) (StandardProxyURL, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return StandardProxyURL{}, fmt.Errorf("url parse error: %w", err)
	}
	scheme := strings.ToLower(u.Scheme)
	defaultPort := 0
	switch scheme {
	case "socks", "socks5":
		defaultPort = 1080
	case "http":
		defaultPort = 80
	case "https":
		defaultPort = 443
	default:
		return StandardProxyURL{}, fmt.Errorf("不支持的代理协议: %s", u.Scheme)
	}
	if u.Hostname() == "" {
		return StandardProxyURL{}, fmt.Errorf("代理地址缺少服务器")
	}
	port := defaultPort
	if u.Port() != "" {
		port, err = strconv.Atoi(u.Port())
		if err != nil || port < 1 || port > 65535 {
			return StandardProxyURL{}, fmt.Errorf("代理端口无效: %q", u.Port())
		}
	}
	username, password := "", ""
	if u.User != nil {
		username = u.User.Username()
		password, _ = u.User.Password()
	}
	name := u.Fragment
	if name == "" {
		name = fmt.Sprintf("%s:%d", u.Hostname(), port)
	}
	return StandardProxyURL{
		Name:     name,
		Scheme:   scheme,
		Server:   u.Hostname(),
		Port:     port,
		Username: username,
		Password: password,
		TLS:      scheme == "https",
	}, nil
}
