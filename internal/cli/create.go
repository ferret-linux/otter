package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/ferret-linux/otter/pkg/commands"
	"github.com/ferret-linux/otter/pkg/config"
	"github.com/ferret-linux/otter/pkg/containermanager"
	"github.com/ferret-linux/otter/pkg/ui"
)

//nolint:funlen // function length is acceptable for CLI command definition
func newCreateCommand(cfg *config.Values) *cli.Command {
	return &cli.Command{
		Name:    "create",
		Aliases: []string{"mk"},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "image",
				Aliases: []string{"i"},
			},
			&cli.StringFlag{
				Name:    "hostname",
				Aliases: []string{"n"},
			},
			&cli.StringFlag{
				Name:    "shell",
				Aliases: []string{"s"},
			},
			&cli.BoolFlag{
				Name:    "always-pull",
				Aliases: []string{"p"},
			},
			&cli.StringFlag{
				Name:    "clone",
				Aliases: []string{"C"},
			},
			&cli.StringFlag{
				Name:    "home",
				Aliases: []string{"H"},
			},
			&cli.StringSliceFlag{
				Name:    "volume",
				Aliases: []string{"v"},
			},
			&cli.StringSliceFlag{
				Name:    "additional-flags",
				Aliases: []string{"a"},
			},
			&cli.StringSliceFlag{
				Name:    "additional-packages",
				Aliases: []string{"ap"},
			},
			&cli.StringFlag{
				Name:    "init-hooks",
				Aliases: []string{"ih"},
			},
			&cli.StringFlag{
				Name:    "pre-init-hooks",
				Aliases: []string{"ph"},
			},
			&cli.BoolFlag{
				Name:    "init",
				Aliases: []string{"I"},
			},
			&cli.StringFlag{
				Name:    "memory",
				Aliases: []string{"m"},
			},
			&cli.IntFlag{
				Name:    "cpu-threads",
				Aliases: []string{"t"},
			},
			&cli.StringFlag{
				Name:    "gpu",
				Aliases: []string{"g"},
			},
			&cli.BoolFlag{
				Name:    "no-userns-limit",
				Aliases: []string{"ul"},
			},
			&cli.StringFlag{
				Name:    "platform",
				Aliases: []string{"P"},
			},
			&cli.BoolFlag{
				Name:    "unshare-devsys",
				Aliases: []string{"ud"},
			},
			&cli.BoolFlag{
				Name:    "unshare-groups",
				Aliases: []string{"ug"},
			},
			&cli.BoolFlag{
				Name:    "unshare-ipc",
				Aliases: []string{"ui"},
			},
			&cli.BoolFlag{
				Name:    "unshare-netns",
				Aliases: []string{"un"},
			},
			&cli.BoolFlag{
				Name:    "unshare-process",
				Aliases: []string{"up"},
			},
			&cli.BoolFlag{
				Name:    "unshare-all",
				Aliases: []string{"ua"},
			},
		&cli.BoolFlag{
			Name:    "no-entry",
			Aliases: []string{"E"},
		},
		&cli.BoolFlag{
			Name: "json",
		},
			&cli.BoolFlag{
				Name: "disable-root-password-i-fully-understand-the-risks-and-accept-the-responsibilities",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return createAction(ctx, cmd, cfg)
		},
	}
}

func createAction(ctx context.Context, cmd *cli.Command, cfg *config.Values) error {
	containerManager, ok := ctx.Value(containerManagerKey).(containermanager.ContainerManager)
	if !ok {
		return errors.New("container manager not found in context")
	}

	opts := commands.CreateOptions{
		ContainerImage:          cmd.String("image"),
		ContainerName:           firstName(cmd.Args().Slice()),
		ContainerHostname:       cmd.String("hostname"),
		ContainerClone:          cmd.String("clone"),
		UnshareNetNs:            cmd.Bool("unshare-netns") || cmd.Bool("unshare-all"),
		UnshareDevsys:           cmd.Bool("unshare-devsys") || cmd.Bool("unshare-all"),
		UnshareGroups:           cmd.Bool("unshare-groups") || cmd.Bool("unshare-all") || cmd.Bool("init"),
		UnshareIpc:              cmd.Bool("unshare-ipc") || cmd.Bool("unshare-all"),
		UnshareProcess:          cmd.Bool("unshare-process") || cmd.Bool("unshare-all") || cmd.Bool("init"),
		AdditionalFlags:         cmd.StringSlice("additional-flags"),
		AdditionalVolumes:       cmd.StringSlice("volume"),
		AdditionalPackages:      cmd.StringSlice("additional-packages"),
		Nopasswd:                cmd.Bool("disable-root-password-i-fully-understand-the-risks-and-accept-the-responsibilities"),
		ContainerUserCustomHome: cmd.String("home"),
		Init:                    cmd.Bool("init") || cfg.DefaultInitSystem,
		GPU:                     cmd.String("gpu"),
		NoUsernsLimit:           cmd.Bool("no-userns-limit") || cfg.DefaultUsernsNoLimit,
		Memory:                  cmd.String("memory"),
		CPUThreads:              cmd.Int("cpu-threads"),
		ContainerInitHook:       cmd.String("init-hooks"),
		ContainerPreInitHook:    cmd.String("pre-init-hooks"),
		ContainerShell:          cmd.String("shell"),
		ContainerPlatform:       cmd.String("platform"),
		GenerateEntry:           !cmd.Bool("no-entry") && !cfg.DefaultNoEntry,
		Rootful:                 cmd.Bool("root"),
		ContainerAlwaysPull:     cmd.Bool("always-pull"),
	}

	var progress *ui.Progress
	if cmd.Bool("json") {
		progress = ui.NewJSONProgress()
	}
	createCmd := commands.NewCreateCommand(cfg, containerManager, progress)
	if _, err := createCmd.Execute(ctx, opts); err != nil {
		return fmt.Errorf("create command failed: %w", err)
	}

	return nil
}
