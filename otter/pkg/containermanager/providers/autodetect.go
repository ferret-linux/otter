package providers

import (
	"errors"
	"fmt"
	"os/exec"

	"github.com/ferret-linux/otter/pkg/containermanager"
)

// ErrNoContainerManager is returned when no supported container runtime is found.
var ErrNoContainerManager = errors.New("no container manager found")

// NewByType builds a container manager for an explicit runtime type. Unlike
// NewAutoDetect it never falls back to another runtime, so commands that work
// with exports recorded by a specific manager don't silently query the wrong
// one. The sudo command must already be resolved by the caller when root is
// true.
func NewByType(managerType string, root bool, sudoCommand string) (containermanager.ContainerManager, error) {
	switch managerType {
	case "podman":
		return NewPodman(root, sudoCommand), nil
	case "docker":
		return NewDocker(root, sudoCommand), nil
	case "nerdctl":
		return NewNerdctl(root, sudoCommand), nil
	default:
		return nil, fmt.Errorf("unsupported container manager %q", managerType)
	}
}

// NewAutoDetect returns a ContainerManager for the first available container runtime.
// Priority order: podman > nerdctl > docker.
func NewAutoDetect(root bool, sudoCommand string) (containermanager.ContainerManager, error) {
	if _, err := exec.LookPath("podman"); err == nil {
		return NewPodman(root, sudoCommand), nil
	}
	if _, err := exec.LookPath("nerdctl"); err == nil {
		return NewNerdctl(root, sudoCommand), nil
	}
	if _, err := exec.LookPath("docker"); err == nil {
		return NewDocker(root, sudoCommand), nil
	}
	return nil, ErrNoContainerManager
}
