package node

import (
	"os"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestDecodeAnyTLSURL(t *testing.T) {
	got, err := DecodeAnyTLSURL("anytls://secret@example.com:443?security=tls&sni=cdn.example.com&fp=chrome&alpn=h2%2Chttp%2F1.1&insecure=1&pcs=abc#Hong%20Kong")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Hong Kong" || got.Server != "example.com" || got.Port != 443 || got.Password != "secret" {
		t.Fatalf("unexpected address fields: %#v", got)
	}
	if got.SNI != "cdn.example.com" || got.ClientFingerprint != "chrome" || !got.SkipCertVerify || got.PCS != "abc" {
		t.Fatalf("unexpected TLS fields: %#v", got)
	}
	if len(got.ALPN) != 2 || got.ALPN[0] != "h2" || got.ALPN[1] != "http/1.1" {
		t.Fatalf("unexpected ALPN: %#v", got.ALPN)
	}
}

func TestDecodeStandardProxyURL(t *testing.T) {
	tests := []struct {
		raw, scheme, host, user, password string
		port                              int
		tls                               bool
	}{
		{"socks5://user:pass@127.0.0.1:1081#local", "socks5", "127.0.0.1", "user", "pass", 1081, false},
		{"http://proxy.example.com", "http", "proxy.example.com", "", "", 80, false},
		{"https://u:p@[2001:db8::1]#secure", "https", "2001:db8::1", "u", "p", 443, true},
	}
	for _, tt := range tests {
		got, err := DecodeStandardProxyURL(tt.raw)
		if err != nil {
			t.Fatalf("%s: %v", tt.raw, err)
		}
		if got.Scheme != tt.scheme || got.Server != tt.host || got.Port != tt.port || got.Username != tt.user || got.Password != tt.password || got.TLS != tt.tls {
			t.Fatalf("%s: unexpected result %#v", tt.raw, got)
		}
	}
}

func TestEncodeClashNewProtocolsAndXHTTP(t *testing.T) {
	template, err := os.CreateTemp(t.TempDir(), "clash-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = template.WriteString("proxies: []\nproxy-groups: []\n"); err != nil {
		t.Fatal(err)
	}
	if err = template.Close(); err != nil {
		t.Fatal(err)
	}

	links := []string{
		"anytls://secret@example.com:443?sni=cdn.example.com&fp=chrome&alpn=h2&allowInsecure=true#any",
		"socks5://user:pass@127.0.0.1:1080#socks",
		"https://user:pass@proxy.example.com:8443#https",
		"vless://00000000-0000-0000-0000-000000000001@example.com:443?security=tls&type=xhttp&path=%2Fapi&host=cdn.example.com&mode=auto#xhttp",
	}
	data, err := EncodeClash(links, SqlConfig{Clash: template.Name(), Udp: true})
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		Proxies []map[string]interface{} `yaml:"proxies"`
	}
	if err = yaml.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	if len(config.Proxies) != 4 {
		t.Fatalf("expected 4 proxies, got %d: %s", len(config.Proxies), data)
	}
	if config.Proxies[0]["type"] != "anytls" || config.Proxies[0]["skip-cert-verify"] != true {
		t.Fatalf("unexpected AnyTLS output: %#v", config.Proxies[0])
	}
	if config.Proxies[1]["type"] != "socks5" || config.Proxies[1]["username"] != "user" {
		t.Fatalf("unexpected SOCKS output: %#v", config.Proxies[1])
	}
	if config.Proxies[2]["type"] != "http" || config.Proxies[2]["tls"] != true {
		t.Fatalf("unexpected HTTPS output: %#v", config.Proxies[2])
	}
	xhttpOpts, ok := config.Proxies[3]["xhttp-opts"].(map[string]interface{})
	if !ok || xhttpOpts["path"] != "/api" || xhttpOpts["mode"] != "auto" {
		t.Fatalf("unexpected XHTTP output: %#v", config.Proxies[3])
	}
}
