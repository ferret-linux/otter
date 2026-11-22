package providers

import (
	"fmt"
	"strings"

	"github.com/ferret-linux/otter/pkg/containermanager"
)

// Runtime phrases that mean "container does not exist". Everything else is
// kept as a real error so callers do not hide daemon or auth failures behind
// a misleading "not found"/"not running" message.
var notFoundMarkers = []string{ //nolint:gochecknoglobals // immutable lookup table
	"no such container",
	"no container with name or id",
	"no such object",
	"container not found",
}

func inspectError(containerName string, err error) error {
	lower := strings.ToLower(err.Error())
	for _, marker := range notFoundMarkers {
		if strings.Contains(lower, marker) {
			return fmt.Errorf("container %q: %w", containerName, containermanager.ErrContainerNotFound)
		}
	}
	return fmt.Errorf("failed to inspect container %q: %w", containerName, err)
}
