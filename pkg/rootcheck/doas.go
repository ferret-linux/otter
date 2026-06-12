package rootcheck

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

func checkDoas(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, name, "-v")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to validate %q: %w", name, err)
	}
	return nil
}
