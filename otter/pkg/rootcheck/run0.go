package rootcheck

import (
	"context"
	"fmt"
	"os/exec"
)

// checkRun0 confirms run0 is available. Like pkexec, run0 has no safe
// validate-only invocation, and actually running it would trigger a real
// authentication attempt. So this only checks that the binary exists.
func checkRun0(_ context.Context, name string) error {
	if _, err := exec.LookPath(name); err != nil {
		return fmt.Errorf("failed to validate %q: %w", name, err)
	}
	return nil
}
