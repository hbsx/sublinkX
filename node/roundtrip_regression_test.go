package node

import (
	"reflect"
	"testing"
)

func TestURLRoundtripsPreserveIPv6AndTransportFields(t *testing.T) {
	for _, host := range []string{"example.test", "2001:db8::1234"} {
		v := VLESS{Server: host, Port: 443, Uuid: "test-id", Name: "VLESS", Query: VLESSQuery{Security: "tls", Sni: "tls.test", Alpn: []string{"h2", "http/1.1"}, Type: "grpc", ServiceName: "service/path", Mode: "multi"}}
		decoded, err := DecodeVLESSURL(EncodeVLESSURL(v))
		if err != nil || decoded.Server != host || decoded.Uuid != v.Uuid || !reflect.DeepEqual(decoded.Query, v.Query) {
			t.Fatalf("VLESS roundtrip lost fields: %+v %v", decoded, err)
		}
		tr := Trojan{Hostname: host, Port: 443, Password: "p@ss:word", Name: "Trojan", Query: TrojanQuery{Security: "tls", Sni: "tls.test", Alpn: []string{"h2", "http/1.1"}, AllowInsecure: 1}}
		td, err := DecodeTrojanURL(EncodeTrojanURL(tr))
		if err != nil || td.Hostname != host || td.Password != tr.Password || !reflect.DeepEqual(td.Query, tr.Query) {
			t.Fatalf("Trojan roundtrip lost fields: %+v %v", td, err)
		}
		hy := HY{Host: host, Port: 443, Auth: "auth", Name: "HY", ALPN: []string{"h3"}, Peer: "tls.test", UpMbps: 10, DownMbps: 20, Insecure: 1}
		hd, err := DecodeHYURL(EncodeHYURL(hy))
		if err != nil || !reflect.DeepEqual(hd, hy) {
			t.Fatalf("HY roundtrip lost fields: %+v %v", hd, err)
		}
		hy2 := HY2{Host: host, Port: 443, Password: "p@ss:word", Name: "HY2", ALPN: []string{"h3"}, Sni: "tls.test", Obfs: "salamander", ObfsPassword: "obfs-secret", Insecure: 1}
		h2d, err := DecodeHY2URL(EncodeHY2URL(hy2))
		if err != nil || !reflect.DeepEqual(h2d, hy2) {
			t.Fatalf("HY2 roundtrip lost fields: %+v %v", h2d, err)
		}
	}
}

func TestHysteria2PeerIsUsedAsSNIFallback(t *testing.T) {
	v, err := DecodeHY2URL("hysteria2://pass@example.test:443?peer=tls.test")
	if err != nil || v.Sni != "tls.test" || v.Peer != "tls.test" {
		t.Fatal("peer alias lost", v, err)
	}
}
