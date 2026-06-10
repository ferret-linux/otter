package netcheck

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

//nolint:gochecknoglobals // package-level HTTP target list is effectively a constant
var httpTargets = []string{
	"http://cloudflare.com",
	"http://google.com",
	"http://github.com",
	"http://example.com",
}

func checkHTTP(ctx context.Context) error {
	client := &http.Client{Timeout: 5 * time.Second}
	return firstSuccess(len(httpTargets), func(i int) error {
		req, err := http.NewRequestWithContext(ctx, http.MethodHead, httpTargets[i], nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("HTTP check failed for %s: %w", httpTargets[i], err)
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
