// Package tui implements otter's interactive terminal UI, launched via
// `otter tui`. See internal/cli/tui.go for the command wiring.
package tui

import (
	"context"
	"fmt"

	tea "charm.land/bubbletea/v2"

	"github.com/ferret-linux/otter/pkg/containermanager"
)

// Run starts the otter TUI program. It blocks until the user quits.
func Run(ctx context.Context, cm containermanager.ContainerManager) error {
	p := tea.NewProgram(NewModel(ctx, cm))
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("tui program exited with error: %w", err)
	}
	return nil
}
