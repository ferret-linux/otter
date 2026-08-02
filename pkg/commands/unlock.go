package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/ferret-linux/otter/pkg/containermanager"
	"github.com/ferret-linux/otter/pkg/ui"
)

var ErrNotLocked = errors.New("container is not locked")

type UnlockOptions struct {
	ContainerNames []string
	All            bool
}

type UnlockCommand struct {
	containerManager containermanager.ContainerManager
	listCmd          *ListCommand
}

func NewUnlockCommand(cm containermanager.ContainerManager) *UnlockCommand {
	return &UnlockCommand{
		containerManager: cm,
		listCmd:          NewListCommand(cm),
	}
}

func (c *UnlockCommand) Execute(ctx context.Context, opts UnlockOptions) error {
	var containerNames []string
	switch {
	case opts.All:
		listResult, err := c.listCmd.Execute(ctx, ListOptions{})
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
		return errors.New("please specify a container name or use --all")
	}

	var lastErr error
	for _, name := range containerNames {
		if err := c.unlockOne(ctx, name); err != nil {
			if errors.Is(err, ErrNotLocked) {
				ui.DefaultLogger.Warn("'%s' is already unlocked, skipping", name)
				continue
			}
			ui.DefaultLogger.Error("failed to unlock '%s': %s", name, err)
			lastErr = err
		}
	}
	return lastErr
}

func (c *UnlockCommand) unlockOne(ctx context.Context, name string) error {
	if !c.containerManager.Exists(ctx, name) {
		return fmt.Errorf("container '%s' not found", name)
	}

	if !isLocked(ctx, c.containerManager, name) {
		return fmt.Errorf("'%s' %w", name, ErrNotLocked)
	}

	if err := c.containerManager.DeleteFromContainer(ctx, name, lockFilePath); err != nil {
		return fmt.Errorf("failed to remove lock file from '%s': %w", name, err)
	}

	ui.DefaultLogger.Ok("unlocked '%s'", name)
	return nil
}
