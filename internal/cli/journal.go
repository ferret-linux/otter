package cli

import (
	"context"
	"errors"

	"github.com/urfave/cli/v3"

	"github.com/ferret-linux/otter/pkg/commands"
	"github.com/ferret-linux/otter/pkg/config"
	"github.com/ferret-linux/otter/pkg/containermanager"
)

func newJournalCommand(cfg *config.Values) *cli.Command {
	return &cli.Command{
		Name:      "journal",
		Aliases:   []string{"logs"},
		ArgsUsage: "CONTAINER",
		Flags: []cli.Flag{
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

	containerName := ""
	if cmd.NArg() > 0 {
		containerName = cmd.Args().First()
	}

	journalCmd := commands.NewJournalCommand(cfg, cm)
	return journalCmd.Execute(ctx, commands.JournalOptions{
		ContainerName: containerName,
		Follow:        cmd.Bool("follow"),
		Since:         cmd.String("since"),
		Until:         cmd.String("until"),
		Timestamps:    cmd.Bool("timestamps"),
		Tail:          cmd.Int("tail"),
	})
}
