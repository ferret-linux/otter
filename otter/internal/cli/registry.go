//nolint:goconst // CLI flag/command names are declarative and intentionally repeated with their aliases/usage
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
		withRootRootful,
		withContainerManager,
	)

	remove := cc.apply(
		newRegistryRemoveCommand,
		withUsageErrorHandler,
		withRootRootful,
		withContainerManager,
	)

	list := cc.apply(
		newRegistryListCommand,
		withUsageErrorHandler,
		withRootRootful,
		withContainerManager,
	)

	return &cli.Command{
		Name:     "registry",
		Aliases:  []string{"reg"},
		Usage:    "Browse and manage otter images",
		Commands: []*cli.Command{list, pull, remove},
	}
}

func newRegistryListCommand(_ *config.Values) *cli.Command {
	return &cli.Command{
		Name:    "list",
		Aliases: []string{"ls"},
		Usage:   "List otter images available in the registry",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "all",
				Aliases: []string{"a"},
				Usage:   "Include disabled images",
			},
			&cli.BoolFlag{
				Name:    "json",
				Aliases: []string{"j"},
				Usage:   "Output as JSON",
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
		Usage:   "Pull otter images from the registry",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "all",
				Aliases: []string{"a"},
				Usage:   "Pull all missing enabled images",
			},
			&cli.BoolFlag{
				Name:    "force",
				Aliases: []string{"f"},
				Usage:   "Force the operation",
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
		Usage:   "Remove otter images from the local registry cache",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "all",
				Aliases: []string{"a"},
				Usage:   "Remove all cached images",
			},
			&cli.BoolFlag{
				Name:    "force",
				Aliases: []string{"f"},
				Usage:   "Force the operation",
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
