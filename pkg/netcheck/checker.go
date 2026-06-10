package netcheck

import (
	"context"
	"errors"
	"sync"

	"github.com/ferret-linux/otter/pkg/ui"
)

var errAllFailed = errors.New("all targets failed")

// Check runs all connectivity checks and returns an error if any fail.
// Each check passes if at least one of its targets succeeds.
// All four checks run in parallel.
//
// Evaluation:
//   - If both DNS and TCP fail, return immediately without waiting for HTTP/HTTPS.
//   - If HTTPS passes, report success immediately.
//   - If HTTP passes, wait for HTTPS; if HTTPS fails, report fail.
//   - If both HTTP and HTTPS fail, report fail.
func Check(ctx context.Context) error {
	type result struct {
		name string
		err  error
	}

	dnsResult := make(chan result, 1)
	tcpResult := make(chan result, 1)
	httpResult := make(chan result, 1)
	httpsResult := make(chan result, 1)

	ui.DefaultLogger.Info("verifying network availability...")

	go func() { dnsResult <- result{"DNS", checkDNS(ctx)} }()
	go func() { tcpResult <- result{"TCP", checkTCP(ctx)} }()
	go func() { httpResult <- result{"HTTP", checkHTTP(ctx)} }()
	go func() { httpsResult <- result{"HTTPS", checkHTTPS(ctx)} }()

	dns := <-dnsResult
	tcp := <-tcpResult

	if dns.err != nil && tcp.err != nil {
		return errors.New("no network connectivity: DNS and TCP checks both failed")
	}

	http := <-httpResult
	https := <-httpsResult

	if https.err == nil {
		return nil
	}

	if http.err != nil {
		return errors.New("no network connectivity: HTTP check failed")
	}

	return errors.New("no network connectivity: HTTPS check failed")
}

// firstSuccess runs n tasks in parallel via fn(i) and returns nil as soon as
// any task succeeds. Returns errAllFailed if all tasks fail.
func firstSuccess(n int, fn func(i int) error) error {
	succeeded := make(chan struct{}, 1)

	var wg sync.WaitGroup
	var once sync.Once
	for i := range n {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if fn(i) == nil {
				once.Do(func() { succeeded <- struct{}{} })
			}
		}(i)
	}

	wg.Wait()

	select {
	case <-succeeded:
		return nil
	default:
		return errAllFailed
	}
}
