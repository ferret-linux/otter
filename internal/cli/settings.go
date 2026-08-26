package cli

import (
	"context"
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/urfave/cli/v3"

	"github.com/ferret-linux/otter/pkg/commands"
	"github.com/ferret-linux/otter/pkg/config"
)

func newSettingsCommand(_ *config.Values) *cli.Command {
	return &cli.Command{
		Name:    "settings",
		Aliases: []string{"conf"},
		Action:  settingsAction,
	}
}

func settingsAction(_ context.Context, _ *cli.Command) error {
	m, err := commands.NewSettingsModel()
	if err != nil {
		return fmt.Errorf("failed to start settings editor: %w", err)
	}

	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("settings editor exited with error: %w", err)
	}
	return nil
}
