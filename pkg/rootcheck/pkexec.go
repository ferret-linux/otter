package rootcheck

import (
	"context"
	"fmt"
	"os/exec"
)

// checkPkexec confirms pkexec is available. Like run0, pkexec has no
// safe validate-only invocation (its only flags are --version,
// --disable-internal-agent, --user, and --help), and actually running it
// would trigger a real polkit authentication attempt, which depends on an
// authentication agent being registered for the session — unreliable in
// headless/non-desktop contexts. So this only checks that the binary exists.
func checkPkexec(_ context.Context, name string) error {
	if _, err := exec.LookPath(name); err != nil {
		return fmt.Errorf("failed to validate %q: %w", name, err)
	}
	return nil
}
