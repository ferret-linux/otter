package netcheck

import "errors"

// Check runs all connectivity checks and returns an error if any fail.
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
	return nil
}
