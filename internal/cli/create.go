package cli

import (
	"context"
	"errors"
	"fmt"
	"os"

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
				Name: "hostname",
			},
			&cli.StringFlag{
				Name:    "shell",
				Aliases: []string{"s"},
				Action: func(_ context.Context, _ *cli.Command, v string) error {
					switch v {
					case "bash", "zsh", "fish":
						return nil
					default:
						return fmt.Errorf("invalid shell %q, must be one of: bash, zsh, fish", v)
					}
				},
			},
			&cli.BoolFlag{
				Name:    "pull",
				Aliases: []string{"p"},
			},
			&cli.BoolFlag{
				Name:    "yes",
				Aliases: []string{"y"},
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
				Name: "volume",
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
				Name: "init-hooks",
			},
			&cli.StringFlag{
				Name: "pre-init-hooks",
			},
			&cli.BoolFlag{
				Name:    "init",
				Aliases: []string{"I"},
			},
			&cli.StringFlag{
				Name: "memory",
			},
			&cli.IntFlag{
				Name: "cpu-threads",
			},
			&cli.BoolFlag{
				Name: "nvidia",
			},
			&cli.StringFlag{
				Name: "platform",
			},
			&cli.BoolFlag{
				Name: "unshare-devsys",
			},
			&cli.BoolFlag{
				Name: "unshare-groups",
			},
			&cli.BoolFlag{
				Name: "unshare-ipc",
			},
			&cli.BoolFlag{
				Name: "unshare-netns",
			},
			&cli.BoolFlag{
				Name: "unshare-process",
			},
			&cli.BoolFlag{
				Name: "unshare-all",
			},
			&cli.BoolFlag{
				Name: "no-entry",
			},
			&cli.BoolFlag{
				Name: "absolutely-disable-root-password-i-am-really-positively-sure",
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
		Nopasswd:                cmd.Bool("absolutely-disable-root-password-i-am-really-positively-sure"),
		ContainerUserCustomHome: cmd.String("home"),
		Init:                    cmd.Bool("init"),
		Nvidia:                  cmd.Bool("nvidia"),
		Memory:                  cmd.String("memory"),
		CPUThreads:              cmd.Int("cpu-threads"),
		ContainerInitHook:       cmd.String("init-hooks"),
		ContainerPreInitHook:    cmd.String("pre-init-hooks"),
		ContainerShell:          cmd.String("shell"),
		ContainerPlatform:       cmd.String("platform"),
		GenerateEntry:           !cmd.Bool("no-entry") && !cfg.DefaultNoEntry,
		Rootful:                 cmd.Bool("root"),
		ContainerAlwaysPull:     cmd.Bool("pull"),
	}

	progress := ui.NewProgress(os.Stderr)

	createCmd := commands.NewCreateCommand(cfg, containerManager, progress)
	result, err := createCmd.Execute(ctx, opts)
	if err != nil {
		return fmt.Errorf("create command failed: %w", err)
	}

	if result.AlreadyExisted {
		printContainerAlreadyExists(progress, result.ContainerName, opts.Rootful)
		return nil
	}

	printCreateCompleted(progress, result.ContainerName, opts.Rootful)

	return nil
}

func printCreateCompleted(progress *ui.Progress, containerName string, rootful bool) {
	rootFlag := ""
	if rootful {
		rootFlag = "--root "
	}

	progress.Finalize("Otter '%s' successfully created.", containerName)
	ui.DefaultLogger.Info("To enter, run: otter enter %s%s", rootFlag, containerName)
}

func printContainerAlreadyExists(progress *ui.Progress, containerName string, rootful bool) {
	rootFlag := ""
	if rootful {
		rootFlag = "--root "
	}

	progress.Finalize("Container named '%s' already exists.", containerName)
	ui.DefaultLogger.Info("To enter, run: otter enter %s%s", rootFlag, containerName)
}
