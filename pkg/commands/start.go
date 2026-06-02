package commands

import (
	"context"
	"fmt"

	"github.com/ferret-linux/otter/pkg/config"
	"github.com/ferret-linux/otter/pkg/containermanager"
	"github.com/ferret-linux/otter/pkg/ui"
)

type StartOptions struct {
	ContainerName string
	All           bool
	DryRun        bool
	Verbose       bool
}

type StartCommand struct {
	cfg              *config.Values
	containerManager containermanager.ContainerManager
	listCmd          *ListCommand
}

func NewStartCommand(cfg *config.Values, cm containermanager.ContainerManager) *StartCommand {
	return &StartCommand{
		cfg:              cfg,
		containerManager: cm,
		listCmd:          NewListCommand(cfg, cm),
	}
}

func (c *StartCommand) Execute(ctx context.Context, opts *StartOptions) error {
	if opts.All {
		containers, err := c.listCmd.Execute(ctx)
		if err != nil {
			return fmt.Errorf("failed to list containers: %w", err)
		}
		if len(containers.Containers) == 0 {
			return ErrEmptyContainerList
		}
		for _, container := range containers.Containers {
			if err := c.startOne(ctx, container.Name, opts); err != nil {
				return err
			}
		}
		return nil
	}

	containerName := opts.ContainerName
	if containerName == "" {
		containerName = c.cfg.DefaultContainerName
	}
	return c.startOne(ctx, containerName, opts)
}

func (c *StartCommand) startOne(ctx context.Context, containerName string, opts *StartOptions) error {
	if opts.Verbose {
		ui.DefaultLogger.Info("starting: %s", containerName)
	}

	if err := c.containerManager.Start(ctx, containerName, opts.DryRun); err != nil {
		return err
	}

	return nil
}
