package cli

import (
	"context"
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/urfave/cli/v3"

	"github.com/ferret-linux/otter/pkg/commands"
	"github.com/ferret-linux/otter/pkg/config"
)

func newDocumentationCommand(_ *config.Values) *cli.Command {
	return &cli.Command{
		Name:    "documentation",
		Aliases: []string{"docs"},
		Action:  documentationAction,
	}
}

func documentationAction(_ context.Context, _ *cli.Command) error {
	m, err := commands.NewDocumentationModel()
	if err != nil {
		return fmt.Errorf("failed to start docs viewer: %w", err)
	}

	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("docs viewer exited with error: %w", err)
	}
	return nil
}
