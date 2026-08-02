package commands

import (
	"context"
	"errors"
	"fmt"
	"strings"

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
	containerManager containermanager.ContainerManager
}

func NewJournalCommand(_ *config.Values, cm containermanager.ContainerManager) *JournalCommand {
	return &JournalCommand{
		containerManager: cm,
	}
}

func (c *JournalCommand) Execute(ctx context.Context, opts JournalOptions) error {
	if opts.ContainerName == "" {
		return errors.New("please specify a container name")
	}

	if strings.Contains(opts.ContainerName, ",") {
		return errors.New("journal only accepts a single container name")
	}

	if !c.containerManager.Exists(ctx, opts.ContainerName) {
		return fmt.Errorf("container '%s' not found", opts.ContainerName)
	}

	if err := c.containerManager.Journal(ctx, opts.ContainerName, containermanager.JournalOptions{
		Follow:     opts.Follow,
		Since:      opts.Since,
		Until:      opts.Until,
		Timestamps: opts.Timestamps,
		Tail:       opts.Tail,
	}); err != nil {
		return fmt.Errorf("failed to get container journal: %w", err)
	}
	return nil
}
