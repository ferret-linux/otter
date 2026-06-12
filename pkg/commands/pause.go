package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/ferret-linux/otter/pkg/containermanager"
	"github.com/ferret-linux/otter/pkg/ui"
)

type PauseOptions struct {
	ContainerNames []string
	All            bool
}

type PauseCommand struct {
	containerManager containermanager.ContainerManager
	listCmd          *ListCommand
}

func NewPauseCommand(cm containermanager.ContainerManager) *PauseCommand {
	return &PauseCommand{
		containerManager: cm,
		listCmd:          NewListCommand(cm),
	}
}

func (c *PauseCommand) Execute(ctx context.Context, opts *PauseOptions) error {
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
			if !container.IsRunning() {
				ui.DefaultLogger.Warn("'%s' is not running, skipping", container.Name)
				continue
			}
			containerNames = append(containerNames, container.Name)
		}
		if len(containerNames) == 0 {
			return ErrEmptyContainerList
		}
	case len(opts.ContainerNames) > 0:
		containerNames = opts.ContainerNames
	default:
		return errors.New("please specify a container name or use --all")
	}

	for _, name := range containerNames {
		if err := c.containerManager.Pause(ctx, name); err != nil {
			return fmt.Errorf("failed to pause container '%s': %w", name, err)
		}
		ui.DefaultLogger.Ok("paused '%s'", name)
	}

	return nil
}
