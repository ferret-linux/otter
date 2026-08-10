package containermanager

import (
	"context"
	"io"
)

const (
	RunningStatus = "running"
	PausedStatus  = "paused"
)

//nolint:revive // ContainerManagerType is intentionally named for clarity despite the stutter
type ContainerManagerType string

// PullOutput is where PullImage streams its pull output when the caller
// wants it rendered live. A nil PullOutput means "no live rendering" —
// today's buffered, silent-until-error behavior.
type PullOutput interface {
	io.Writer
}

type ContainerManager interface {
	Name() string
	// CloneAsRoot returns a copy of the manager configured to run in root
	// mode. The original instance is not modified.
	CloneAsRoot() ContainerManager
	Enter(ctx context.Context, options EnterOptions) error
	ListContainers(ctx context.Context) ([]Container, error)
	Create(ctx context.Context, opts CreateOptions) error
	Remove(ctx context.Context, containerName string, opts RmOptions) error
	Exists(ctx context.Context, containerName string) bool
	ImageExists(ctx context.Context, imageName string) bool
	// ImageLabel returns the value of the given label on a locally present
	// image. It returns ("", false) if the image doesn't exist locally, the
	// engine call fails, or the label isn't set — no error, matching the
	// ImageExists precedent.
	ImageLabel(ctx context.Context, imageName, key string) (string, bool)
	Start(ctx context.Context, containerName string) error
	Stop(ctx context.Context, containerNames []string, force bool) error
	InspectContainer(ctx context.Context, containerName string) (*InspectResult, error)
	PullImage(ctx context.Context, imageName string, platform string, out PullOutput) error
	RemoveImage(ctx context.Context, imageName string, force bool) error
	Commit(ctx context.Context, containerID string, imageTag string) error
	// CopyFromContainer copies a file from the container filesystem to the host.
	CopyFromContainer(ctx context.Context, containerName string, srcPath string, destPath string) error
	// WriteToContainer copies a file from the host into the container filesystem.
	WriteToContainer(ctx context.Context, containerName string, srcPath string, destPath string) error
	// DeleteFromContainer removes a file or directory from inside the container filesystem.
	DeleteFromContainer(ctx context.Context, containerName string, filePath string) error
	// IsSetupDone returns true if the container has completed its initial setup.
	IsSetupDone(ctx context.Context, containerName string) bool
	// IsUpgrading returns true if the container is currently running an
	// 'otter upgrade' (otter-init --upgrade), based on the presence of the
	// container.upgrading marker file otter-init maintains for the
	// duration of that run.
	IsUpgrading(ctx context.Context, containerName string) bool
	// Journal streams the logs of a container to stdout.
	Journal(ctx context.Context, containerName string, opts JournalOptions) error
	// Pause freezes a running container without stopping it.
	Pause(ctx context.Context, containerName string) error
	// Unpause resumes a paused container.
	Unpause(ctx context.Context, containerName string) error
}
