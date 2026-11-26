package cli

import (
	"context"
	"errors"
	"fmt"
	"os/exec"

	"github.com/urfave/cli/v3"

	"github.com/ferret-linux/otter/pkg/commands"
	"github.com/ferret-linux/otter/pkg/config"
	"github.com/ferret-linux/otter/pkg/rootcheck"
)

const (
	cleanupFlagJSON     = "json"
	cleanupFlagScanPath = "scan-path"
)

func newCleanupCommand(_ *config.Values) *cli.Command {
	return &cli.Command{
		Name:    "cleanup",
		Aliases: []string{"purge"},
		Usage:   "Remove orphaned otter state from exports, launchers, icons and images",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "no-apply",
				Usage: "Print the cleanup plan without writing or taking locks",
			},
			&cli.BoolFlag{
				Name:  cleanupFlagJSON,
				Usage: "Show the plan or report as JSON",
			},
			&cli.BoolFlag{
				Name:  "all-managers",
				Usage: "Include every installed podman, docker and nerdctl manager",
			},
			&cli.BoolFlag{
				Name:  "images",
				Usage: "Allow deletion of safe unreferenced otter-derived engine images",
			},
			&cli.StringSliceFlag{
				Name:  cleanupFlagScanPath,
				Usage: "Additional file or directory to scan for export markers (repeatable)",
			},
		},
		Action: cleanupAction,
	}
}

func cleanupAction(ctx context.Context, cmd *cli.Command) error {
	opts := commands.CleanupOptions{
		Apply:       !cmd.Bool("no-apply"),
		JSON:        cmd.Bool(cleanupFlagJSON),
		AllManagers: cmd.Bool("all-managers"),
		Images:      cmd.Bool("images"),
		Root:        cmd.Bool("root"),
		ScanPaths:   cmd.StringSlice(cleanupFlagScanPath),
		ManagerType: cmd.Root().String("container-manager"),
		SudoCommand: cmd.Root().String("sudo-command"),
	}

	//nolint:nestif // escalation branches are intentionally explicit for plan vs apply
	if opts.Root {
		if opts.Apply {
			resolved, err := rootcheck.Validate(ctx, opts.SudoCommand)
			if err != nil {
				return fmt.Errorf("cannot run in root mode: %w", err)
			}
			if err := cmd.Root().Set("sudo-command", resolved); err != nil {
				return fmt.Errorf("failed to apply resolved escalator: %w", err)
			}
			ctx = withEscalator(ctx, resolved)
			opts.SudoCommand = resolved
		} else {
			resolved, err := resolveEscalatorForPlan(opts.SudoCommand)
			if err != nil {
				return fmt.Errorf("cannot plan rootful cleanup: %w", err)
			}
			opts.SudoCommand = resolved
		}
	}

	result, err := commands.NewCleanupCommand().Execute(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to execute cleanup: %w", err)
	}
	if err := commands.PrintCleanupResult(result, opts.JSON); err != nil {
		return fmt.Errorf("failed to print cleanup report: %w", err)
	}
	if result.HasHardProblems() {
		return errors.New("cleanup finished with problems")
	}
	return nil
}

// resolveEscalatorForPlan finds an escalator for read-only rootful engine
// queries without running rootcheck.Validate (which probes privileges).
func resolveEscalatorForPlan(sudoCommand string) (string, error) {
	if sudoCommand != "" && sudoCommand != autodetectEscalator {
		return sudoCommand, nil
	}
	for _, name := range []string{"sudo", "sudo-rs", "doas", "run0", "pkexec"} {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}
	return "", errors.New("no privilege escalation program found in PATH")
}
