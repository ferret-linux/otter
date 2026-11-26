// Package rootcheck validates privilege escalation programs before root-mode operations.
package rootcheck

import (
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ferret-linux/otter/pkg/ui"
)

// escalatorPreference is the single source of truth for autodetect order.
var escalatorPreference = []string{"sudo", "sudo-rs", "doas", "run0", "pkexec"} //nolint:gochecknoglobals // immutable policy table

// Resolve expands "autodetect" with the shared preference order and checks
// only that the chosen program exists. It deliberately performs no privilege
// probe, so read-only plan mode can use it without prompting or validating.
func Resolve(sudoCommand string) (string, error) {
	if sudoCommand != "" && sudoCommand != "autodetect" {
		return sudoCommand, nil
	}
	for _, name := range escalatorPreference {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}
	return "", errors.New("no privilege escalation program found; install " + strings.Join(escalatorPreference, ", "))
}

type validationResult struct {
	resolved string
	err      error
}

//nolint:gochecknoglobals // singleton: process-wide memoization is the intent
var (
	validateMu      sync.Mutex
	validationCache = make(map[string]validationResult)
)

// Validate ensures that the given privilege escalation program is available
// and, for known programs, that the user can elevate privileges.
// Known programs (sudo, sudo-rs, doas, run0, pkexec) are checked specifically.
// Unknown programs trigger a warning and proceed without validation.
// Each distinct sudoCommand is checked at most once per process; later calls
// with a different escalator get its own cached result instead of a mismatch
// error. Returns the resolved program name (useful for "autodetect").
func Validate(ctx context.Context, sudoCommand string) (string, error) {
	validateMu.Lock()
	defer validateMu.Unlock()

	if result, ok := validationCache[sudoCommand]; ok {
		return result.resolved, result.err
	}

	resolved, err := check(ctx, sudoCommand)
	validationCache[sudoCommand] = validationResult{resolved: resolved, err: err}
	return resolved, err
}

func check(ctx context.Context, sudoCommand string) (string, error) {
	switch filepath.Base(sudoCommand) {
	case "autodetect":
		for _, name := range escalatorPreference {
			if resolved, err := check(ctx, name); err == nil {
				return resolved, nil
			}
		}
		return "", errors.New("no privilege escalation program found; install one of: " + strings.Join(escalatorPreference, ", "))
	case "run0":
		return sudoCommand, checkRun0(ctx, sudoCommand)
	case "pkexec":
		return sudoCommand, checkPkexec(ctx, sudoCommand)
	case "sudo", "sudo-rs":
		return sudoCommand, checkSudo(ctx, sudoCommand)
	case "doas":
		return sudoCommand, checkDoas(ctx, sudoCommand)
	default:
		ui.DefaultLogger.Warn("unknown privilege escalator, skipping validation — ensure it works correctly", "escalator", sudoCommand)
		return sudoCommand, nil
	}
}
