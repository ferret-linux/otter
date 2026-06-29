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

func newAssembleCommand(cfg *config.Values) *cli.Command {
	fileFlag := &cli.StringFlag{Name: "file", Aliases: []string{"f"}}
	return &cli.Command{
		Name:    "assemble",
		Aliases: []string{"dmk"},
		Commands: []*cli.Command{
			{
				Name:    "create",
				Aliases: []string{"mk"},
				Flags: []cli.Flag{
					fileFlag,
					&cli.BoolFlag{
						Name:    "replace",
						Aliases: []string{"R"},
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
	containerManager, ok := ctx.Value(containerManagerKey).(containermanager.ContainerManager)
	if !ok {
		return errors.New("container manager not found in context")
	}

	opts := commands.AssembleOptions{
		ManifestPath: cmd.String("file"),
		SudoCommand:  cmd.String("sudo-command"),
		Boxname:      firstName(cmd.Args().Slice()),
	}
	if deleteFlag {
		opts.Delete = true
	} else {
		opts.Replace = cmd.Bool("replace")
	}

	assembleCmd := commands.NewAssembleCommand(cfg, containerManager)

	err := assembleCmd.Execute(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to execute assemble command: %w", err)
	}
	return nil
}
