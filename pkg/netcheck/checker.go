package netcheck

import (
	"errors"
	"sync"
)

var errAllFailed = errors.New("all targets failed")

// Check runs all connectivity checks and returns an error if any fail.
// Each check passes if at least one of its targets succeeds.
// All four checks run in parallel.
func Check() error {
	type result struct {
		name string
		err  error
	}

	checks := []struct {
		name string
		fn   func() error
	}{
		{"DNS", checkDNS},
		{"TCP", checkTCP},
		{"HTTP", checkHTTP},
		{"HTTPS", checkHTTPS},
	}

	results := make(chan result, len(checks))

	var wg sync.WaitGroup
	for _, c := range checks {
		wg.Add(1)
		go func(name string, fn func() error) {
			defer wg.Done()
			results <- result{name: name, err: fn()}
		}(c.name, c.fn)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	for r := range results {
		if r.err != nil {
			return errors.New("no network connectivity: " + r.name + " check failed")
		}
	}
	return nil
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
