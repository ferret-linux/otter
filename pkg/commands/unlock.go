package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/ferret-linux/otter/pkg/config"
	"github.com/ferret-linux/otter/pkg/containermanager"
	"github.com/ferret-linux/otter/pkg/ui"
)

var ErrNotLocked = errors.New("container is not locked")

type UnlockOptions struct {
	ContainerName string
}

type UnlockCommand struct {
	cfg              *config.Values
	containerManager containermanager.ContainerManager
}

func NewUnlockCommand(cfg *config.Values, cm containermanager.ContainerManager) *UnlockCommand {
	return &UnlockCommand{
		cfg:              cfg,
		containerManager: cm,
	}
}

func (c *UnlockCommand) Execute(ctx context.Context, opts UnlockOptions) error {
	if !c.containerManager.Exists(ctx, opts.ContainerName) {
		return fmt.Errorf("%w: '%s'", ErrContainerNotFound, opts.ContainerName)
	}

	if !isLocked(ctx, c.containerManager, opts.ContainerName) {
		return fmt.Errorf("'%s' %w", opts.ContainerName, ErrNotLocked)
	}

	if err := c.containerManager.DeleteFromContainer(ctx, opts.ContainerName, lockFilePath); err != nil {
		return fmt.Errorf("failed to remove lock file from '%s': %w", opts.ContainerName, err)
	}

	ui.DefaultLogger.Ok("unlocked '%s'", opts.ContainerName)
	return nil
}
