package netcheck

import "net"

var dnsTargets = []string{
	"cloudflare.com",
	"google.com",
	"github.com",
	"ghcr.io",
}

func checkDNS() error {
	for _, host := range dnsTargets {
		if _, err := net.LookupHost(host); err == nil {
			return nil
		}
	}
	return errAllFailed
}
