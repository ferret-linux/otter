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
	stopCmd  *StopCommand
	startCmd *StartCommand
	listCmd  *ListCommand
}

func NewRestartCommand(cm containermanager.ContainerManager) *RestartCommand {
	return &RestartCommand{
		stopCmd:  NewStopCommand(cm),
		startCmd: NewStartCommand(cm),
		listCmd:  NewListCommand(cm),
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
		if err := c.stopCmd.Execute(ctx, &StopOptions{
			ContainerNames: []string{name},
			Force:          opts.Force,
		}); err != nil {
			ui.DefaultLogger.Error("failed to restart '%s': %s", name, err)
			return false, err
		}
		if err := c.startCmd.Execute(ctx, &StartOptions{
			ContainerNames: []string{name},
		}); err != nil {
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
