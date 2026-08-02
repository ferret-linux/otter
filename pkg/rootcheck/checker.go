// Package rootcheck validates privilege escalation programs before root-mode operations.
package rootcheck

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/ferret-linux/otter/pkg/ui"
)

//nolint:gochecknoglobals // singleton: process-wide memoization is the intent
var (
	validateOnce      sync.Once
	errValidate       error
	cachedSudoCommand string
	resolvedCommand   string
)

// Validate ensures that the given privilege escalation program is available
// and, for known programs, that the user can elevate privileges.
// Known programs (sudo, sudo-rs, doas, run0) are checked specifically.
// Unknown programs trigger a warning and proceed without validation.
// The check runs at most once per process; subsequent calls return the cached result.
// Validate ensures that the given privilege escalation program is available
// and, for known programs, that the user can elevate privileges.
// Known programs (sudo, sudo-rs, doas, run0) are checked specifically.
// Unknown programs trigger a warning and proceed without validation.
// The check runs at most once per process; subsequent calls return the cached result.
// Returns the resolved program name (useful when sudoCommand is "autodetect").
func Validate(ctx context.Context, sudoCommand string) (string, error) {
	if cachedSudoCommand != "" && cachedSudoCommand != sudoCommand {
		return "", fmt.Errorf("sudoCommand mismatch: already validated with %q, got %q", cachedSudoCommand, sudoCommand)
	}
	validateOnce.Do(func() {
		cachedSudoCommand = sudoCommand
		resolvedCommand, errValidate = check(ctx, sudoCommand)
	})
	return resolvedCommand, errValidate
}

func check(ctx context.Context, sudoCommand string) (string, error) {
	switch filepath.Base(sudoCommand) {
	case "autodetect":
		for _, name := range []string{"sudo", "sudo-rs", "doas", "run0"} {
			if resolved, err := check(ctx, name); err == nil {
				return resolved, nil
			}
		}
		return "", errors.New("no privilege escalation program found; install one of: sudo, sudo-rs, doas, run0")
	case "run0":
		return sudoCommand, checkRun0(sudoCommand)
	case "sudo", "sudo-rs":
		return sudoCommand, checkSudo(ctx, sudoCommand)
	case "doas":
		return sudoCommand, checkDoas(ctx, sudoCommand)
	default:
		ui.DefaultLogger.Warn("unknown privilege escalator %q, skipping validation — ensure it works correctly", sudoCommand)
		return sudoCommand, nil
	}
}
