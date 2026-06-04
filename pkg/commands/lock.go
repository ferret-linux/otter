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

type LockOptions struct {
	ContainerNames []string
	All            bool
	Verbose        bool
	DryRun         bool
}

type LockCommand struct {
	containerManager containermanager.ContainerManager
	listCmd          *ListCommand
}

func NewLockCommand(cfg *config.Values, cm containermanager.ContainerManager) *LockCommand {
	return &LockCommand{
		containerManager: cm,
		listCmd:          NewListCommand(cfg, cm),
	}
}

func (c *LockCommand) Execute(ctx context.Context, opts LockOptions) error {
	var containerNames []string
	switch {
	case opts.All:
		listResult, err := c.listCmd.Execute(ctx)
		if err != nil {
			return fmt.Errorf("failed to list containers: %w", err)
		}
		if len(listResult.Containers) == 0 {
			return ErrEmptyContainerList
		}
		for _, container := range listResult.Containers {
			containerNames = append(containerNames, container.Name)
		}
	case len(opts.ContainerNames) > 0:
		containerNames = opts.ContainerNames
	default:
		return fmt.Errorf("please specify a container name with --name/-n")
	}

	var lastErr error
	for _, name := range containerNames {
		if err := c.lockOne(ctx, name, opts); err != nil {
			ui.DefaultLogger.Error("failed to lock '%s': %s", name, err)
			lastErr = err
		}
	}
	return lastErr
}

func (c *LockCommand) lockOne(ctx context.Context, name string, opts LockOptions) error {
	if !c.containerManager.Exists(ctx, name) {
		return fmt.Errorf("container '%s' not found", name)
	}

	if isLocked(ctx, c.containerManager, name) {
		return fmt.Errorf("'%s' %w", name, ErrAlreadyLocked)
	}

	if opts.Verbose {
		ui.DefaultLogger.Info("writing lock file to '%s' at %s", name, lockFilePath)
	}

	if opts.DryRun {
		ui.DefaultLogger.Info("would write lock file to '%s' at %s", name, lockFilePath)
		return nil
	}

	f, err := os.CreateTemp("", "otter-lock-*")
	if err != nil {
		return fmt.Errorf("failed to create lock file: %w", err)
	}
	f.Close()
	defer os.Remove(f.Name())

	if err := c.containerManager.WriteToContainer(ctx, name, f.Name(), lockFilePath); err != nil {
		return fmt.Errorf("failed to write lock file into '%s': %w", name, err)
	}

	ui.DefaultLogger.Ok("locked '%s'", name)
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
