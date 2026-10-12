package node

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

const testTUICUUID = "00000000-0000-0000-0000-000000000001"

func TestDecodeTuicURLUserinfoForms(t *testing.T) {
	tests := []struct {
		name     string
		link     string
		insecure bool
	}{
		{
			name: "literal separator",
			link: "tuic://" + testTUICUUID + ":secret@[2001:db8::1]:41016?alpn=h3#literal",
		},
		{
			name:     "v2rayN escaped separator",
			link:     "tuic://" + testTUICUUID + "%3Asecret@[2001:db8::1]:41016?alpn=h3&allow_insecure=1&congestion_control=bbr&udp_relay_mode=native#escaped",
			insecure: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DecodeTuicURL(tt.link)
			if err != nil {
				t.Fatal(err)
			}
			if got.Uuid != testTUICUUID || got.Password != "secret" || got.Host != "2001:db8::1" || got.Port != 41016 {
				t.Fatalf("TUIC credentials or IPv6 endpoint decoded incorrectly: %#v", got)
			}
			if got.SkipCertVerify != tt.insecure || len(got.Alpn) != 1 || got.Alpn[0] != "h3" {
				t.Fatalf("TUIC options decoded incorrectly: %#v", got)
			}
			if tt.insecure && (got.Congestion_control != "bbr" || got.Udp_relay_mode != "native") {
				t.Fatalf("v2rayN TUIC options decoded incorrectly: %#v", got)
			}
		})
	}
	for _, link := range []string{
		"tuic://[2001:db8::1]:41016",
		"tuic://" + testTUICUUID + "%3A@[2001:db8::1]:41016",
	} {
		if _, err := DecodeTuicURL(link); err == nil {
			t.Fatal("TUIC link without a password was accepted")
		}
	}
}

func TestEncodeClashTUICV2rayNLink(t *testing.T) {
	template := filepath.Join(t.TempDir(), "clash.yaml")
	if err := os.WriteFile(template, []byte("proxies: []\nproxy-groups: []\n"), 0600); err != nil {
		t.Fatal(err)
	}
	links := []string{
		"tuic://" + testTUICUUID + "%3Asecret@[2001:db8::1]:41016?alpn=h3&allow_insecure=1&congestion_control=bbr&udp_relay_mode=native#escaped",
		"tuic://" + testTUICUUID + ":other@example.test:41016?alpn=h3#literal",
	}
	data, err := EncodeClash(links, SqlConfig{Clash: template})
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		Proxies []map[string]interface{} `yaml:"proxies"`
	}
	if err := yaml.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	if len(config.Proxies) != 2 {
		t.Fatalf("expected two TUIC nodes, got %d", len(config.Proxies))
	}
	escaped, literal := config.Proxies[0], config.Proxies[1]
	if escaped["uuid"] != testTUICUUID || escaped["password"] != "secret" || escaped["skip-cert-verify"] != true || escaped["congestion-controller"] != "bbr" || escaped["udp-relay-mode"] != "native" {
		t.Fatalf("v2rayN TUIC link produced wrong Clash fields: %#v", escaped)
	}
	if literal["uuid"] != testTUICUUID || literal["password"] != "other" || literal["skip-cert-verify"] != nil {
		t.Fatalf("ordinary TUIC link produced wrong Clash fields: %#v", literal)
	}
}
