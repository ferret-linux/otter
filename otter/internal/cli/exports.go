//nolint:goconst // CLI flag/command names are declarative and intentionally repeated with their aliases/usage
package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/ferret-linux/otter/pkg/commands"
	"github.com/ferret-linux/otter/pkg/config"
)

func newExportsCommand(cfg *config.Values) *cli.Command {
	cc := &CommandComposer[config.Values]{cfg: cfg}

	list := cc.apply(newExportsListCommand, withUsageErrorHandler)
	rebuild := cc.apply(newExportsRebuildCommand, withUsageErrorHandler)
	prune := cc.apply(newExportsPruneCommand, withUsageErrorHandler)
	deleteCmd := cc.apply(newExportsDeleteCommand, withUsageErrorHandler)

	return &cli.Command{
		Name:     "exports",
		Aliases:  []string{"exp"},
		Usage:    "Manage host export index",
		Commands: []*cli.Command{list, rebuild, prune, deleteCmd},
	}
}

func newExportsListCommand(_ *config.Values) *cli.Command {
	return &cli.Command{
		Name:    "list",
		Aliases: []string{"ls"},
		Usage:   "List all recorded exports",
		Action: func(ctx context.Context, _ *cli.Command) error {
			if _, err := commands.NewExportsListCommand().Execute(ctx, commands.ExportsListOptions{}); err != nil {
				return fmt.Errorf("failed to list exports: %w", err)
			}
			return nil
		},
	}
}

func newExportsRebuildCommand(_ *config.Values) *cli.Command {
	return &cli.Command{
		Name:  "rebuild",
		Usage: "Rebuild index from markers",
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:  "scan-path",
				Usage: "Extra export path to scan",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if _, err := commands.NewExportsRebuildCommand().Execute(ctx, commands.ExportsRebuildOptions{
				ScanPaths: cmd.StringSlice("scan-path"),
			}); err != nil {
				return fmt.Errorf("failed to rebuild export index: %w", err)
			}
			return nil
		},
	}
}

func newExportsPruneCommand(_ *config.Values) *cli.Command {
	return &cli.Command{
		Name:  "prune",
		Usage: "Drop stale records and orphans",
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:  "scan-path",
				Usage: "Extra export path to scan",
			},
			&cli.BoolFlag{
				Name:    "root",
				Aliases: []string{"r"},
				Usage:   "Include rootful owners",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if _, err := commands.NewExportsPruneCommand().Execute(ctx, commands.ExportsPruneOptions{
				ScanPaths: cmd.StringSlice("scan-path"),
				Root:      cmd.Bool("root"),
			}); err != nil {
				return fmt.Errorf("failed to prune exports: %w", err)
			}
			return nil
		},
	}
}

func newExportsDeleteCommand(_ *config.Values) *cli.Command {
	return &cli.Command{
		Name:      "delete",
		Aliases:   []string{"rm"},
		Usage:     "Delete one export by path",
		ArgsUsage: "<path>",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() != 1 {
				return errors.New("expected exactly one export path")
			}
			if err := commands.NewExportsDeleteCommand().Execute(ctx, commands.ExportsDeleteOptions{
				Path: cmd.Args().First(),
			}); err != nil {
				return fmt.Errorf("failed to delete export: %w", err)
			}
			return nil
		},
	}
}
