//nolint:goconst // CLI flag strings are intentionally repeated per-provider; they may diverge independently
package providers

import "context"

type containerRunner func(ctx context.Context, args []string, opts runOptions) (string, error)

// hasOtterManagedMarker reports whether the container carries otter's marker
// directory. Setup watchers use this to avoid replayed log lines from a
// previous boot being mistaken for completion of the current one.
func hasOtterManagedMarker(ctx context.Context, run containerRunner, containerName string) bool {
	_, err := run(ctx, []string{"exec", containerName, "test", "-d", "/usr/lib/otter"}, runOptions{})
	return err == nil
}
