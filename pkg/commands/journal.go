package commands

import (
	"context"
	"fmt"

	"github.com/ferret-linux/otter/pkg/config"
	"github.com/ferret-linux/otter/pkg/containermanager"
)

type JournalOptions struct {
	ContainerName string
	Follow        bool
	Since         string
	Until         string
	Timestamps    bool
	Tail          int
}

type JournalCommand struct {
	cfg              *config.Values
	containerManager containermanager.ContainerManager
}

func NewJournalCommand(cfg *config.Values, cm containermanager.ContainerManager) *JournalCommand {
	return &JournalCommand{
		cfg:              cfg,
		containerManager: cm,
	}
}

func (c *JournalCommand) Execute(ctx context.Context, opts JournalOptions) error {
	containerName := opts.ContainerName
	if containerName == "" {
		containerName = c.cfg.DefaultContainerName
	}

	if !c.containerManager.Exists(ctx, containerName) {
		return fmt.Errorf("container '%s' not found", containerName)
	}

	return c.containerManager.Journal(ctx, containerName, containermanager.JournalOptions{
		Follow:     opts.Follow,
		Since:      opts.Since,
		Until:      opts.Until,
		Timestamps: opts.Timestamps,
		Tail:       opts.Tail,
	})
}
