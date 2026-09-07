package providers

import (
	"errors"
	"os/exec"

	"github.com/ferret-linux/otter/pkg/containermanager"
)

// ErrNoContainerManager is returned when no supported container runtime is found.
var ErrNoContainerManager = errors.New("no container manager found")

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
