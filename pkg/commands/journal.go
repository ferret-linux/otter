package commands

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/ferret-linux/otter/pkg/containermanager"
)

type JournalOptions struct {
	ContainerName string
	Follow        bool
	Since         string
	Until         string
	Timestamps    bool
	Tail          int
	// Stdout and Stderr are passed straight through to
	// containermanager.JournalOptions; see the doc comment there. Non-CLI
	// callers (e.g. the webui) set these, the CLI leaves them at zero value.
	Stdout io.Writer
	Stderr io.Writer
}

type JournalCommand struct {
	containerManager containermanager.ContainerManager
}

func NewJournalCommand(cm containermanager.ContainerManager) *JournalCommand {
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
		Stdout:     opts.Stdout,
		Stderr:     opts.Stderr,
	}); err != nil {
		return fmt.Errorf("failed to get container journal: %w", err)
	}
	return nil
}
