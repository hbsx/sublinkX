package node

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func clashProxyForTest(t *testing.T, link string, cert bool) map[string]interface{} {
	t.Helper()
	template := filepath.Join(t.TempDir(), "clash.yaml")
	if err := os.WriteFile(template, []byte("proxies: []\nproxy-groups: []\n"), 0600); err != nil {
		t.Fatal(err)
	}
	data, err := EncodeClash([]string{link}, SqlConfig{Clash: template, Cert: cert})
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		Proxies []map[string]interface{} `yaml:"proxies"`
	}
	if err := yaml.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	if len(config.Proxies) != 1 {
		t.Fatalf("expected one proxy, got %d: %s", len(config.Proxies), data)
	}
	return config.Proxies[0]
}

func surgeConfigForTest(t *testing.T, link string, cert bool) string {
	t.Helper()
	template := filepath.Join(t.TempDir(), "surge.conf")
	if err := os.WriteFile(template, []byte("[Proxy]\n[Proxy Group]\nAll = select, DIRECT\n[Rule]\nFINAL,All\n"), 0600); err != nil {
		t.Fatal(err)
	}
	result, err := EncodeSurge([]string{link}, SqlConfig{Surge: template, Cert: cert})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestSurgeTUICV5KeepsUUIDAndPassword(t *testing.T) {
	const uuid = "0233d11c-15a4-47d3-ade3-48ffca0ce119"
	result := surgeConfigForTest(t, "tuic://"+uuid+":secret@example.test:443?sni=tls.example.test&allow_insecure=1#tuic", false)
	if !strings.Contains(result, "tuic = tuic-v5, example.test, 443, uuid="+uuid+", password=secret, skip-cert-verify=true, sni=tls.example.test") {
		t.Fatalf("TUIC v5 credentials or TLS options missing: %s", result)
	}
	if strings.Contains(result, "token=") || strings.Contains(result, "udp-relay=") {
		t.Fatalf("TUIC v4 or unsupported options emitted: %s", result)
	}
	result = surgeConfigForTest(t, "tuic://"+uuid+":secret@example.test:443#tuic", true)
	if !strings.Contains(result, "skip-cert-verify=true") {
		t.Fatalf("global certificate option ignored for TUIC: %s", result)
	}
}

func TestSSRParamsSurviveDecodeEncodeAndClashConversion(t *testing.T) {
	const body = "example.test:443:auth_aes128_md5:aes-256-cfb:tls1.2_ticket_auth:" +
		"c2VjcmV0/?obfsparam=b2Jmcy5leGFtcGxlLnRlc3Q=&protoparam=MTIzOnVzZXI=&remarks=U1NS"
	link := "ssr://" + Base64Encode(body)
	decoded, err := DecodeSSRURL(link)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Qurey.Obfsparam != "obfs.example.test" || decoded.Qurey.Protoparam != "123:user" {
		t.Fatalf("SSR parameters were lost during decode: %+v", decoded.Qurey)
	}
	encoded := EncodeSSRURL(decoded)
	roundtrip, err := DecodeSSRURL(encoded)
	if err != nil || roundtrip.Qurey != decoded.Qurey {
		t.Fatalf("SSR parameters were lost during encode: %+v, %v", roundtrip.Qurey, err)
	}
	proxy := clashProxyForTest(t, link, false)
	if proxy["obfs-param"] != "obfs.example.test" || proxy["protocol-param"] != "123:user" {
		t.Fatalf("SSR Clash fields are wrong: %#v", proxy)
	}
	if _, ok := proxy["obfs-password"]; ok {
		t.Fatalf("SSR output contains Hysteria2 field obfs-password: %#v", proxy)
	}
}

func TestClashPerNodeCertificateOptionsAndHysteriaSNI(t *testing.T) {
	tests := []struct {
		name, insecureLink, ordinaryLink string
	}{
		{"trojan", "trojan://secret@example.test:443?allowInsecure=1#trojan", "trojan://secret@example.test:443#trojan"},
		{"hysteria", "hy://example.test:443?auth=secret&insecure=1&peer=tls.example.test#hy", "hy://example.test:443?auth=secret#hy"},
		{"hysteria2", "hy2://secret@example.test:443?insecure=1#hy2", "hy2://secret@example.test:443#hy2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proxy := clashProxyForTest(t, tt.insecureLink, false)
			if proxy["skip-cert-verify"] != true {
				t.Fatalf("per-node insecure option ignored: %#v", proxy)
			}
			if tt.name == "hysteria" {
				if proxy["sni"] != "tls.example.test" {
					t.Fatalf("HY1 peer was not mapped to sni: %#v", proxy)
				}
				if _, ok := proxy["peer"]; ok {
					t.Fatalf("HY1 output still has unsupported peer field: %#v", proxy)
				}
			}
			proxy = clashProxyForTest(t, tt.ordinaryLink, true)
			if proxy["skip-cert-verify"] != true {
				t.Fatalf("global certificate option ignored: %#v", proxy)
			}
		})
	}
}

func TestSurgePerNodeCertificateOptions(t *testing.T) {
	tests := []struct {
		name, insecureLink, ordinaryLink string
	}{
		{"trojan", "trojan://secret@example.test:443?allowInsecure=1#trojan", "trojan://secret@example.test:443#trojan"},
		{"hysteria2", "hy2://secret@example.test:443?insecure=1#hy2", "hy2://secret@example.test:443#hy2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if result := surgeConfigForTest(t, tt.insecureLink, false); !strings.Contains(result, "skip-cert-verify=true") {
				t.Fatalf("per-node insecure option ignored: %s", result)
			}
			if result := surgeConfigForTest(t, tt.ordinaryLink, true); !strings.Contains(result, "skip-cert-verify=true") {
				t.Fatalf("global certificate option ignored: %s", result)
			}
			if result := surgeConfigForTest(t, tt.ordinaryLink, false); !strings.Contains(result, "skip-cert-verify=false") {
				t.Fatalf("default certificate verification changed: %s", result)
			}
		})
	}
}
