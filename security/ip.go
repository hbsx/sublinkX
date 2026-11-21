package security

import (
	"errors"
	"net"
	"net/http"
	"net/netip"
	"strings"
)

// Forwarded headers are accepted only from explicitly configured immediate peers.
// With Docker, configure the connector's actual peer address, not the entire LAN.
type IPResolver struct {
	trusted []netip.Prefix
	header  string
}

func NewIPResolver(peers, header string) (*IPResolver, error) {
	if header == "" {
		header = "CF-Connecting-IP"
	}
	if !strings.EqualFold(header, "CF-Connecting-IP") && !strings.EqualFold(header, "X-Forwarded-For") {
		return nil, errors.New("login IP header must be CF-Connecting-IP or X-Forwarded-For")
	}
	r := &IPResolver{header: header}
	if strings.TrimSpace(peers) == "" {
		return r, nil
	}
	for _, text := range strings.Split(peers, ",") {
		text = strings.TrimSpace(text)
		prefix, err := netip.ParsePrefix(text)
		if err != nil {
			ip, ipErr := netip.ParseAddr(text)
			if ipErr != nil || ip.Zone() != "" {
				return nil, errors.New("invalid trusted login proxy")
			}
			ip = ip.Unmap()
			prefix = netip.PrefixFrom(ip, ip.BitLen())
		}
		if prefix.Bits() == 0 {
			return nil, errors.New("trusting all login peers is not allowed")
		}
		r.trusted = append(r.trusted, prefix)
	}
	return r, nil
}

func (r *IPResolver) trusts(ip netip.Addr) bool {
	for _, prefix := range r.trusted {
		if prefix.Contains(ip) {
			return true
		}
	}
	return false
}

func (r *IPResolver) Resolve(request *http.Request) (string, string) {
	host, _, err := net.SplitHostPort(request.RemoteAddr)
	if err != nil {
		host = request.RemoteAddr
	}
	peer, err := netip.ParseAddr(host)
	if err != nil {
		return "unknown", "direct"
	}
	peer = peer.Unmap()
	if r != nil && r.trusts(peer) {
		values := request.Header.Values(r.header)
		if len(values) != 1 || len(values[0]) > 2048 {
			return peer.String(), "direct"
		}
		parts := strings.Split(values[0], ",")
		if strings.EqualFold(r.header, "CF-Connecting-IP") && len(parts) != 1 {
			return peer.String(), "direct"
		}
		var chain []netip.Addr
		for _, part := range parts {
			ip, err := netip.ParseAddr(strings.TrimSpace(part))
			if err != nil || ip.Zone() != "" {
				return peer.String(), "direct"
			}
			chain = append(chain, ip.Unmap())
		}
		for i := len(chain) - 1; i >= 0; i-- {
			if !r.trusts(chain[i]) || i == 0 {
				return chain[i].String(), r.header
			}
		}
	}
	return peer.String(), "direct"
}
