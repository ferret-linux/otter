package commands

import (
	"context"
	"fmt"

	"github.com/ferret-linux/otter/pkg/containermanager"
	"github.com/ferret-linux/otter/pkg/ui"
)

type RestartOptions struct {
	ContainerNames []string
	All            bool
	DryRun         bool
	Verbose        bool
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
		containers, err := c.listCmd.Execute(ctx)
		if err != nil {
			return fmt.Errorf("failed to list containers: %w", err)
		}
		if len(containers.Containers) == 0 {
			return ErrEmptyContainerList
		}
		for _, container := range containers.Containers {
			containerNames = append(containerNames, container.Name)
		}
	case len(opts.ContainerNames) > 0:
		containerNames = opts.ContainerNames
	default:
		return fmt.Errorf("please specify a container name with --name/-n")
	}

	for _, name := range containerNames {
		if opts.Verbose {
			ui.DefaultLogger.Info("restarting: %s", name)
		}
		if err := c.stopCmd.Execute(ctx, &StopOptions{
			ContainerNames: []string{name},
			DryRun:         opts.DryRun,
			Verbose:        opts.Verbose,
		}); err != nil {
			return fmt.Errorf("failed to stop '%s': %w", name, err)
		}
		if err := c.startCmd.Execute(ctx, &StartOptions{
			ContainerNames: []string{name},
			DryRun:         opts.DryRun,
			Verbose:        opts.Verbose,
		}); err != nil {
			return fmt.Errorf("failed to start '%s': %w", name, err)
		}
	}

	return nil
}
