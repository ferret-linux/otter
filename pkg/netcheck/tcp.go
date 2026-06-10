package netcheck

import (
	"context"
	"fmt"
	"net"
	"time"
)

//nolint:gochecknoglobals // package-level TCP target list is effectively a constant
var tcpTargets = []string{
	"1.1.1.1:443",
	"8.8.8.8:443",
	"140.82.112.4:443",
	"185.199.108.133:443",
}

func checkTCP(ctx context.Context) error {
	dialer := &net.Dialer{Timeout: 3 * time.Second}
	return firstSuccess(len(tcpTargets), func(i int) error {
		conn, err := dialer.DialContext(ctx, "tcp", tcpTargets[i])
		if err != nil {
			return fmt.Errorf("TCP check failed for %s: %w", tcpTargets[i], err)
		}
		if err := conn.Close(); err != nil {
			return fmt.Errorf("failed to close TCP connection: %w", err)
		}
		return nil
	})
}
