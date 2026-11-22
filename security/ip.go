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

// Non-public special-use ranges not covered by IsPrivate/IsGlobalUnicast.
// See the IANA IPv4 and IPv6 Special-Purpose Address Registries.
var nonPublicLoginRanges = [...]netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("192.88.99.0/24"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("64:ff9b:1::/48"),
	netip.MustParsePrefix("100::/64"),
	netip.MustParsePrefix("100:0:0:1::/64"),
	netip.MustParsePrefix("2001:2::/48"),
	netip.MustParsePrefix("2001:db8::/32"),
	netip.MustParsePrefix("3fff::/20"),
	netip.MustParsePrefix("5f00::/16"),
}

func isPublicLoginIP(value string) bool {
	ip, err := netip.ParseAddr(value)
	if err != nil || ip.Zone() != "" {
		return false
	}
	ip = ip.Unmap()
	if !ip.IsGlobalUnicast() || ip.IsPrivate() {
		return false
	}
	// Globally reachable exceptions inside otherwise non-public IPv4 ranges.
	if ip == netip.MustParseAddr("192.0.0.9") || ip == netip.MustParseAddr("192.0.0.10") {
		return true
	}
	for _, prefix := range nonPublicLoginRanges {
		if prefix.Contains(ip) {
			return false
		}
	}
	return true
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
