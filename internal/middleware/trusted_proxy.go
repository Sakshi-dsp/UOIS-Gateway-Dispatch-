package middleware

import (
	"net"
)

type TrustedProxyList struct {
	trustedIPs []*net.IPNet
}

func NewTrustedProxyList(cidrs []string) (*TrustedProxyList, error) {
	trustedIPs := make([]*net.IPNet, 0, len(cidrs))
	for _, cidr := range cidrs {
		if cidr == "" {
			continue
		}
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			return nil, err
		}
		trustedIPs = append(trustedIPs, ipNet)
	}
	return &TrustedProxyList{trustedIPs: trustedIPs}, nil
}

func (t *TrustedProxyList) IsTrustedProxy(remoteAddr string) bool {
	if len(t.trustedIPs) == 0 {
		return false
	}

	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		// Fallback: maybe it's already a raw IP (no port)
		host = remoteAddr
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}

	for _, trustedNet := range t.trustedIPs {
		if trustedNet.Contains(ip) {
			return true
		}
	}

	return false
}
