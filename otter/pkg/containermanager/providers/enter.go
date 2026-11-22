package providers

import (
	"context"
	"errors"
	"fmt"

	"github.com/ferret-linux/otter/pkg/containermanager"
)

// resolveInspectResult reuses an inspection already made by the caller.
// Direct ContainerManager.Enter callers that did not supply one fall back to
// inspecting through fn, so this remains a single inspect in every path.
func resolveInspectResult(
	ctx context.Context,
	containerName string,
	supplied *containermanager.InspectResult,
	fn func(context.Context, string) (*containermanager.InspectResult, error),
) (*containermanager.InspectResult, error) {
	if supplied != nil {
		return supplied, nil
	}
	result, err := fn(ctx, containerName)
	if err != nil {
		if errors.Is(err, containermanager.ErrContainerNotFound) {
			return nil, fmt.Errorf("container '%s' not found", containerName)
		}
		return nil, fmt.Errorf("failed to inspect container '%s': %w", containerName, err)
	}
	return result, nil
}
