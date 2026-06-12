// Package rootcheck validates privilege escalation programs before root-mode operations.
package rootcheck

import (
	"context"
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
)

// Validate ensures that the given privilege escalation program is available
// and, for known programs, that the user can elevate privileges.
// Known programs (run0, sudo, sudo-rs, doas) are checked specifically.
// Unknown programs trigger a warning and proceed without validation.
// The check runs at most once per process; subsequent calls return the cached result.
func Validate(ctx context.Context, sudoCommand string) error {
	if cachedSudoCommand != "" && cachedSudoCommand != sudoCommand {
		return fmt.Errorf("sudoCommand mismatch: already validated with %q, got %q", cachedSudoCommand, sudoCommand)
	}
	validateOnce.Do(func() {
		cachedSudoCommand = sudoCommand
		errValidate = check(ctx, sudoCommand)
	})
	return errValidate
}

func check(ctx context.Context, sudoCommand string) error {
	switch filepath.Base(sudoCommand) {
	case "run0":
		return checkRun0(sudoCommand)
	case "sudo", "sudo-rs":
		return checkSudo(ctx, sudoCommand)
	case "doas":
		return checkDoas(ctx, sudoCommand)
	default:
		ui.DefaultLogger.Warn("unknown privilege escalator %q, skipping validation — ensure it works correctly", sudoCommand)
		return nil
	}
}
