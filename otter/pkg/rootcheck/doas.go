package rootcheck

import (
	"context"
	"fmt"
	"os/exec"
)

// checkDoas confirms doas is available. Unlike sudo, doas has no verify-only
// invocation (e.g. "doas -v") that can validate credentials without either
// requiring a real command to run or erroring out — see the doas(1) synopsis:
// "doas [-Lns] [-C config] [-u user] command [args]". So this only checks
// that the binary exists, matching the same approach used by checkRun0.
func checkDoas(_ context.Context, name string) error {
	if _, err := exec.LookPath(name); err != nil {
		return fmt.Errorf("failed to validate %q: %w", name, err)
	}
	return nil
}
