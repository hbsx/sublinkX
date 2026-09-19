package node

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

type AnyTLS struct {
	Name              string
	Server            string
	Port              int
	Password          string
	SNI               string
	ClientFingerprint string
	ALPN              []string
	SkipCertVerify    bool
	PCS               string
}

func DecodeAnyTLSURL(raw string) (AnyTLS, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return AnyTLS{}, fmt.Errorf("url parse error: %w", err)
	}
	if !strings.EqualFold(u.Scheme, "anytls") {
		return AnyTLS{}, fmt.Errorf("非 anytls 协议: %s", raw)
	}
	if u.User == nil || u.User.Username() == "" || u.Hostname() == "" {
		return AnyTLS{}, fmt.Errorf("anytls 地址缺少密码或服务器")
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil || port < 1 || port > 65535 {
		return AnyTLS{}, fmt.Errorf("anytls 端口无效: %q", u.Port())
	}
	q := u.Query()
	sni := q.Get("sni")
	if sni == "" {
		sni = q.Get("peer")
	}
	name := u.Fragment
	if name == "" {
		name = fmt.Sprintf("%s:%d", u.Hostname(), port)
	}
	return AnyTLS{
		Name:              name,
		Server:            u.Hostname(),
		Port:              port,
		Password:          u.User.Username(),
		SNI:               sni,
		ClientFingerprint: q.Get("fp"),
		ALPN:              splitCommaList(q.Get("alpn")),
		SkipCertVerify:    queryBool(q.Get("insecure")) || queryBool(q.Get("allowInsecure")),
		PCS:               q.Get("pcs"),
	}, nil
}

func queryBool(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func splitCommaList(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			result = append(result, part)
		}
	}
	return result
}
