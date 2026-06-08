package netcheck

import "errors"

var errAllFailed = errors.New("all targets failed")

// Check runs all connectivity checks and returns an error if any fail.
// Each check passes if at least one of its targets succeeds.
func Check() error {
	if err := checkDNS(); err != nil {
		return errors.New("no network connectivity: DNS resolution failed")
	}
	if err := checkTCP(); err != nil {
		return errors.New("no network connectivity: TCP connect failed")
	}
	if err := checkHTTP(); err != nil {
		return errors.New("no network connectivity: HTTP check failed")
	}
	if err := checkHTTPS(); err != nil {
		return errors.New("no network connectivity: HTTPS check failed")
	}
	return nil
}
