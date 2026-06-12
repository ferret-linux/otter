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

// autodetectOrder is the order in which privilege escalators are tried during autodetection.
//
//nolint:gochecknoglobals // package-level detection order is effectively a constant
var autodetectOrder = []string{"pkexec", "run0", "sudo", "doas", "su"}

// Validate ensures that the given privilege escalation program is available
// and, for known programs, that the user can elevate privileges.
// If sudoCommand is "autodetect" or empty, the first available escalator
// from the autodetect order is used.
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
	if sudoCommand == "autodetect" || sudoCommand == "" {
		return autodetect(ctx)
	}
	name := filepath.Base(sudoCommand)
	switch name {
	case "pkexec":
		return checkPkexec(sudoCommand)
	case "run0":
		return checkRun0(sudoCommand)
	case "sudo", "sudo-rs":
		return checkSudo(ctx, sudoCommand)
	case "doas":
		return checkDoas(ctx, sudoCommand)
	case "su":
		return checkSu(sudoCommand)
	default:
		ui.DefaultLogger.Warn("unknown privilege escalator %q, skipping validation — ensure it works correctly", sudoCommand)
		return nil
	}
}

func autodetect(ctx context.Context) error {
	for _, name := range autodetectOrder {
		if err := check(ctx, name); err == nil {
			return nil
		}
	}
	return fmt.Errorf("no supported privilege escalator found (tried: %v)", autodetectOrder)
}
