package commands

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ferret-linux/otter/pkg/config"
	"github.com/ferret-linux/otter/pkg/containermanager"
	"github.com/ferret-linux/otter/pkg/ui"
)

type StopCommand struct {
	cfg              *config.Values
	containerManager containermanager.ContainerManager
	listCmd          *ListCommand
}

type StopOptions struct {
	ContainerNames []string
	All            bool
	DryRun         bool
	Verbose        bool
}

var ErrEmptyContainerList = errors.New("cannot find containers to stop")

func NewStopCommand(cfg *config.Values, containerManager containermanager.ContainerManager) *StopCommand {
	return &StopCommand{
		cfg:              cfg,
		containerManager: containerManager,
		listCmd:          NewListCommand(cfg, containerManager),
	}
}

func (c *StopCommand) Execute(ctx context.Context, opts *StopOptions) error {
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
		containerNames = make([]string, 0, len(containers.Containers))
		for _, container := range containers.Containers {
			containerNames = append(containerNames, container.Name)
		}
	case len(opts.ContainerNames) > 0:
		containerNames = opts.ContainerNames
	default:
		containerNames = []string{c.cfg.DefaultContainerName}
	}

	if opts.Verbose {
		ui.DefaultLogger.Info("stopping: %s", strings.Join(containerNames, ", "))
	}

	err := c.containerManager.Stop(ctx, containerNames, opts.DryRun)
	if err != nil {
		return fmt.Errorf("failed to stop containers: %w", err)
	}

	return nil
}
