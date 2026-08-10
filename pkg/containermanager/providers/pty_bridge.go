package providers

import (
	"context"
	"fmt"
	"io"
	"os/exec"

	"github.com/creack/pty"
)

// runInteractivePTY runs cmd attached to a locally-allocated pseudo-terminal
// instead of wiring runOptions.Stdin/Stdout/Stderr directly to the child's
// stdio. This matters specifically for terminal resize: podman/docker/
// nerdctl's own exec clients only forward window-size changes to the
// container when they themselves believe they're attached to a real
// terminal (they watch for SIGWINCH on their own stdin/stdout). Giving the
// client a real local pty — rather than a plain pipe — makes that detection
// succeed, so its existing resize-forwarding logic activates with no
// otter-side protocol needed; we just need to trigger SIGWINCH on the local
// pty, which Setsize below does.
//
// Bytes are pumped between the pty and opts.Stdin/Stdout/Stderr. Sizes read
// from opts.Resize are applied to the pty via pty.Setsize.
func runInteractivePTY(ctx context.Context, cmd *exec.Cmd, opts runOptions) error {
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return fmt.Errorf("failed to allocate local pty: %w", err)
	}
	defer ptmx.Close() //nolint:errcheck // best-effort cleanup

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case size, ok := <-opts.Resize:
				if !ok {
					return
				}
				//nolint:errcheck // best-effort; a failed resize isn't fatal to the session
				pty.Setsize(ptmx, &pty.Winsize{Rows: size.Rows, Cols: size.Cols})
			}
		}
	}()

	if opts.Stdin != nil {
		go func() { _, _ = io.Copy(ptmx, opts.Stdin) }()
	}
	if opts.Stdout != nil {
		go func() { _, _ = io.Copy(opts.Stdout, ptmx) }()
	}

	return cmd.Wait()
}
