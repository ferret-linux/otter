package netcheck

import "net"

var dnsTargets = []string{
	"cloudflare.com",
	"google.com",
	"github.com",
	"ghcr.io",
}

func checkDNS() error {
	return firstSuccess(len(dnsTargets), func(i int) error {
		_, err := net.LookupHost(dnsTargets[i])
		return err
	})
}
