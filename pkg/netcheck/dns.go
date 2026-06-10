package netcheck

import (
	"context"
	"fmt"
	"net"
)

//nolint:gochecknoglobals // package-level DNS target list is effectively a constant
var dnsTargets = []string{
	"cloudflare.com",
	"google.com",
	"github.com",
	"ghcr.io",
}

func checkDNS(ctx context.Context) error {
	resolver := &net.Resolver{}
	return firstSuccess(len(dnsTargets), func(i int) error {
		_, err := resolver.LookupHost(ctx, dnsTargets[i])
		if err != nil {
			return fmt.Errorf("DNS lookup failed for %s: %w", dnsTargets[i], err)
		}
		return nil
	})
}
