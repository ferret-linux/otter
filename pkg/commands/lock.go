package commands

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/ferret-linux/otter/pkg/config"
	"github.com/ferret-linux/otter/pkg/containermanager"
	"github.com/ferret-linux/otter/pkg/ui"
)

const lockFilePath = "/usr/lib/otter/container.lock"

var ErrAlreadyLocked = errors.New("container is already locked")
var ErrContainerNotFound = errors.New("container not found")

type LockOptions struct {
	ContainerName string
	Verbose       bool
	DryRun        bool
}

type LockCommand struct {
	cfg              *config.Values
	containerManager containermanager.ContainerManager
}

func NewLockCommand(cfg *config.Values, cm containermanager.ContainerManager) *LockCommand {
	return &LockCommand{
		cfg:              cfg,
		containerManager: cm,
	}
}

func (c *LockCommand) Execute(ctx context.Context, opts LockOptions) error {
	if !c.containerManager.Exists(ctx, opts.ContainerName) {
		return fmt.Errorf("%w: '%s'", ErrContainerNotFound, opts.ContainerName)
	}

	if isLocked(ctx, c.containerManager, opts.ContainerName) {
		return fmt.Errorf("'%s' %w", opts.ContainerName, ErrAlreadyLocked)
	}

	if opts.Verbose {
		ui.DefaultLogger.Info("writing lock file to '%s' at %s", opts.ContainerName, lockFilePath)
	}

	if opts.DryRun {
		ui.DefaultLogger.Info("would write lock file to '%s' at %s", opts.ContainerName, lockFilePath)
		return nil
	}

	f, err := os.CreateTemp("", "otter-lock-*")
	if err != nil {
		return fmt.Errorf("failed to create lock file: %w", err)
	}
	f.Close()
	defer os.Remove(f.Name())

	if err := c.containerManager.WriteToContainer(ctx, opts.ContainerName, f.Name(), lockFilePath); err != nil {
		return fmt.Errorf("failed to write lock file into '%s': %w", opts.ContainerName, err)
	}

	ui.DefaultLogger.Ok("locked '%s'", opts.ContainerName)
	return nil
}

// isLocked checks whether a container has a lock file present.
// Shared by lock, unlock, remove, and upgrade.
func isLocked(ctx context.Context, cm containermanager.ContainerManager, containerName string) bool {
	tmp, err := os.CreateTemp("", "otter-lockcheck-*")
	if err != nil {
		return false
	}
	tmp.Close()
	defer os.Remove(tmp.Name())

	return cm.CopyFromContainer(ctx, containerName, lockFilePath, tmp.Name()) == nil
}
