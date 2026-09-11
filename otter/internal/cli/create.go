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
				Usage:   "Image to use for the container",
			},
			&cli.StringFlag{
				Name:    "hostname",
				Aliases: []string{"n"},
				Usage:   "Hostname for the container",
			},
			&cli.StringFlag{
				Name:    "shell",
				Aliases: []string{"s"},
				Usage:   "Shell to use inside the container",
			},
			&cli.BoolFlag{
				Name:    "always-pull",
				Aliases: []string{"p"},
				Usage:   "Pull image even if it exists locally",
			},
			&cli.StringFlag{
				Name:    "clone",
				Aliases: []string{"C"},
				Usage:   "Name of container to use as base for new container",
			},
			&cli.StringFlag{
				Name:    "home",
				Aliases: []string{"H"},
				Usage:   "Custom HOME directory for the container",
			},
			&cli.StringSliceFlag{
				Name:    "volume",
				Aliases: []string{"v"},
				Usage:   "Additional volumes to add to the container",
			},
			&cli.StringSliceFlag{
				Name:    "additional-flags",
				Aliases: []string{"a"},
				Usage:   "Additional flags to pass to the container manager",
			},
			&cli.StringSliceFlag{
				Name:    "additional-packages",
				Aliases: []string{"ap"},
				Usage:   "Additional packages to install during container setup",
			},
			&cli.StringFlag{
				Name:    "init-hooks",
				Aliases: []string{"ih"},
				Usage:   "Commands to run at end of container initialization",
			},
			&cli.StringFlag{
				Name:    "pre-init-hooks",
				Aliases: []string{"ph"},
				Usage:   "Commands to run at start of container initialization",
			},
			&cli.BoolFlag{
				Name:    "init",
				Aliases: []string{"I"},
				Usage:   "Use init system (e.g. systemd) inside the container",
			},
			&cli.StringFlag{
				Name:    "memory",
				Aliases: []string{"m"},
				Usage:   "Memory limit for the container (e.g. 512m, 2g)",
			},
			&cli.IntFlag{
				Name:    "cpu-threads",
				Aliases: []string{"t"},
				Usage:   "Number of CPU threads the container can use",
			},
			&cli.StringFlag{
				Name:    "gpu",
				Aliases: []string{"g"},
				Usage:   "GPU mode: mesa, nvidia, or nvidia-toolkit",
			},
			&cli.BoolFlag{
				Name:    "no-userns-limit",
				Aliases: []string{"ul"},
				Usage:   "Skip the size-capped userns range for podman",
			},
			&cli.StringFlag{
				Name:    "platform",
				Aliases: []string{"P"},
				Usage:   "Specify the target platform, e.g. linux/arm64",
			},
			&cli.BoolFlag{
				Name:    "unshare-devsys",
				Aliases: []string{"ud"},
				Usage:   "Do not share host devices and sysfs dirs",
			},
			&cli.BoolFlag{
				Name:    "unshare-groups",
				Aliases: []string{"ug"},
				Usage:   "Do not forward user's additional groups into container",
			},
			&cli.BoolFlag{
				Name:    "unshare-ipc",
				Aliases: []string{"ui"},
				Usage:   "Do not share IPC namespace with host",
			},
			&cli.BoolFlag{
				Name:    "unshare-netns",
				Aliases: []string{"un"},
				Usage:   "Do not share net namespace with host",
			},
			&cli.BoolFlag{
				Name:    "unshare-process",
				Aliases: []string{"up"},
				Usage:   "Do not share process namespace with host",
			},
			&cli.BoolFlag{
				Name:    "unshare-all",
				Aliases: []string{"ua"},
				Usage:   "Activate all unshare flags",
			},
		&cli.BoolFlag{
			Name:    "no-entry",
			Aliases: []string{"E"},
			Usage:   "Do not generate a container entry on host system",
		},
		&cli.BoolFlag{
			Name:  "json",
			Usage: "Output progress as JSON",
		},
			&cli.BoolFlag{
				Name:  "disable-root-password-i-fully-understand-the-risks-and-accept-the-responsibilities",
				Usage: "Disable the root password inside the container",
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
