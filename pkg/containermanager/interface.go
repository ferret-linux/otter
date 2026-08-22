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

// PullOutputSizer is an optional extension of PullOutput. Implementations
// that render into a resizable region (e.g. a live-resized box) implement
// it so PullImage can give the pulling process a pseudo-terminal sized to
// match, and keep it in sync as the region is resized. Implementations
// that don't need this (e.g. tests, or a plain io.Writer) simply don't
// implement it — PullImage falls back to no live sizing.
type PullOutputSizer interface {
	// Size returns the current size, in cells, of the region PullOutput
	// renders into.
	Size() (rows, cols int)
	// OnResize registers a callback invoked with the new size every time
	// the region is resized, for as long as the PullOutput is active.
	// Replaces any previously registered callback.
	OnResize(func(rows, cols int))
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
	// ImageID returns the local image ID for the given image name/ref. It
	// returns ("", false) if the image doesn't exist locally or the engine
	// call fails — no error, matching the ImageExists/ImageLabel precedent.
	ImageID(ctx context.Context, imageName string) (string, bool)
	// ContainerImageID returns the image ID a container was actually created
	// from — its frozen binding, not the (possibly since-moved) tag it was
	// created with. Unlike resolving a tag through ImageID, this value does
	// not change if the tag is later re-pulled to a different build. Returns
	// ("", false) if the container doesn't exist or the engine call fails.
	ContainerImageID(ctx context.Context, containerName string) (string, bool)
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
	// Journal streams the logs of a container to stdout.
	Journal(ctx context.Context, containerName string, opts JournalOptions) error
	// Pause freezes a running container without stopping it.
	Pause(ctx context.Context, containerName string) error
	// Unpause resumes a paused container.
	Unpause(ctx context.Context, containerName string) error
}
