package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/ferret-linux/otter/internal/rootful"
	"github.com/ferret-linux/otter/pkg/commands"
	"github.com/ferret-linux/otter/pkg/config"
	"github.com/ferret-linux/otter/pkg/containermanager"
	"github.com/ferret-linux/otter/pkg/manifest"
	"github.com/ferret-linux/otter/pkg/ui"
)

func newAssembleCommand(cfg *config.Values) *cli.Command {
	fileFlag := &cli.StringFlag{Name: "file"}
	nameFlag := &cli.StringFlag{
		Name:    "name",
		Aliases: []string{"n"},
	}
	dryRunFlag := &cli.BoolFlag{
		Name:    "dry-run",
		Aliases: []string{"d"},
	}

	return &cli.Command{
		Name: "assemble",
		Commands: []*cli.Command{
			{
				Name: "create",
				Flags: []cli.Flag{
					fileFlag,
					nameFlag,
					&cli.BoolFlag{
						Name:    "replace",
						Aliases: []string{"R"},
					},
					dryRunFlag,
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return assembleAction(ctx, cmd, cfg, false)
				},
			},
			{
				Name:    "remove",
				Aliases: []string{"rm"},
				Flags: []cli.Flag{
					fileFlag,
					nameFlag,
					dryRunFlag,
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return assembleAction(ctx, cmd, cfg, true)
				},
			},
		},
	}
}

const defaultManifestPath = "./otter.ini"

func resolveManifestPath(flagValue string, positional []string) string {
	if flagValue != "" {
		return flagValue
	}
	if len(positional) > 0 && positional[0] != "" {
		return positional[0]
	}
	return defaultManifestPath
}

func assembleAction(ctx context.Context, cmd *cli.Command, cfg *config.Values, deleteFlag bool) error {
	containerManager, ok := ctx.Value(containerManagerKey).(containermanager.ContainerManager)
	if !ok {
		return errors.New("container manager not found in context")
	}

	manifestFilePath := resolveManifestPath(cmd.String("file"), cmd.Args().Slice())

	manifest, err := manifest.Parse(ctx, manifestFilePath)
	if err != nil {
		return fmt.Errorf("failed to parse manifest file: %w", err)
	}

	// if at least one item in the manifest requires root, validate sudo before proceeding
	for _, item := range manifest {
		if item.Root {
			if err := rootful.Validate(ctx, cmd.String("sudo-command")); err != nil {
				return fmt.Errorf("cannot run in root mode: %w", err)
			}
			break
		}
	}

	opts := commands.AssembleOptions{
		Items:   manifest,
		Boxname: cmd.String("name"),
		DryRun:  cmd.Bool("dry-run"),
	}
	if deleteFlag {
		opts.Delete = true
	} else {
		opts.Replace = cmd.Bool("replace")
	}

	prompter := ui.NewPrompter(*bufio.NewReader(os.Stdin), os.Stdout)
	progress := ui.NewProgress(os.Stderr)
	assembleCmd := commands.NewAssembleCommand(cfg, containerManager, prompter, progress)

	err = assembleCmd.Execute(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to execute assemble command: %w", err)
	}
	return nil
}
