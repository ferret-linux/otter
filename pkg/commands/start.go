package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/ferret-linux/otter/pkg/containermanager"
	"github.com/ferret-linux/otter/pkg/netcheck"
	"github.com/ferret-linux/otter/pkg/ui"
)

type StartOptions struct {
	ContainerNames []string
	All            bool
}

type StartCommand struct {
	containerManager containermanager.ContainerManager
	listCmd          *ListCommand
}

func NewStartCommand(cm containermanager.ContainerManager) *StartCommand {
	return &StartCommand{
		containerManager: cm,
		listCmd:          NewListCommand(cm),
	}
}

func (c *StartCommand) Execute(ctx context.Context, opts *StartOptions) error {
	if err := netcheck.Check(ctx); err != nil {
		ui.DefaultLogger.Warn("%s", err)
		return nil
	}
	if opts.All {
		containers, err := c.listCmd.Execute(ctx)
		if err != nil {
			return fmt.Errorf("failed to list containers: %w", err)
		}
		if len(containers.Containers) == 0 {
			return ErrEmptyContainerList
		}
		for _, container := range containers.Containers {
			if err := c.startOne(ctx, container.Name); err != nil {
				return err
			}
		}
		return nil
	}

	if len(opts.ContainerNames) == 0 {
		return errors.New("please specify a container name with --name/-n")
	}
	for _, name := range opts.ContainerNames {
		if err := c.startOne(ctx, name); err != nil {
			return err
		}
	}
	return nil
}

func (c *StartCommand) startOne(ctx context.Context, containerName string) error {
	if err := c.containerManager.Start(ctx, containerName); err != nil {
		return fmt.Errorf("failed to start container '%s': %w", containerName, err)
	}
	return nil
}
