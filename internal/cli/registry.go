package cli

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/ferret-linux/otter/pkg/commands"
	"github.com/ferret-linux/otter/pkg/config"
	"github.com/ferret-linux/otter/pkg/containermanager"
	"github.com/ferret-linux/otter/pkg/registry"
	"github.com/ferret-linux/otter/pkg/ui"
)

func newRegistryCommand(cfg *config.Values) *cli.Command {
	cc := &CommandComposer[config.Values]{cfg: cfg}

	pull := cc.apply(
		newRegistryPullCommand,
		withUsageErrorHandler,
		withRoot,
		withContainerManager,
	)

	remove := cc.apply(
		newRegistryRemoveCommand,
		withUsageErrorHandler,
		withRoot,
		withContainerManager,
	)

	return &cli.Command{
		Name:    "registry",
		Aliases: []string{"img"},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "all",
				Aliases: []string{"a"},
			},
		},
		Action:   registryAction,
		Commands: []*cli.Command{pull, remove},
	}
}

func registryAction(_ context.Context, cmd *cli.Command) error {
	props, err := registry.Fetch()
	if err != nil {
		return fmt.Errorf("failed to fetch registry: %w", err)
	}
	commands.RegistryList(props, cmd.Bool("all"))
	return nil
}

func newRegistryPullCommand(_ *config.Values) *cli.Command {
	return &cli.Command{
		Name:    "pull",
		Aliases: []string{"get"},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "all",
				Aliases: []string{"a"},
			},
			&cli.BoolFlag{
				Name:    "force",
				Aliases: []string{"f"},
			},
		},
		Action: registryPullAction,
	}
}

func registryPullAction(ctx context.Context, cmd *cli.Command) error {
	containerManager, ok := ctx.Value(containerManagerKey).(containermanager.ContainerManager)
	if !ok {
		return errors.New("container manager not found in context")
	}

	names, err := splitNames(cmd.Args().Slice())
	if err != nil {
		return err
	}

	props, err := registry.Fetch()
	if err != nil {
		return fmt.Errorf("failed to fetch registry: %w", err)
	}

	progress := ui.NewProgress(os.Stderr)

	if err := commands.RegistryPull(
		ctx,
		containerManager,
		props,
		names,
		cmd.Bool("all"),
		cmd.Bool("force"),
		progress,
	); err != nil {
		return fmt.Errorf("failed to pull from registry: %w", err)
	}
	return nil
}

func newRegistryRemoveCommand(_ *config.Values) *cli.Command {
	return &cli.Command{
		Name:    "remove",
		Aliases: []string{"rm"},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "all",
				Aliases: []string{"a"},
			},
			&cli.BoolFlag{
				Name:    "force",
				Aliases: []string{"f"},
			},
		},
		Action: registryRemoveAction,
	}
}

func registryRemoveAction(ctx context.Context, cmd *cli.Command) error {
	containerManager, ok := ctx.Value(containerManagerKey).(containermanager.ContainerManager)
	if !ok {
		return errors.New("container manager not found in context")
	}

	names, err := splitNames(cmd.Args().Slice())
	if err != nil {
		return err
	}

	props, err := registry.Fetch()
	if err != nil {
		return fmt.Errorf("failed to fetch registry: %w", err)
	}

	if err := commands.RegistryRemove(
		ctx,
		containerManager,
		props,
		names,
		cmd.Bool("all"),
		cmd.Bool("force"),
	); err != nil {
		return fmt.Errorf("failed to remove from registry: %w", err)
	}
	return nil
}
