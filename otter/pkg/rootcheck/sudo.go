package rootcheck

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
)

// checkSudo verifies that sudo is available without attempting to validate
// cached credentials. `sudo -v` needs a TTY or an existing timestamp and can
// fail in non-interactive sessions even when passwordless sudo works; `sudo
// -V` only prints the version, so it is safe in scripts. As with doas, run0
// and pkexec, this checks availability rather than actual elevation rights.
func checkSudo(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, name, "-V")
	cmd.Stdout = io.Discard
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to validate %q: %w", name, err)
	}
	return nil
}
