package client

import (
	"net"
)

// IsPrivateIP 判断是否为内网IP
func IsPrivateIP(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsPrivate()
}

// IsCommonDNS 判断是否为常见的DNS地址
func IsCommonDNS(ip string) bool {
	commonDNS := []string{
		"1.1.1.1",
		"1.0.0.1",
		"8.8.8.8",
		"8.8.4.4",
		"0.0.0.0",
	}
	for _, dns := range commonDNS {
		if dns == ip {
			return true
		}
	}
	return false
}
