package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/ferret-linux/otter/pkg/containermanager"
	"github.com/ferret-linux/otter/pkg/ui"
)

type RestartOptions struct {
	ContainerNames []string
	All            bool
	Force          bool
}

type RestartCommand struct {
	containerManager containermanager.ContainerManager
	listCmd          *ListCommand
}

func NewRestartCommand(cm containermanager.ContainerManager) *RestartCommand {
	return &RestartCommand{
		containerManager: cm,
		listCmd:          NewListCommand(cm),
	}
}

func (c *RestartCommand) Execute(ctx context.Context, opts *RestartOptions) error {
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
		for _, container := range containers.Containers {
			containerNames = append(containerNames, container.Name)
		}
	case len(opts.ContainerNames) > 0:
		containerNames = opts.ContainerNames
	default:
		return errors.New("please specify a container name or use --all")
	}

	outcome := runBatch(ctx, containerNames, func(ctx context.Context, name string) (bool, error) {
		if err := c.containerManager.Stop(ctx, []string{name}, opts.Force); err != nil {
			ui.DefaultLogger.Error("failed to restart '%s': %s", name, err)
			return false, err
		}
		if err := c.containerManager.Start(ctx, name); err != nil {
			ui.DefaultLogger.Error("failed to restart '%s': %s", name, err)
			return false, err
		}
		ui.DefaultLogger.Ok("restarted '%s'", name)
		return false, nil
	})

	return summarizeBatch(outcome, batchSummaryConfig{
		PastVerb: "restarted",
		BaseVerb: "restart",
	})
}
