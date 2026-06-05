package commands

import (
	"context"
	"fmt"

	"github.com/ferret-linux/otter/pkg/containermanager"
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

	if len(opts.ContainerNames) == 0 {
		return fmt.Errorf("please specify a container name with --name/-n")
	}
	for _, name := range opts.ContainerNames {
		if err := c.startOne(ctx, name, opts); err != nil {
			return err
		}
	}
	return nil
}

func (c *StartCommand) startOne(ctx context.Context, containerName string, opts *StartOptions) error {
	if err := c.containerManager.Start(ctx, containerName); err != nil {
		return err
	}

	return nil
}
