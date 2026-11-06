//nolint:goconst // CLI flag strings are intentionally repeated per-command; they may diverge independently
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
		Usage:    "Manage the host export index",
		Commands: []*cli.Command{list, rebuild, prune, deleteCmd},
	}
}

func newExportsListCommand(_ *config.Values) *cli.Command {
	return &cli.Command{
		Name:    "list",
		Aliases: []string{"ls"},
		Usage:   "List every recorded export",
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
		Usage: "Rebuild the export index from export files on disk",
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:  "scan-path",
				Usage: "Additional file or directory to scan (repeatable)",
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
		Usage: "Remove stale records and orphaned export files",
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:  "scan-path",
				Usage: "Additional file or directory to scan (repeatable)",
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
		Usage:     "Delete one export by its host path",
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
