package netcheck

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

//nolint:gochecknoglobals // package-level HTTPS target list is effectively a constant
var httpsTargets = []string{
	"https://cloudflare.com",
	"https://google.com",
	"https://github.com",
	"https://ghcr.io",
}

func checkHTTPS(ctx context.Context) error {
	client := &http.Client{Timeout: 5 * time.Second}
	return firstSuccess(len(httpsTargets), func(i int) error {
		req, err := http.NewRequestWithContext(ctx, http.MethodHead, httpsTargets[i], nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("HTTPS check failed for %s: %w", httpsTargets[i], err)
		}
		if err := resp.Body.Close(); err != nil {
			return fmt.Errorf("failed to close response body: %w", err)
		}
		if resp.StatusCode >= 500 {
			return errAllFailed
		}
		return nil
	})
}
