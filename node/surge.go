package node

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sublink/utils"
)

func EncodeSurge(urls []string, sqlconfig SqlConfig) (string, error) {
	var proxys, groups []string
	usedNames := map[string]bool{"DIRECT": true, "REJECT": true}
	for _, link := range urls {
		if _, err := NodeName(link); err != nil {
			return "", fmt.Errorf("节点解析失败")
		}
		Scheme := strings.Split(link, "://")[0]
		switch {
		case Scheme == "ss":
			ss, err := DecodeSSURL(link)
			if err != nil {
				return "", fmt.Errorf("节点解析失败")
			}
			ss.Name = uniqueName(ss.Name, usedNames)
			proxy := map[string]interface{}{
				"name":     ss.Name,
				"server":   ss.Server,
				"port":     ss.Port,
				"cipher":   ss.Param.Cipher,
				"password": ss.Param.Password,
				"udp":      sqlconfig.Udp,
			}
			ssproxy := fmt.Sprintf("%s = ss, %s, %d, encrypt-method=%s, password=%s, udp-relay=%t",
				proxy["name"], proxy["server"], proxy["port"], proxy["cipher"], proxy["password"], proxy["udp"])
			groups = append(groups, ss.Name)
			proxys = append(proxys, ssproxy)
		case Scheme == "vmess":
			vmess, err := DecodeVMESSURL(link)
			if err != nil {
				return "", fmt.Errorf("节点解析失败")
			}
			tls := false
			if vmess.Tls != "none" && vmess.Tls != "" {
				tls = true
			}
			port, _ := convertToInt(vmess.Port)
			vmess.Ps = uniqueName(vmess.Ps, usedNames)
			proxy := map[string]interface{}{
				"name":             vmess.Ps,
				"server":           vmess.Add,
				"port":             port,
				"uuid":             vmess.Id,
				"tls":              tls,
				"network":          vmess.Net,
				"ws-path":          vmess.Path,
				"ws-host":          vmess.Host,
				"udp":              sqlconfig.Udp,
				"skip-cert-verify": sqlconfig.Cert,
			}
			vmessproxy := fmt.Sprintf("%s = vmess, %s, %d, username=%s , tls=%t, vmess-aead=true,  udp-relay=%t , skip-cert-verify=%t",
				proxy["name"], proxy["server"], proxy["port"], proxy["uuid"], proxy["tls"], proxy["udp"], proxy["skip-cert-verify"])
			if vmess.Net == "ws" {
				vmessproxy = fmt.Sprintf("%s, ws=true,ws-path=%s", vmessproxy, proxy["ws-path"])
				if vmess.Host != "" && vmess.Host != "none" {
					vmessproxy = fmt.Sprintf("%s, ws-headers=Host:%s", vmessproxy, proxy["ws-host"])
				}
			}
			if vmess.Sni != "" {
				vmessproxy = fmt.Sprintf("%s, sni=%s", vmessproxy, vmess.Sni)
			}
			groups = append(groups, vmess.Ps)
			proxys = append(proxys, vmessproxy)
		case Scheme == "trojan":
			trojan, err := DecodeTrojanURL(link)
			if err != nil {
				return "", fmt.Errorf("节点解析失败")
			}
			trojan.Name = uniqueName(trojan.Name, usedNames)
			proxy := map[string]interface{}{
				"name":             trojan.Name,
				"server":           trojan.Hostname,
				"port":             trojan.Port,
				"password":         trojan.Password,
				"udp":              sqlconfig.Udp,
				"skip-cert-verify": sqlconfig.Cert,
			}
			trojanproxy := fmt.Sprintf("%s = trojan, %s, %d, password=%s, udp-relay=%t, skip-cert-verify=%t",
				proxy["name"], proxy["server"], proxy["port"], proxy["password"], proxy["udp"], proxy["skip-cert-verify"])
			if trojan.Query.Sni != "" {
				trojanproxy = fmt.Sprintf("%s, sni=%s", trojanproxy, trojan.Query.Sni)

			}
			groups = append(groups, trojan.Name)
			proxys = append(proxys, trojanproxy)
		case Scheme == "hysteria2" || Scheme == "hy2":
			hy2, err := DecodeHY2URL(link)
			if err != nil {
				return "", fmt.Errorf("节点解析失败")
			}
			hy2.Name = uniqueName(hy2.Name, usedNames)
			proxy := map[string]interface{}{
				"name":             hy2.Name,
				"server":           hy2.Host,
				"port":             hy2.Port,
				"password":         hy2.Password,
				"udp":              sqlconfig.Udp,
				"skip-cert-verify": sqlconfig.Cert,
			}
			hy2proxy := fmt.Sprintf("%s = hysteria2, %s, %d, password=%s, udp-relay=%t, skip-cert-verify=%t",
				proxy["name"], proxy["server"], proxy["port"], proxy["password"], proxy["udp"], proxy["skip-cert-verify"])
			if hy2.Sni != "" {
				hy2proxy = fmt.Sprintf("%s, sni=%s", hy2proxy, hy2.Sni)

			}
			groups = append(groups, hy2.Name)
			proxys = append(proxys, hy2proxy)
		case Scheme == "tuic":
			tuic, err := DecodeTuicURL(link)
			if err != nil {
				return "", fmt.Errorf("节点解析失败")
			}
			tuic.Name = uniqueName(tuic.Name, usedNames)
			proxy := map[string]interface{}{
				"name":             tuic.Name,
				"server":           tuic.Host,
				"port":             tuic.Port,
				"password":         tuic.Password,
				"udp":              sqlconfig.Udp,
				"skip-cert-verify": sqlconfig.Cert,
			}
			tuicproxy := fmt.Sprintf("%s = tuic, %s, %d, token=%s, udp-relay=%t, skip-cert-verify=%t",
				proxy["name"], proxy["server"], proxy["port"], proxy["password"], proxy["udp"], proxy["skip-cert-verify"])
			groups = append(groups, tuic.Name)
			proxys = append(proxys, tuicproxy)
		}
	}
	if len(proxys) == 0 {
		return "", fmt.Errorf("没有 Surge 支持的节点")
	}
	return DecodeSurge(proxys, groups, sqlconfig.Surge)
}
func DecodeSurge(proxys, groups []string, file string) (string, error) {
	var surge []byte
	var err error
	if strings.Contains(file, "://") {
		surge, err = utils.Fetch(context.Background(), file, 2<<20)
		if err != nil {
			return "", err
		}
	} else {
		surge, err = os.ReadFile(file)
		if err != nil {
			log.Println("节点或模板解析失败")
			return "", err
		}
	}

	lines := strings.Split(strings.ReplaceAll(string(surge), "\r\n", "\n"), "\n")
	reserved := map[string]bool{"DIRECT": true, "REJECT": true}
	currentSection := ""
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			currentSection = trimmed
			continue
		}
		if (currentSection == "[Proxy]" || currentSection == "[Proxy Group]") && !strings.HasPrefix(trimmed, "#") && !strings.HasPrefix(trimmed, ";") {
			if name, _, ok := strings.Cut(trimmed, "="); ok {
				reserved[strings.TrimSpace(name)] = true
			}
		}
	}
	for i, name := range groups {
		renamed := uniqueName(name, reserved)
		if i < len(proxys) {
			proxys[i] = strings.Replace(proxys[i], name+" =", renamed+" =", 1)
		}
		groups[i] = renamed
	}
	var output []string
	section := ""
	foundProxy, foundGroup := false, false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			section = trimmed
			output = append(output, line)
			if section == "[Proxy]" {
				foundProxy = true
				output = append(output, proxys...)
			}
			if section == "[Proxy Group]" {
				foundGroup = true
			}
			continue
		}
		if section == "[Proxy Group]" && strings.Contains(line, "=") && !strings.HasPrefix(trimmed, "#") && !strings.HasPrefix(trimmed, ";") && len(groups) > 0 {
			line = strings.TrimRight(line, " \t") + ", " + strings.Join(groups, ",")
		}
		output = append(output, line)
	}
	if !foundProxy || !foundGroup {
		return "", fmt.Errorf("Surge 模板缺少 Proxy 或 Proxy Group 段")
	}
	return strings.Join(output, "\n"), nil
}
