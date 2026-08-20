package commands

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/ferret-linux/otter/pkg/containermanager"
	"github.com/ferret-linux/otter/pkg/ui"
)

type EnterResult struct{}

type EnterOptions struct {
	ContainerName   string
	AdditionalFlags string
	CustomCommand   []string
	AddEnv          []string
	NoTTY           bool
	CleanPath       bool
	EmptyEnv        bool
	NoWorkDir       bool
	// ForceTTY and Stdin/Stdout/Stderr are passed straight through to
	// containermanager.EnterOptions; see the doc comments there. Non-CLI
	// callers (e.g. the webui) set these, the CLI leaves them at zero value.
	ForceTTY bool
	Stdin    io.Reader
	Stdout   io.Writer
	Stderr   io.Writer
}

type EnterCommand struct {
	containerManager containermanager.ContainerManager
}

func NewEnterCommand(
	cm containermanager.ContainerManager,
) *EnterCommand {
	return &EnterCommand{
		containerManager: cm,
	}
}

func (c *EnterCommand) Execute(ctx context.Context, opts EnterOptions) (*EnterResult, error) {
	if strings.Contains(opts.ContainerName, ",") {
		return nil, errors.New("enter only accepts a single container name")
	}

	inspectResult, err := c.containerManager.InspectContainer(ctx, opts.ContainerName)
	if err != nil {
		return nil, fmt.Errorf("container '%s' not found", opts.ContainerName)
	}
	if inspectResult.ContainerStatus != containermanager.RunningStatus {
		ui.DefaultLogger.Info("starting...", "container", opts.ContainerName)
		if err := c.containerManager.Start(ctx, opts.ContainerName); err != nil {
			return nil, fmt.Errorf("failed to start container '%s': %w", opts.ContainerName, err)
		}
	}
	if !c.containerManager.IsSetupDone(ctx, opts.ContainerName) {
		return nil, fmt.Errorf("container '%s' is initializing, please wait", opts.ContainerName)
	}

	cmdOpts := containermanager.EnterOptions{
		ContainerName:   opts.ContainerName,
		AdditionalFlags: opts.AdditionalFlags,
		CustomCommand:   opts.CustomCommand,
		AddEnv:          opts.AddEnv,
		NoTTY:           opts.NoTTY,
		CleanPath:       opts.CleanPath,
		EmptyEnv:        opts.EmptyEnv,
		NoWorkDir:       opts.NoWorkDir,
		ForceTTY:        opts.ForceTTY,
		Stdin:           opts.Stdin,
		Stdout:          opts.Stdout,
		Stderr:          opts.Stderr,
	}

	if !opts.NoTTY {
		ui.DefaultLogger.Info("entering...", "container", opts.ContainerName)
		if inspectResult.ContainerCustomHomeSource != "" {
			ui.DefaultLogger.Warn("This container may briefly show harmless errors on entry")
		}
	}

	err = c.containerManager.Enter(ctx, cmdOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to enter the container: %w", err)
	}

	if !opts.NoTTY {
		fmt.Println() //nolint:forbidigo // blank line separator after interactive session output
		ui.DefaultLogger.Info("container still running — use 'otter stop' to stop it", "container", opts.ContainerName)
	}

	return &EnterResult{}, nil
}
