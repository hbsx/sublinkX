package node

import (
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestClashRejectsInvalidGroupsAndKeepsGroupsAfterRelay(t *testing.T) {
	file := filepath.Join(t.TempDir(), "clash.yaml")
	for _, source := range []string{"proxies: []", "proxy-groups: wrong", "proxy-groups: [wrong]", "proxy-groups:\n- name: bad\n  proxies: wrong"} {
		os.WriteFile(file, []byte(source), 0600)
		if _, err := DecodeClash(nil, file); err == nil {
			t.Fatalf("invalid template accepted: %s", source)
		}
	}
	source := "proxies: []\nproxy-groups:\n- name: relay\n  type: relay\n  proxies: [DIRECT]\n- name: normal\n  type: select\n  proxies: [DIRECT]\n"
	os.WriteFile(file, []byte(source), 0600)
	result, err := DecodeClash([]Proxy{{Name: "NewNode", Type: "ss"}}, file)
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		Groups []struct {
			Proxies []string `yaml:"proxies"`
		} `yaml:"proxy-groups"`
	}
	if err := yaml.Unmarshal(result, &config); err != nil {
		t.Fatal(err)
	}
	if len(config.Groups[0].Proxies) != 1 || len(config.Groups[1].Proxies) != 2 {
		t.Fatalf("groups were incorrectly modified: %s", result)
	}
}

func TestSurgeProducesOneRuleSectionAndPreservesRules(t *testing.T) {
	file := filepath.Join(t.TempDir(), "surge.conf")
	os.WriteFile(file, []byte("[Proxy]\nDIRECT = direct\n[Proxy Group]\nAll = select, DIRECT\n[Rule]\nFINAL,All\n"), 0600)
	result, err := DecodeSurge([]string{"NewNode = ss, example.test, 443"}, []string{"NewNode"}, file)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(result, "[Rule]") != 1 || strings.Count(result, "All = select") != 1 || !strings.Contains(result, "All = select, DIRECT, NewNode") || !strings.Contains(result, "FINAL,All") {
		t.Fatal(result)
	}
}
