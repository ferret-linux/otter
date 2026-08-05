package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/ferret-linux/otter/pkg/containermanager"
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
		if err := c.startOne(ctx, name); err != nil {
			ui.DefaultLogger.Error("%s", err)
			return false, err
		}
		return false, nil
	})

	return summarizeBatch(outcome, batchSummaryConfig{
		PastVerb: "started",
		BaseVerb: "start",
	})
}

func (c *StartCommand) startOne(ctx context.Context, containerName string) error {
	if err := c.containerManager.Start(ctx, containerName); err != nil {
		return fmt.Errorf("failed to start container '%s': %w", containerName, err)
	}
	return nil
}
