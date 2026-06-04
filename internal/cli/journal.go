package cli

import (
	"context"
	"errors"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/ferret-linux/otter/pkg/commands"
	"github.com/ferret-linux/otter/pkg/config"
	"github.com/ferret-linux/otter/pkg/containermanager"
)

func newJournalCommand(cfg *config.Values) *cli.Command {
	return &cli.Command{
		Name:    "journal",
		Aliases: []string{"logs"},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "name",
				Aliases: []string{"n"},
			},
			&cli.BoolFlag{
				Name:    "follow",
				Aliases: []string{"f"},
			},
			&cli.StringFlag{
				Name: "since",
			},
			&cli.StringFlag{
				Name: "until",
			},
			&cli.BoolFlag{
				Name:    "timestamps",
				Aliases: []string{"t"},
			},
			&cli.IntFlag{
				Name:  "tail",
				Value: -1,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return journalAction(ctx, cmd, cfg)
		},
	}
}

func journalAction(ctx context.Context, cmd *cli.Command, cfg *config.Values) error {
	cm, ok := ctx.Value(containerManagerKey).(containermanager.ContainerManager)
	if !ok {
		return errors.New("container manager not found in context")
	}

	if strings.Contains(cmd.String("name"), ",") {
		return errors.New("journal only accepts a single container name")
	}

	journalCmd := commands.NewJournalCommand(cfg, cm)
	return journalCmd.Execute(ctx, commands.JournalOptions{
		ContainerName: cmd.String("name"),
		Follow:        cmd.Bool("follow"),
		Since:         cmd.String("since"),
		Until:         cmd.String("until"),
		Timestamps:    cmd.Bool("timestamps"),
		Tail:          cmd.Int("tail"),
	})
}
