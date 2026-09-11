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

func newJournalCommand(_ *config.Values) *cli.Command {
	return &cli.Command{
		Name:    "journal",
		Aliases: []string{"logs"},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "follow",
				Aliases: []string{"f"},
				Usage:   "Stream logs live",
			},
			&cli.StringFlag{
				Name:    "since",
				Aliases: []string{"s"},
				Usage:   "Show logs since duration",
			},
			&cli.StringFlag{
				Name:    "until",
				Aliases: []string{"u"},
				Usage:   "Show logs until duration",
			},
			&cli.BoolFlag{
				Name:    "timestamps",
				Aliases: []string{"t"},
				Usage:   "Show timestamps on each line",
			},
			&cli.IntFlag{
				Name:    "tail",
				Aliases: []string{"n"},
				Value:   -1,
				Usage:   "Show last N lines",
			},
		},
		Action: journalAction,
	}
}

func journalAction(ctx context.Context, cmd *cli.Command) error {
	cm, ok := ctx.Value(containerManagerKey).(containermanager.ContainerManager)
	if !ok {
		return errors.New("container manager not found in context")
	}

	journalCmd := commands.NewJournalCommand(cm)
	if err := journalCmd.Execute(ctx, commands.JournalOptions{
		ContainerName: firstName(cmd.Args().Slice()),
		Follow:        cmd.Bool("follow"),
		Since:         cmd.String("since"),
		Until:         cmd.String("until"),
		Timestamps:    cmd.Bool("timestamps"),
		Tail:          cmd.Int("tail"),
	}); err != nil {
		return fmt.Errorf("failed to get journal: %w", err)
	}
	return nil
}
