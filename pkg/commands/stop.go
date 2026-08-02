package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/ferret-linux/otter/pkg/containermanager"
)

type StopCommand struct {
	containerManager containermanager.ContainerManager
	listCmd          *ListCommand
}

type StopOptions struct {
	ContainerNames []string
	All            bool
	Force          bool
}

var ErrEmptyContainerList = errors.New("cannot find containers to stop")

func NewStopCommand(cm containermanager.ContainerManager) *StopCommand {
	return &StopCommand{
		containerManager: cm,
		listCmd:          NewListCommand(cm),
	}
}

func (c *StopCommand) Execute(ctx context.Context, opts *StopOptions) error {
	var containerNames []string
	switch {
	case opts.All:
		containers, err := c.listCmd.Execute(ctx, ListOptions{})
		if err != nil {
			return fmt.Errorf("failed to list containers: %w", err)
		}
		if len(containers.Containers) == 0 {
			return ErrEmptyContainerList
		}
		containerNames = make([]string, 0, len(containers.Containers))
		for _, container := range containers.Containers {
			containerNames = append(containerNames, container.Name)
		}
	case len(opts.ContainerNames) > 0:
		containerNames = opts.ContainerNames
	default:
		return errors.New("please specify a container name or use --all")
	}

	for _, name := range containerNames {
		if _, err := c.containerManager.InspectContainer(ctx, name); err != nil {
			return fmt.Errorf("container '%s' not found", name)
		}
	}

	if err := c.containerManager.Stop(ctx, containerNames, opts.Force); err != nil {
		return fmt.Errorf("failed to stop containers: %w", err)
	}

	return nil
}
