package commands

import (
	"context"
	"fmt"

	"github.com/ferret-linux/otter/pkg/containermanager"
	"github.com/ferret-linux/otter/pkg/ui"
)

type EnterResult struct{}

type EnterOptions struct {
	ContainerName   string
	AdditionalFlags string
	CustomCommand   []string
	AddEnv          []string
	DryRun          bool
	NoTTY           bool
	Verbose         bool
	CleanPath       bool
	EmptyEnv        bool
	AutoStart       bool
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
	if !opts.DryRun {
		inspectResult, err := c.containerManager.InspectContainer(ctx, opts.ContainerName)
		if err != nil {
			return nil, fmt.Errorf("container '%s' not found", opts.ContainerName)
		}
		if inspectResult.ContainerStatus != containermanager.RunningStatus {
			if opts.AutoStart {
				ui.DefaultLogger.Info("starting '%s'...\n", opts.ContainerName)
				if err := c.containerManager.Start(ctx, opts.ContainerName, opts.DryRun); err != nil {
					return nil, fmt.Errorf("failed to start container '%s': %w", opts.ContainerName, err)
				}
			} else {
				return nil, fmt.Errorf("container '%s' is stopped, run 'otter start %s'", opts.ContainerName, opts.ContainerName)
			}
		}
		if !c.containerManager.IsSetupDone(ctx, opts.ContainerName) {
			return nil, fmt.Errorf("container '%s' is initializing, please wait", opts.ContainerName)
		}
	}

	cmdOpts := containermanager.EnterOptions{
		ContainerName:   opts.ContainerName,
		AdditionalFlags: opts.AdditionalFlags,
		CustomCommand:   opts.CustomCommand,
		AddEnv:          opts.AddEnv,
		DryRun:          opts.DryRun,
		NoTTY:           opts.NoTTY,
		Verbose:         opts.Verbose,
		CleanPath:       opts.CleanPath,
		EmptyEnv:        opts.EmptyEnv,
	}

	if !opts.DryRun && !opts.NoTTY {
		ui.DefaultLogger.Info("entering '%s'...\n", opts.ContainerName)
	}

	err := c.containerManager.Enter(ctx, cmdOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to enter the container: %w", err)
	}

	if !opts.DryRun && !opts.NoTTY {
		fmt.Println()
		ui.DefaultLogger.Info("container still running — use 'otter stop %s' to stop it", opts.ContainerName)
	}

	return &EnterResult{}, nil
}
