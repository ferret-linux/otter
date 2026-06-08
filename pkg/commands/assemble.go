package commands

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/ferret-linux/otter/pkg/config"
	"github.com/ferret-linux/otter/pkg/containermanager"
	"github.com/ferret-linux/otter/pkg/manifest"
	"github.com/ferret-linux/otter/pkg/ui"
)

const assembleCleanupTimeout = 30 * time.Second

type AssembleOptions struct {
	Items []manifest.Item
	// Boxname is the name of the box to assemble
	// If specified, the Assemble command will only assemble the given box
	// If empty, the command will assemble all boxes defined in the manifest
	Boxname string
	// Delete indicates whether to delete the existing box before assembling
	// true=delete, false=create or update
	Delete  bool
	Replace bool
}

type AssembleCommand struct {
	createCmd     *CreateCommand
	rmCmd         *RmCommand
	lockCmd       *LockCommand
	startCmd      *StartCommand
	enterCmd      *EnterCommand
	createCmdRoot *CreateCommand
	rmCmdRoot     *RmCommand
	lockCmdRoot   *LockCommand
	startCmdRoot  *StartCommand
	enterCmdRoot  *EnterCommand
	progress      *ui.Progress
}

func NewAssembleCommand(
	cfg *config.Values,
	cm containermanager.ContainerManager,
	prompter *ui.Prompter,
	progress *ui.Progress,
) *AssembleCommand {
	cmRoot := cm.CloneAsRoot()
	return &AssembleCommand{
		createCmd:     NewCreateCommand(cfg, cm, ui.NewDevNullProgress()),
		rmCmd:         NewRmCommand(cm, prompter),
		lockCmd:       NewLockCommand(cm),
		startCmd:      NewStartCommand(cm),
		enterCmd:      NewEnterCommand(cm),
		createCmdRoot: NewCreateCommand(cfg, cmRoot, ui.NewDevNullProgress()),
		rmCmdRoot:     NewRmCommand(cmRoot, prompter),
		lockCmdRoot:   NewLockCommand(cmRoot),
		startCmdRoot:  NewStartCommand(cmRoot),
		enterCmdRoot:  NewEnterCommand(cmRoot),
		progress:      progress,
	}
}

func (ac *AssembleCommand) Execute(ctx context.Context, opts AssembleOptions) error {
	var items []manifest.Item
	if opts.Boxname != "" {
		idx := slices.IndexFunc(opts.Items, func(i manifest.Item) bool {
			return i.Name == opts.Boxname
		})
		if idx == -1 {
			return fmt.Errorf("box '%s' not found in manifest", opts.Boxname)
		}
		items = []manifest.Item{opts.Items[idx]}
	} else {
		items = opts.Items
	}

	for _, item := range items {
		switch {
		case opts.Delete:
			if err := ac.deleteItem(ctx, item); err != nil {
				return fmt.Errorf("failed to delete item '%s': %w", item.Name, err)
			}
		case opts.Replace:
			if err := ac.replaceItem(ctx, item); err != nil {
				return fmt.Errorf("failed to replace item '%s': %w", item.Name, err)
			}
		default:
			if err := ac.createItem(ctx, item); err != nil {
				return fmt.Errorf("failed to create item '%s': %w", item.Name, err)
			}
		}
	}

	return nil
}

func (ac *AssembleCommand) deleteItem(ctx context.Context, item manifest.Item) error {
	ac.progress.Next("Deleting %s...", item.Name)
	opts := RmOptions{
		NoTTY:          true,
		Force:          true,
		All:            false,
		ContainerNames: []string{item.Name},
	}

	rmCmd := ac.rmCmd
	if item.Root {
		rmCmd = ac.rmCmdRoot
	}

	_, err := rmCmd.Execute(ctx, opts)
	if err != nil {
		ac.progress.Fail()
		return fmt.Errorf("failed to execute delete item '%s': %w", item.Name, err)
	}
	ac.progress.Done()
	return nil
}

func (ac *AssembleCommand) replaceItem(ctx context.Context, item manifest.Item) error {
	err := ac.deleteItem(ctx, item)
	if err != nil {
		return err
	}

	return ac.createItem(ctx, item)
}

