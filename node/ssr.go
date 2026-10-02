package node

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

func CallSSRURL() {
	ssr := new(Ssr)
	ssr.Server = "xx.com"
	ssr.Port = 443
	ssr.Protocol = "auth_aes128_md5"
	ssr.Method = "aes-256-cfb"
	ssr.Obfs = "tls1.2_ticket_auth"
	ssr.Password = "123456"
	ssr.Qurey = Ssrquery{
		Obfsparam: "",
		Remarks:   "没有名字",
	}
	cc := EncodeSSRURL(*ssr)
	fmt.Println(cc)
}

// ssr格式编码输出
func EncodeSSRURL(s Ssr) string {
	/*编码格式
	ssr://base64(host:port:protocol:method:obfs:base64(password)/?obfsparam=base64(obfsparam)&protoparam=base64(protoparam)&remarks=base64(remarks)&group=base64(group))
	*/
	obfsparam := "obfsparam=" + Base64Encode(s.Qurey.Obfsparam)
	remarks := "remarks=" + Base64Encode(s.Qurey.Remarks)
	// 如果没有备注默认使用服务器+端口作为备注
	if s.Qurey.Remarks == "" {
		server_port := Base64Encode(s.Server + ":" + strconv.Itoa(s.Port))
		remarks = fmt.Sprintf("remarks=%s", server_port)
	}
	param := fmt.Sprintf("%s:%d:%s:%s:%s:%s/?%s&%s",
		s.Server,
		s.Port,
		s.Protocol,
		s.Method,
		s.Obfs,
		Base64Encode(s.Password),
		obfsparam,
		remarks,
	)
	return "ssr://" + Base64Encode(param)

}

// ssr解码
func DecodeSSRURL(raw string) (Ssr, error) {
	if !strings.HasPrefix(raw, "ssr://") {
		return Ssr{}, errors.New("SSR 协议无效")
	}
	decoded, err := decodeBase64(strings.TrimPrefix(raw, "ssr://"))
	if err != nil {
		return Ssr{}, err
	}
	body, query, _ := strings.Cut(decoded, "/?")
	values := make(map[string]string)
	if query != "" {
		for _, part := range strings.Split(query, "&") {
			key, value, ok := strings.Cut(part, "=")
			if !ok || key == "" {
				return Ssr{}, errors.New("SSR 查询参数无效")
			}
			if value != "" {
				value, err = decodeBase64(value)
				if err != nil {
					return Ssr{}, err
				}
			}
			values[key] = value
		}
	}
	fields := strings.Split(body, ":")
	if len(fields) < 6 {
		return Ssr{}, errors.New("SSR 字段不足")
	}
	offset := len(fields) - 5
	server := strings.Trim(strings.Join(fields[:offset], ":"), "[]")
	port, err := validPort(fields[offset])
	if err != nil {
		return Ssr{}, err
	}
	password, err := decodeBase64(fields[offset+4])
	if err != nil || password == "" {
		return Ssr{}, errors.New("SSR 密码无效")
	}
	if server == "" || fields[offset+1] == "" || fields[offset+2] == "" || fields[offset+3] == "" {
		return Ssr{}, errors.New("SSR 必要字段为空")
	}
	name := values["remarks"]
	if name == "" {
		name = fmt.Sprintf("%s:%d", server, port)
	}
	return Ssr{Server: server, Port: port, Protocol: fields[offset+1], Method: fields[offset+2], Obfs: fields[offset+3], Password: password,
		Qurey: Ssrquery{Remarks: name, Obfsparam: values["obfsparam"]}, Type: "ssr"}, nil
}

type Ssr struct {
	Server   string
	Port     int
	Protocol string
	Method   string
	Obfs     string
	Password string
	Qurey    Ssrquery
	Type     string
}
type Ssrquery struct {
	Obfsparam string
	Remarks   string
}
