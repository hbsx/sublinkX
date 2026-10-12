package node

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

type Tuic struct {
	Name               string
	Password           string
	Host               string
	Port               int
	Uuid               string
	SkipCertVerify     bool
	Congestion_control string
	Alpn               []string
	Sni                string
	Udp_relay_mode     string
	Disable_sni        int
}

// Tuic 解码
func DecodeTuicURL(s string) (Tuic, error) {
	u, err := url.Parse(s)
	if err != nil {
		return Tuic{}, fmt.Errorf("TUIC URL 解析失败")
	}
	if u.Scheme != "tuic" {
		return Tuic{}, fmt.Errorf("非 TUIC 协议")
	}
	if err := validateEndpoint(u, true); err != nil {
		return Tuic{}, err
	}
	uuid := u.User.Username()
	password, hasPassword := u.User.Password()
	// v2rayN escapes the separator as %3A, leaving both values in Username().
	if !hasPassword {
		if id, secret, ok := strings.Cut(uuid, ":"); ok {
			uuid, password = id, secret
		}
	}
	if uuid == "" || password == "" {
		return Tuic{}, fmt.Errorf("TUIC 节点缺少 UUID 或密码")
	}
	server := u.Hostname()
	port, _ := validPort(u.Port())
	q := u.Query()
	Congestioncontrol := q.Get("congestion_control")
	if Congestioncontrol == "" {
		Congestioncontrol = q.Get("Congestion_control")
	}
	alpns := q.Get("alpn")
	alpn := strings.Split(alpns, ",")
	if alpns == "" {
		alpn = nil
	}
	sni := q.Get("sni")
	Udprelay_mode := q.Get("udp_relay_mode")
	if Udprelay_mode == "" {
		Udprelay_mode = q.Get("Udp_relay_mode")
	}
	disableSNI := q.Get("disable_sni")
	if disableSNI == "" {
		disableSNI = q.Get("Disable_sni")
	}
	Disablesni, _ := strconv.Atoi(disableSNI)
	name := u.Fragment
	// 如果没有设置 Name，则使用 Host:Port 作为 Fragment
	if name == "" {
		name = server + ":" + u.Port()
	}
	return Tuic{
		Name:               name,
		Password:           password,
		Host:               server,
		Port:               port,
		Uuid:               uuid,
		SkipCertVerify:     queryBool(q.Get("allow_insecure")) || queryBool(q.Get("allowInsecure")) || queryBool(q.Get("insecure")),
		Congestion_control: Congestioncontrol,
		Alpn:               alpn,
		Sni:                sni,
		Udp_relay_mode:     Udprelay_mode,
		Disable_sni:        Disablesni,
	}, nil
}
