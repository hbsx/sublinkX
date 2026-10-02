package node

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type Vmess struct {
	Add  string      `json:"add,omitempty"` // 服务器地址
	Aid  interface{} `json:"aid,omitempty"`
	Alpn string      `json:"alpn,omitempty"`
	Fp   string      `json:"fp,omitempty"`
	Host string      `json:"host,omitempty"`
	Id   string      `json:"id,omitempty"`
	Net  string      `json:"net,omitempty"`
	Path string      `json:"path,omitempty"`
	Port interface{} `json:"port,omitempty"`
	Ps   string      `json:"ps,omitempty"`
	Scy  string      `json:"scy,omitempty"`
	Sni  string      `json:"sni,omitempty"`
	Tls  string      `json:"tls,omitempty"`
	Type string      `json:"type,omitempty"`
	V    string      `json:"v,omitempty"`
}

// 开发者测试
func CallVmessURL() {
	vmess := Vmess{
		Add:  "xx.xxx.ru",
		Port: "2095",
		Aid:  0,
		Scy:  "auto",
		Net:  "ws",
		Type: "none",
		Id:   "7a737f41-b792-4260-94ff-3d864da67380",
		Host: "xx.xxx.ru",
		Path: "/",
		Tls:  "",
	}
	fmt.Println(EncodeVmessURL(vmess))
}

// vmess 编码
func EncodeVmessURL(v Vmess) string {
	// 如果备注为空，则使用服务器地址+端口
	if v.Ps == "" {
		v.Ps = fmt.Sprintf("%s:%v", v.Add, v.Port)
	}
	// 如果版本为空，则默认为2
	if v.V == "" {
		v.V = "2"
	}
	param, _ := json.Marshal(v)
	return "vmess://" + Base64Encode(string(param))
}

// vmess 解码
func DecodeVMESSURL(s string) (Vmess, error) {
	if !strings.HasPrefix(s, "vmess://") {
		return Vmess{}, fmt.Errorf("非 VMess 协议")
	}
	param, err := decodeBase64(strings.TrimPrefix(s, "vmess://"))
	if err != nil {
		return Vmess{}, err
	}
	var vmess Vmess
	err = json.Unmarshal([]byte(param), &vmess)
	if err != nil {
		return Vmess{}, fmt.Errorf("VMess JSON 无效")
	}
	port, err := convertToInt(vmess.Port)
	if err != nil || port < 1 || port > 65535 || vmess.Add == "" || vmess.Id == "" {
		return Vmess{}, fmt.Errorf("VMess 服务器、端口或认证信息无效")
	}
	vmess.Port = strconv.Itoa(port)
	if vmess.Scy == "" {
		vmess.Scy = "auto"
	}
	// 如果备注为空，则使用服务器地址+端口
	if vmess.Ps == "" {
		vmess.Ps = fmt.Sprintf("%s:%d", vmess.Add, port)
	}
	return vmess, nil
}
