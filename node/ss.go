package node

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

type Ss struct {
	Param  Param
	Server string
	Port   int
	Name   string
	Type   string
}
type Param struct {
	Cipher   string
	Password string
}

func CallSSURL() {
	fmt.Println(EncodeSSURL(Ss{Server: "example.test", Port: 443, Param: Param{Cipher: "aes-128-gcm", Password: "example"}}))
}
func EncodeSSURL(s Ss) string {
	if s.Name == "" {
		s.Name = net.JoinHostPort(s.Server, strconv.Itoa(s.Port))
	}
	u := url.URL{Scheme: "ss", User: url.User(Base64Encode(s.Param.Cipher + ":" + s.Param.Password)), Host: net.JoinHostPort(s.Server, strconv.Itoa(s.Port)), Fragment: s.Name}
	return u.String()
}
func DecodeSSURL(raw string) (Ss, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "ss" {
		return Ss{}, fmt.Errorf("SS 地址无效")
	}
	if u.User == nil {
		payload, _, _ := strings.Cut(strings.TrimPrefix(raw, "ss://"), "#")
		decoded, e := decodeBase64(payload)
		if e != nil {
			return Ss{}, fmt.Errorf("SS 编码无效")
		}
		name := u.Fragment
		u, err = url.Parse("ss://" + decoded)
		if err != nil {
			return Ss{}, fmt.Errorf("SS 地址无效")
		}
		if name != "" {
			u.Fragment = name
		}
	}
	if err = validateEndpoint(u, true); err != nil {
		return Ss{}, err
	}
	port, _ := validPort(u.Port())
	auth := u.User.Username()
	if password, ok := u.User.Password(); ok {
		auth += ":" + password
	} else {
		auth, err = decodeBase64(auth)
		if err != nil {
			return Ss{}, fmt.Errorf("SS 认证编码无效")
		}
	}
	cipher, password, ok := strings.Cut(auth, ":")
	if !ok || cipher == "" || password == "" {
		return Ss{}, fmt.Errorf("SS 认证信息无效")
	}
	name := u.Fragment
	if name == "" {
		name = net.JoinHostPort(u.Hostname(), strconv.Itoa(port))
	}
	return Ss{Param: Param{Cipher: cipher, Password: password}, Server: u.Hostname(), Port: port, Name: name, Type: "ss"}, nil
}
