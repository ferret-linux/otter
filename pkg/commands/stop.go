package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/ferret-linux/otter/pkg/containermanager"
	"github.com/ferret-linux/otter/pkg/ui"
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

var ErrNoContainersFound = errors.New("no containers found")

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
			return ErrNoContainersFound
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

	outcome := runBatch(ctx, containerNames, func(ctx context.Context, name string) (bool, error) {
		if _, err := c.containerManager.InspectContainer(ctx, name); err != nil {
			err = fmt.Errorf("container '%s' not found", name)
			ui.DefaultLogger.Error(err)
			return false, err
		}
		if err := c.containerManager.Stop(ctx, []string{name}, opts.Force); err != nil {
			err = fmt.Errorf("failed to stop '%s': %w", name, err)
			ui.DefaultLogger.Error(err)
			return false, err
		}
		ui.DefaultLogger.Info("stopped", "name", name)
		return false, nil
	})

	return summarizeBatch(outcome, batchSummaryConfig{
		PastVerb: "stopped",
		BaseVerb: "stop",
	})
}
