package commands

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/ferret-linux/otter/pkg/containermanager"
)

type ListResult struct {
	Containers []containermanager.Container
}

type ListCommand struct {
	containerManager containermanager.ContainerManager
}

func NewListCommand(cm containermanager.ContainerManager) *ListCommand {
	return &ListCommand{
		containerManager: cm,
	}
}

func (c *ListCommand) Execute(ctx context.Context) (*ListResult, error) {
	containers, err := c.containerManager.ListContainers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed while listing contaiers: %w", err)
	}

	var otterContainers []containermanager.Container
	for _, container := range containers {
		if container.IsOtterContainer() {
			otterContainers = append(otterContainers, container)
		}
	}

	// Sort by container name to keep `otter list` output stable for downstream UIs; see https://github.com/89luca89/distrobox/issues/2071.
	slices.SortFunc(otterContainers, func(a, b containermanager.Container) int {
		return strings.Compare(a.Name, b.Name)
	})

	return &ListResult{Containers: otterContainers}, nil
}
