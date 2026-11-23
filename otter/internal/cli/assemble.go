//nolint:goconst // CLI flag/command names are declarative and intentionally repeated with their aliases/usage
package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/ferret-linux/otter/pkg/commands"
	"github.com/ferret-linux/otter/pkg/config"
	"github.com/ferret-linux/otter/pkg/rootcheck"
)

func newAssembleCommand(cfg *config.Values) *cli.Command {
	fileFlag := &cli.StringFlag{
		Name:    "file",
		Aliases: []string{"f"},
		Usage:   "Path to the manifest file (required)",
	}
	return &cli.Command{
		Name:    "assemble",
		Aliases: []string{"dmf"},
		Commands: []*cli.Command{
			{
				Name:    "create",
				Aliases: []string{"mk"},
				Flags: []cli.Flag{
					fileFlag,
					&cli.BoolFlag{
						Name:    "replace",
						Aliases: []string{"R"},
						Usage:   "Delete and recreate boxes that already exist",
					},
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
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return assembleAction(ctx, cmd, cfg, true)
				},
			},
		},
	}
}

func assembleAction(ctx context.Context, cmd *cli.Command, cfg *config.Values, deleteFlag bool) error {
	names, err := splitNames(cmd.Args().Slice())
	if err != nil {
		return err
	}

	if cmd.String("file") == "" {
		return errors.New("manifest path is required, use --file to specify it")
	}

	sudoCommand := escalatorFromContext(ctx)
	if sudoCommand == "" {
		sudoCommand = cmd.Root().String("sudo-command")
	}
	if sudoCommand == autodetectEscalator {
		// Assemble creates rootful boxes without setting --root on the
		// command, so resolve the escalator here once for both the manager
		// clones and the command options.
		resolved, err := rootcheck.Validate(ctx, sudoCommand)
		if err != nil {
			return fmt.Errorf("failed to resolve privilege escalator: %w", err)
		}
		sudoCommand = resolved
		if err := cmd.Root().Set("sudo-command", resolved); err != nil {
			return fmt.Errorf("failed to apply resolved escalator: %w", err)
		}
		ctx = withEscalator(ctx, resolved)
	}

	opts := commands.AssembleOptions{
		ManifestPath: cmd.String("file"),
		SudoCommand:  sudoCommand,
		BoxNames:     names,
	}
	if deleteFlag {
		opts.Delete = true
	} else {
		opts.Replace = cmd.Bool("replace")
	}

	containerManager, err := buildContainerManager(
		ctx,
		cmd.Root().String("container-manager"),
		sudoCommand,
		false,
	)
	if err != nil {
		return err
	}

	assembleCmd := commands.NewAssembleCommand(cfg, containerManager)

	err = assembleCmd.Execute(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to execute assemble command: %w", err)
	}
	return nil
}
