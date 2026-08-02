package commands

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/ferret-linux/otter/pkg/containermanager"
	"github.com/ferret-linux/otter/pkg/ui"
)

const lockFilePath = "/usr/lib/otter/container.lock"

var ErrAlreadyLocked = errors.New("container is already locked")

type LockOptions struct {
	ContainerNames []string
	All            bool
}

type LockCommand struct {
	containerManager containermanager.ContainerManager
	listCmd          *ListCommand
}

func NewLockCommand(cm containermanager.ContainerManager) *LockCommand {
	return &LockCommand{
		containerManager: cm,
		listCmd:          NewListCommand(cm),
	}
}

func (c *LockCommand) Execute(ctx context.Context, opts LockOptions) error {
	var containerNames []string
	switch {
	case opts.All:
		listResult, err := c.listCmd.Execute(ctx, ListOptions{})
		if err != nil {
			return fmt.Errorf("failed to list containers: %w", err)
		}
		if len(listResult.Containers) == 0 {
			return ErrNoContainersFound
		}
		for _, container := range listResult.Containers {
			containerNames = append(containerNames, container.Name)
		}
	case len(opts.ContainerNames) > 0:
		containerNames = opts.ContainerNames
	default:
		return errors.New("please specify a container name or use --all")
	}

	var lastErr error
	for _, name := range containerNames {
		if err := c.lockOne(ctx, name); err != nil {
			if errors.Is(err, ErrAlreadyLocked) {
				ui.DefaultLogger.Warn("'%s' is already locked, skipping", name)
				continue
			}
			ui.DefaultLogger.Error("failed to lock '%s': %s", name, err)
			lastErr = err
		}
	}
	return lastErr
}

func (c *LockCommand) lockOne(ctx context.Context, name string) error {
	if !c.containerManager.Exists(ctx, name) {
		return fmt.Errorf("container '%s' not found", name)
	}

	if isLocked(ctx, c.containerManager, name) {
		return fmt.Errorf("'%s' %w", name, ErrAlreadyLocked)
	}

	f, err := os.CreateTemp("", "otter-lock-*")
	if err != nil {
		return fmt.Errorf("failed to create lock file: %w", err)
	}
	defer os.Remove(f.Name())
	if err := f.Close(); err != nil {
		return fmt.Errorf("failed to close temp lock file: %w", err)
	}

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
	defer os.Remove(tmp.Name())
	if err := tmp.Close(); err != nil {
		return false
	}

	return cm.CopyFromContainer(ctx, containerName, lockFilePath, tmp.Name()) == nil
}
