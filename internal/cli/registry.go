//nolint:goconst // CLI flag strings are intentionally repeated per-command; they may diverge independently
package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/ferret-linux/otter/pkg/commands"
	"github.com/ferret-linux/otter/pkg/config"
	"github.com/ferret-linux/otter/pkg/containermanager"
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

	list := cc.apply(
		newRegistryListCommand,
		withUsageErrorHandler,
		withRoot,
		withContainerManager,
	)

	return &cli.Command{
		Name:     "registry",
		Aliases:  []string{"reg"},
		Commands: []*cli.Command{list, pull, remove},
	}
}

func newRegistryListCommand(_ *config.Values) *cli.Command {
	return &cli.Command{
		Name:    "list",
		Aliases: []string{"ls"},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "all",
				Aliases: []string{"a"},
			},
			&cli.BoolFlag{
				Name:    "json",
				Aliases: []string{"j"},
			},
		},
		Action: registryListAction,
	}
}

func registryListAction(ctx context.Context, cmd *cli.Command) error {
	containerManager, ok := ctx.Value(containerManagerKey).(containermanager.ContainerManager)
	if !ok {
		return errors.New("container manager not found in context")
	}

	if err := commands.NewRegistryListCommand(containerManager).Execute(ctx, commands.RegistryListOptions{
		All:  cmd.Bool("all"),
		JSON: cmd.Bool("json"),
	}); err != nil {
		return fmt.Errorf("failed to list registry: %w", err)
	}
	return nil
}

func newRegistryPullCommand(_ *config.Values) *cli.Command {
	return &cli.Command{
		Name:    "pull",
		Aliases: []string{"pl"},
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

	if err := commands.NewRegistryPullCommand(containerManager).Execute(ctx, commands.RegistryPullOptions{
		Names: names,
		All:   cmd.Bool("all"),
		Force: cmd.Bool("force"),
	}); err != nil {
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

	if err := commands.NewRegistryRemoveCommand(containerManager).Execute(ctx, commands.RegistryRemoveOptions{
		Names: names,
		All:   cmd.Bool("all"),
		Force: cmd.Bool("force"),
	}); err != nil {
		return fmt.Errorf("failed to remove from registry: %w", err)
	}
	return nil
}
