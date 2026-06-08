package netcheck

import (
	"net/http"
	"time"
)

var httpTargets = []string{
	"http://cloudflare.com",
	"http://google.com",
	"http://github.com",
	"http://example.com",
}

func checkHTTP() error {
	client := &http.Client{Timeout: 5 * time.Second}
	return firstSuccess(len(httpTargets), func(i int) error {
		resp, err := client.Head(httpTargets[i])
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
