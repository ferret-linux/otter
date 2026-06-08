package netcheck

import (
	"fmt"
	"net/http"
	"time"
)

func checkHTTP() error {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Head("https://cloudflare.com")
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode >= 500 {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}
	return nil
}
