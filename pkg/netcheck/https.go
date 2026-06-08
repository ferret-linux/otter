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
	return firstSuccess(len(httpsTargets), func(i int) error {
		resp, err := client.Head(httpsTargets[i])
		if err != nil {
			return err
		}
		resp.Body.Close()
		if resp.StatusCode >= 500 {
			return errAllFailed
		}
		return nil
	})
}
