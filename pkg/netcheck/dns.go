package netcheck

import "net"

func checkDNS() error {
	_, err := net.LookupHost("cloudflare.com")
	return err
}
