package commands

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/ferret-linux/otter/pkg/config"
	"github.com/ferret-linux/otter/pkg/containermanager"
	"github.com/ferret-linux/otter/pkg/ui"
)

const (
	ephemeralCleanupTimeout     = 30 * time.Second
	ephemeralMaxNameGenAttempts = 10
)

type EphemeralOptions struct {
	CreateOptions

	CustomCommand []string
}

type EphemeralCommand struct {
	containerManager containermanager.ContainerManager
	createCmd        *CreateCommand
	startCmd         *StartCommand
	enterCmd         *EnterCommand
	rmCmd            *RmCommand
}

func NewEphemeralCommand(
	cfg *config.Values,
	cm containermanager.ContainerManager,
	progress *ui.Progress,
	prompter *ui.Prompter,
) *EphemeralCommand {
	return &EphemeralCommand{
		containerManager: cm,
		createCmd:        NewCreateCommand(cfg, cm, progress, prompter),
		startCmd:         NewStartCommand(cm),
		enterCmd:         NewEnterCommand(cm),
		rmCmd:            NewRmCommand(cm, prompter),
	}
}

func (c *EphemeralCommand) Execute(ctx context.Context, opts EphemeralOptions) error {
	name := opts.ContainerName
	if name == "" {
		generatedName, err := c.makeUniqueRandomName(ctx)
		if err != nil {
			return fmt.Errorf("ephemeral: %w", err)
		}
		name = generatedName
	}

	// create ephemeral container
	createOpts := opts.CreateOptions
	createOpts.ContainerName = name
	// override options not relevant for creating ephemeral containers
	createOpts.GenerateEntry = false
	createOpts.NonInteractive = true
	if _, createErr := c.createCmd.Execute(ctx, createOpts); createErr != nil {
		return fmt.Errorf("ephemeral: %w", createErr)
	}

	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), ephemeralCleanupTimeout)
		defer cancel()
		rmOpts := RmOptions{
			ContainerNames: []string{name},
			Force:          true,
			NoTTY:          true,
		}
		if _, rmErr := c.rmCmd.Execute(cleanupCtx, rmOpts); rmErr != nil {
			ui.DefaultLogger.Warn("%s: %s", name, rmErr)
		}
	}()

	if err := c.startCmd.Execute(ctx, &StartOptions{
		ContainerNames: []string{name},
	}); err != nil {
		return fmt.Errorf("ephemeral: %w", err)
	}

	// enter into it
	enterOpts := EnterOptions{
		ContainerName: name,
		CustomCommand: opts.CustomCommand,
	}
	if _, enterErr := c.enterCmd.Execute(ctx, enterOpts); enterErr != nil {
		return fmt.Errorf("ephemeral: %w", enterErr)
	}

	return nil
}

func (c *EphemeralCommand) makeUniqueRandomName(ctx context.Context) (string, error) {
	for range ephemeralMaxNameGenAttempts {
		name := makeRandomName()
		if !c.containerManager.Exists(ctx, name) {
			return name, nil
		}
	}
	return "", fmt.Errorf("failed to generate unique ephemeral container name after %d attempts", ephemeralMaxNameGenAttempts)
}

func makeRandomName() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	l := len(charset)
	b := make([]byte, 3) //nolint:mnd // length of random part
	for i := range b {
		b[i] = charset[rand.IntN(l)] //nolint:gosec // cryptographic security not needed
	}
	return fmt.Sprintf("otter-%s", string(b))
}
