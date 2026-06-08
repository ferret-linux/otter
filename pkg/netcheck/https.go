package netcheck

import (
	"net/http"
	"time"
)

var httpsTargets = []string{
	"https://cloudflare.com",
	"https://google.com",
	"https://github.com",
	"https://ghcr.io",
}

func checkHTTPS() error {
	client := &http.Client{Timeout: 5 * time.Second}
	for _, url := range httpsTargets {
		resp, err := client.Head(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode < 500 {
				return nil
			}
		}
	}
	return errAllFailed
}