func (ac *AssembleCommand) createItem(ctx context.Context, item manifest.Item) error {
	ac.progress.Next("Creating %s...", item.Name)
	opts := CreateOptions{
		ContainerClone:          item.Clone,
		ContainerName:           item.Name,
		ContainerImage:          item.Image,
		ContainerHostname:       item.Hostname,
		UnshareNetNs:            item.UnshareNetns || item.UnshareAll,
		UnshareDevsys:           item.UnshareDevsys || item.UnshareAll,
		UnshareGroups:           item.UnshareGroups || item.UnshareAll || item.Init,
		UnshareIpc:              item.UnshareIPC || item.UnshareAll,
		UnshareProcess:          item.UnshareProcess || item.UnshareAll || item.Init,
		AdditionalFlags:         item.AdditionalFlags,
		AdditionalVolumes:       item.Volumes,
		AdditionalPackages:      item.AdditionalPackages,
		ContainerPreInitHook:    ac.joinHooks(item.PreInitHooks),
		ContainerInitHook:       ac.joinHooks(item.InitHooks),
		ContainerUserCustomHome: item.Home,
		Init:                    item.Init,
		Nvidia:                  item.Nvidia,
		Memory:                  item.Memory,
		CPUThreads:              item.CPUThreads,
		GenerateEntry:           item.Entry,
		Rootful:                 item.Root,
		NonInteractive:          true,
		ContainerAlwaysPull:     item.AlwaysPull,
		ContainerShell:          item.UserShell,
	}

	createCmd := ac.createCmd
	if item.Root {
		createCmd = ac.createCmdRoot
	}
	_, err := createCmd.Execute(ctx, opts)
	if err != nil {
		ac.progress.Fail()
		return err
	}

	success := false
	defer func() {
		if success {
			return
		}
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), assembleCleanupTimeout)
		defer cancel()
		rmCmd := ac.rmCmd
		if item.Root {
			rmCmd = ac.rmCmdRoot
		}
		if _, rmErr := rmCmd.Execute(cleanupCtx, RmOptions{
			NoTTY:          true,
			Force:          true,
			Root:           item.Root,
			ContainerNames: []string{item.Name},
		}); rmErr != nil {
			ui.DefaultLogger.Warn("%s: %s", item.Name, rmErr)
		}
	}()

	err = ac.setupBox(ctx, item)
	if err != nil {
		ac.progress.Fail()
		return err
	}

	if item.Lock {
		lockCmd := ac.lockCmd
		if item.Root {
			lockCmd = ac.lockCmdRoot
		}
		if err := lockCmd.Execute(ctx, LockOptions{ContainerNames: []string{item.Name}}); err != nil {
			ac.progress.Fail()
			return err
		}
	}

	ac.progress.Done()
	success = true
	return nil
}

func (ac *AssembleCommand) joinHooks(hooks []string) string {
	sb := strings.Builder{}

	for i, hook := range hooks {
		sb.WriteString(hook)

		if i < len(hooks)-1 {
			semicolonRegex := regexp.MustCompile(`;[[:space:]]{0,1}$`)
			andAndRegex := regexp.MustCompile(`&&[[:space:]]{0,1}$`)

			separator := "  " // two spaces just because v1 does that, so it's comparable in regression tests
			if !semicolonRegex.MatchString(hook) && !andAndRegex.MatchString(hook) {
				separator = " && "
			}

			sb.WriteString(separator)
		}
	}

	return sb.String()
}

func (ac *AssembleCommand) setupBox(ctx context.Context, item manifest.Item) error {
	startCmd := ac.startCmd
	enterCmd := ac.enterCmd
	if item.Root {
		startCmd = ac.startCmdRoot
		enterCmd = ac.enterCmdRoot
	}

	if err := startCmd.Execute(ctx, &StartOptions{ContainerNames: []string{item.Name}}); err != nil {
		return err
	}
	if item.StartNow {
		_, err := enterCmd.Execute(ctx, EnterOptions{
			ContainerName: item.Name,
			NoTTY:         true,
			CustomCommand: []string{"true"}, // we just want to run the init hooks, so we can skip the shell
		})
		if err != nil {
			return fmt.Errorf("failed to execute init hooks for item '%s': %w", item.Name, err)
		}
	}

	// validate app name to prevent command injection, since it's used in a custom command
	var validAppName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._+\-]*$`)
	for _, app := range item.ExportedApps {
		if !validAppName.MatchString(app) {
			return fmt.Errorf("invalid app name '%s' for item '%s': must be alphanumeric (with dots, underscores, hyphens)", app, item.Name)
		}
	}
	for _, app := range item.ExportedApps {
		_, err := enterCmd.Execute(ctx, EnterOptions{
			ContainerName: item.Name,
			NoTTY:         true,
			CustomCommand: []string{"otter-export", "--app", app},
		})
		if err != nil {
			return fmt.Errorf("failed to export app '%s' for item '%s': %w", app, item.Name, err)
		}
	}

	// validate bin path to prevent command injection, since it's used in a custom command
	var validBinPath = regexp.MustCompile(`^/[a-zA-Z0-9._+\-/]+$`)
	if len(item.ExportedBins) > 0 && !validBinPath.MatchString(item.ExportedBinsPath) {
		return fmt.Errorf("invalid exported bins path '%s' for item '%s': must be an absolute path with alphanumeric characters, dots, underscores, or hyphens", item.ExportedBinsPath, item.Name)
	}
	// we allow slashes in bin paths, but we validate each path segment to prevent command injection
	for _, bin := range item.ExportedBins {
		if !validBinPath.MatchString(bin) {
			return fmt.Errorf("invalid bin path '%s' for item '%s': must be an absolute path with alphanumeric characters, dots, underscores, or hyphens", bin, item.Name)
		}
	}
	for _, bin := range item.ExportedBins {
		_, err := enterCmd.Execute(ctx, EnterOptions{
			ContainerName: item.Name,
			NoTTY:         true,
			CustomCommand: []string{"otter-export", "--bin", bin, "--export-path", item.ExportedBinsPath},
		})
		if err != nil {
			return fmt.Errorf("failed to export bin '%s' for item '%s': %w", bin, item.Name, err)
		}
	}

	return nil
}
