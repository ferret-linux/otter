package cli

import (
	"bufio"
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
		Name: "create",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "image",
				Aliases: []string{"i"},
				Value:   cfg.DefaultContainerImage,
			},
			&cli.StringFlag{
				Name:    "name",
				Aliases: []string{"n"},
			},
			&cli.StringFlag{
				Name: "hostname",
			},
			&cli.BoolFlag{
				Name:    "pull",
				Aliases: []string{"p"},
			},
			&cli.BoolFlag{
				Name:    "yes",
				Aliases: []string{"Y"},
			},
			&cli.StringFlag{
				Name:    "clone",
				Aliases: []string{"c"},
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
				Name:    "dry-run",
				Aliases: []string{"d"},
			},
			&cli.BoolFlag{
				Name:    "verbose",
				Aliases: []string{"v"},
			},
			&cli.BoolFlag{
				Name: "absolutely-disable-root-password-i-am-really-positively-sure",
			},
			&cli.BoolFlag{
				Name:    "compatibility",
				Aliases: []string{"C"},
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return createAction(ctx, cmd, cfg)
		},
	}
}

func createAction(ctx context.Context, cmd *cli.Command, cfg *config.Values) error {
	if cmd.Bool("compatibility") {
		err := showCompatibility()
		if err != nil {
			return fmt.Errorf("compatibility check failed: %w", err)
		}
		return nil
	}

	containerManager, ok := ctx.Value(containerManagerKey).(containermanager.ContainerManager)
	if !ok {
		return errors.New("container manager not found in context")
	}

	opts := commands.CreateOptions{
		ContainerImage:          cmd.String("image"),
		ContainerName:           cmd.String("name"),
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
		ContainerInitHook:       cmd.String("init-hooks"),
		ContainerPreInitHook:    cmd.String("pre-init-hooks"),
		ContainerPlatform:       cmd.String("platform"),
		DryRun:                  cmd.Bool("dry-run"),
		GenerateEntry:           !cmd.Bool("no-entry"),
		Rootful:                 cmd.Bool("root"),
		ContainerAlwaysPull:     cmd.Bool("pull"),
		NonInteractive:          cmd.Bool("yes"),
	}

	progress := ui.NewProgress(os.Stderr)
	prompter := ui.NewPrompter(*bufio.NewReader(os.Stdin), os.Stdout)

	createCmd := commands.NewCreateCommand(cfg, containerManager, progress, prompter)
	result, err := createCmd.Execute(ctx, opts)

	var containerAlreadyExistsErr *commands.ContainerAlreadyExistsError
	if errors.As(err, &containerAlreadyExistsErr) {
		printContainerAlreadyExists(progress, containerAlreadyExistsErr.ContainerName, opts.Rootful)
	}

	if errors.Is(err, commands.ErrImagePullAbortedByUser) {
		progress.Finalize("next time, pull the image first")
		return nil
	}

	if err != nil {
		return fmt.Errorf("create command failed: %w", err)
	}

	if !opts.DryRun {
		printCreateCompleted(progress, result.ContainerName, opts.Rootful)
	}

	return nil
}

func showCompatibility() error {
	printFile("image_compatibility")
	return nil
}

func printCreateCompleted(progress *ui.Progress, containerName string, rootful bool) {
	rootFlag := ""
	if rootful {
		rootFlag = "--root "
	}

	msg := "Otter '%s' successfully created.\nTo enter, run:\n\notter enter %s%s\n\n"

	progress.Finalize(msg, containerName, rootFlag, containerName)
}

func printContainerAlreadyExists(progress *ui.Progress, containerName string, rootful bool) {
	rootFlag := ""
	if rootful {
		rootFlag = "--root "
	}

	msg := `Container named '%s' already exists.
To enter, run:

otter enter %s%s

`

	progress.Finalize(msg, containerName, rootFlag, containerName)
}
