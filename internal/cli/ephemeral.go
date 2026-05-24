package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"slices"

	"github.com/urfave/cli/v3"

	"github.com/ferret-linux/otter/pkg/commands"
	"github.com/ferret-linux/otter/pkg/config"
	"github.com/ferret-linux/otter/pkg/containermanager"
	"github.com/ferret-linux/otter/pkg/ui"
)

func newEphemeralCommand(cfg *config.Values) *cli.Command {
	createCmd := newCreateCommand(cfg)

	ignoredFlags := []string{
		"compatibility",
		"no-entry",
	}
	flags := make([]cli.Flag, 0, len(createCmd.Flags))
	for _, f := range createCmd.Flags {
		if slices.Contains(ignoredFlags, f.Names()[0]) {
			continue
		}
		flags = append(flags, f)
	}

	return &cli.Command{
		Name:    "ephemeral",
		Aliases: []string{"eph"},
		Flags:   flags,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return ephemeralAction(ctx, cmd, cfg)
		},
	}
}

func ephemeralAction(ctx context.Context, cmd *cli.Command, cfg *config.Values) error {
	containerManager, ok := ctx.Value(containerManagerKey).(containermanager.ContainerManager)
	if !ok {
		return errors.New("container manager not found in context")
	}
	opts := commands.EphemeralOptions{
		CreateOptions: commands.CreateOptions{
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
			GenerateEntry:           false,
			Rootful:                 cmd.Bool("root"),
		},
		DryRun:        cmd.Bool("dry-run"),
		Verbose:       cmd.Bool("verbose"),
		CustomCommand: cmd.Args().Slice(),
	}

	progress := ui.NewProgress(os.Stderr)
	prompter := ui.NewPrompter(*bufio.NewReader(os.Stdin), os.Stdout)

	ephemeralCmd := commands.NewEphemeralCommand(cfg, containerManager, progress, prompter)

	err := ephemeralCmd.Execute(ctx, opts)
	if err != nil {
		return fmt.Errorf("ephemeral command failed: %w", err)
	}

	return nil
}
