package providers

import (
	"context"
	"fmt"
	"strings"
)

// inspectStatus runs a status-only container inspect. Errors go through
// inspectError so the wording matches a full InspectContainer call.
func inspectStatus(
	ctx context.Context,
	run func(context.Context, []string, runOptions) (string, error),
	args []string,
	containerName string,
) (string, error) {
	out, err := run(ctx, args, runOptions{})
	if err != nil {
		return "", inspectError(containerName, err)
	}
	status := strings.TrimSpace(out)
	if status == "" {
		return "", fmt.Errorf("failed to inspect container %q: empty status", containerName)
	}
	return status, nil
}
