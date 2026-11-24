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

// fatalSetupErrorPrefixes are otter-init's own fatal messages. Init systems
// and package managers can print their own "Error:" lines while the container
// is still booting (OpenRC, for example, reports a real/virtual loopback
// service collision but completes boot normally). Only otter's messages mean
// setup has actually failed.
var fatalSetupErrorPrefixes = []string{ //nolint:gochecknoglobals // immutable lookup table
	"Error: An error occurred",
	"Error: container is borked",
	"Error: Invalid arg",
	"Error: Missing value",
	"Error: Empty value",
	"Error: Invalid flag",
	"Error: could not",
	"Error: unsupported",
	"Error: no network connectivity",
	"Error: Failed to modify user login name",
}

func isFatalSetupError(line string) bool {
	for _, prefix := range fatalSetupErrorPrefixes {
		if strings.HasPrefix(line, prefix) {
			return true
		}
	}
	return false
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
