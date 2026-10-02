package node

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParserBoundaryInputsReturnErrors(t *testing.T) {
	links := []string{
		"ss://" + Base64Encode("aes-128-gcm:password@host:invalid"),
		"ssr://" + Base64Encode("example.test:443:auth_aes128_md5:aes-256-cfb:plain:dGVzdA/?remarks"),
		"vmess://" + Base64Encode(`{"add":"example.test","port":443.5,"id":"dummy"}`),
		"vmess://" + Base64Encode(`{"add":"example.test","port":{},"id":"dummy"}`),
		"trojan://pass@example.test:70000", "vless://id@example.test:0", "hy://example.test:70000",
		"hy2://pass@example.test:0", "tuic://id:pass@example.test:70000", "not-a-proxy-url",
	}
	for _, link := range links {
		t.Run(strings.Split(link, "://")[0], func(t *testing.T) {
			if _, err := NodeName(link); err == nil {
				t.Fatal("invalid node accepted")
			}
		})
	}
	v, err := DecodeVMESSURL("vmess://" + Base64Encode(`{"add":"example.test","port":443,"id":"dummy"}`))
	if err != nil || v.Ps != "example.test:443" {
		t.Fatalf("numeric VMess port failed: %+v %v", v, err)
	}
}
func TestSSAndSSRRoundtripPreserveCredentialsAndIPv6(t *testing.T) {
	for _, server := range []string{"example.test", "2001:db8:443::1"} {
		ss := Ss{Server: server, Port: 443, Name: "IPv6, Name", Param: Param{Cipher: "aes-128-gcm", Password: "p@ss:word"}}
		decoded, err := DecodeSSURL(EncodeSSURL(ss))
		if err != nil || decoded.Server != ss.Server || decoded.Param != ss.Param || decoded.Name != ss.Name {
			t.Fatalf("SS roundtrip failed: %+v %v", decoded, err)
		}
		ssr := Ssr{Server: server, Port: 443, Protocol: "auth_aes128_md5", Method: "aes-256-cfb", Obfs: "plain", Password: "p@ss:word", Qurey: Ssrquery{Remarks: "SSR", Obfsparam: "tls.test"}}
		d, err := DecodeSSRURL(EncodeSSRURL(ssr))
		if err != nil || d.Server != server || d.Password != ssr.Password || d.Qurey != ssr.Qurey {
			t.Fatalf("SSR roundtrip failed: %+v %v", d, err)
		}
	}
}
func TestSubscriptionValidationAndCommaParameters(t *testing.T) {
	valid := "anytls://test-password@example.test:443?alpn=h2,http/1.1#Name with spaces"
	for _, body := range []string{valid, Base64Encode(valid), valid + ",ss://YWVzLTEyOC1nY206cGFzcw@example.test:443#Other"} {
		links, err := DecodeSubscription(body)
		if err != nil || links[0] != valid {
			t.Fatalf("valid subscription corrupted: %q %v", links, err)
		}
	}
	for _, body := range []string{"<html>Login required</html>", "<a href=\"https://example.test\">Login</a>", Base64Encode("Unauthorized"), "", valid + "\nerror"} {
		if _, err := DecodeSubscription(body); err == nil {
			t.Fatal("invalid subscription accepted")
		}
	}
}
func TestClashConversionPreservesSNIAndMakesNamesUnique(t *testing.T) {
	file := filepath.Join(t.TempDir(), "clash.yaml")
	os.WriteFile(file, []byte("proxies: []\nproxy-groups:\n- name: All\n  type: select\n  proxies: [DIRECT]\n"), 0600)
	vmess := "vmess://" + Base64Encode(`{"add":"example.test","port":443,"id":"dummy","ps":"Same","tls":"tls","sni":"tls.example.test","alpn":"h2,http/1.1"}`)
	result, err := EncodeClash([]string{vmess, "ss://YWVzLTEyOC1nY206cGFzcw@second.test:443#Same"}, SqlConfig{Clash: file})
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		Proxies []Proxy
		Groups  []ProxyGroup `yaml:"proxy-groups"`
	}
	if err := yaml.Unmarshal(result, &config); err != nil {
		t.Fatal(err)
	}
	if len(config.Proxies) != 2 || config.Proxies[0].Servername != "tls.example.test" || len(config.Proxies[0].Alpn) != 2 || config.Proxies[0].Name == config.Proxies[1].Name {
		t.Fatalf("conversion lost fields or names: %s", result)
	}
	if len(config.Groups[0].Proxies) != 3 || config.Groups[0].Proxies[2] != config.Proxies[1].Name {
		t.Fatal("renamed proxy missing from group")
	}
	for _, link := range []string{"not-a-url", "ssr://bad"} {
		if _, err := EncodeClash([]string{link}, SqlConfig{Clash: file}); err == nil {
			t.Fatal("bad node silently ignored")
		}
	}
}
func FuzzNodeParsersDoNotPanic(f *testing.F) {
	for _, seed := range []string{"", "remarks", "aes-128-gcm:password@host:invalid", `{"add":"a","port":443,"id":"id"}`, "foo:443:p:m:o:dGVzdA/?remarks"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, body string) {
		for _, scheme := range []string{"ss", "ssr", "vmess"} {
			NodeName(fmt.Sprintf("%s://%s", scheme, Base64Encode(body)))
		}
	})
}

func TestHysteria2DefaultPortAndUserpass(t *testing.T) {
	for _, link := range []string{"hy2://example.test", "hysteria2://user:pass@example.test"} {
		v, err := DecodeHY2URL(link)
		if err != nil || v.Port != 443 || v.Host != "example.test" {
			t.Fatalf("default port failed: %+v %v", v, err)
		}
		if strings.Contains(link, "user:pass") && v.Password != "user:pass" {
			t.Fatal("userpass authentication truncated")
		}
	}
}

func TestSurgeTemplateNameCollisionsKeepReferences(t *testing.T) {
	file := filepath.Join(t.TempDir(), "surge.conf")
	if err := os.WriteFile(file, []byte("[Proxy]\nSame = direct\n[Proxy Group]\nAll = select, Same\n[Rule]\nFINAL,All\n"), 0600); err != nil {
		t.Fatal(err)
	}
	result, err := EncodeSurge([]string{"ss://YWVzLTEyOC1nY206cGFzcw@example.test:443#Same", "ss://YWVzLTEyOC1nY206cGFzcw@other.test:443#Same"}, SqlConfig{Surge: file})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result, "Same (2) = ss,") || !strings.Contains(result, "Same (2) (2) = ss,") || !strings.Contains(result, "All = select, Same, Same (2),Same (2) (2)") {
		t.Fatalf("proxy names and references mismatch: %s", result)
	}
}
