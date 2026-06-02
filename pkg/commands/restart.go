package commands

import (
	"context"
	"fmt"

	"github.com/ferret-linux/otter/pkg/config"
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
	cfg              *config.Values
	containerManager containermanager.ContainerManager
	stopCmd          *StopCommand
	startCmd         *StartCommand
	listCmd          *ListCommand
}

func NewRestartCommand(cfg *config.Values, cm containermanager.ContainerManager) *RestartCommand {
	return &RestartCommand{
		cfg:              cfg,
		containerManager: cm,
		stopCmd:          NewStopCommand(cfg, cm),
		startCmd:         NewStartCommand(cfg, cm),
		listCmd:          NewListCommand(cfg, cm),
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
		containerNames = []string{c.cfg.DefaultContainerName}
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
			ContainerName: name,
			DryRun:        opts.DryRun,
			Verbose:       opts.Verbose,
		}); err != nil {
			return fmt.Errorf("failed to start '%s': %w", name, err)
		}
	}

	return nil
}
