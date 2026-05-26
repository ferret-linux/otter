package commands

import (
	"context"

	"github.com/ferret-linux/otter/pkg/config"
	"github.com/ferret-linux/otter/pkg/containermanager"
	"github.com/ferret-linux/otter/pkg/ui"
)

type StartOptions struct {
	ContainerName string
	DryRun        bool
	Verbose       bool
}

type StartCommand struct {
	cfg              *config.Values
	containerManager containermanager.ContainerManager
}

func NewStartCommand(cfg *config.Values, cm containermanager.ContainerManager) *StartCommand {
	return &StartCommand{
		cfg:              cfg,
		containerManager: cm,
	}
}

func (c *StartCommand) Execute(ctx context.Context, opts *StartOptions) error {
	containerName := opts.ContainerName
	if containerName == "" {
		containerName = c.cfg.DefaultContainerName
	}

	if opts.Verbose {
		ui.DefaultLogger.Info("starting: %s", containerName)
	}

	if err := c.containerManager.Start(ctx, containerName, opts.DryRun); err != nil {
		return err
	}

	return nil
}
