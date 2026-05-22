package commands_test

import (
	"bufio"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ferret-linux/otter/pkg/commands"
	"github.com/ferret-linux/otter/pkg/config"
	"github.com/ferret-linux/otter/pkg/internal/testutil"
	"github.com/ferret-linux/otter/pkg/ui"
)

func newTestEphemeralCommand(mock *testutil.MockContainerManager) *commands.EphemeralCommand {
	progress := ui.NewDevNullProgress()
	prompter := ui.NewPrompter(*bufio.NewReader(strings.NewReader("")), io.Discard)
	return commands.NewEphemeralCommand(&config.Values{}, mock, progress, prompter)
}

func TestEphemeralCommand_PassesCustomCommandToEnter(t *testing.T) {
	mock := &testutil.MockContainerManager{}
	cmd := newTestEphemeralCommand(mock)

	customCommand := []string{"cat", "/etc/os-release"}
	err := cmd.Execute(context.Background(), commands.EphemeralOptions{
		CreateOptions: commands.CreateOptions{
			ContainerName:  "ephemeral-test",
			ContainerImage: "alpine:latest",
		},
		CustomCommand: customCommand,
		DryRun:        true,
	})
	require.NoError(t, err)

	require.Len(t, mock.Spy.Enter, 1, "expected Enter to be called exactly once")
	opts := getEnterOptions(mock.Spy, 0)
	assert.Equal(t, "ephemeral-test", opts.ContainerName)
	assert.Equal(t, customCommand, opts.CustomCommand)
}

func TestEphemeralCommand_EmptyCustomCommandIsNotForwardedAsArgs(t *testing.T) {
	mock := &testutil.MockContainerManager{}
	cmd := newTestEphemeralCommand(mock)

	err := cmd.Execute(context.Background(), commands.EphemeralOptions{
		CreateOptions: commands.CreateOptions{
			ContainerName:  "ephemeral-no-cmd",
			ContainerImage: "alpine:latest",
		},
		DryRun: true,
	})
	require.NoError(t, err)

	require.Len(t, mock.Spy.Enter, 1, "expected Enter to be called exactly once")
	opts := getEnterOptions(mock.Spy, 0)
	assert.Equal(t, "ephemeral-no-cmd", opts.ContainerName)
	assert.Empty(t, opts.CustomCommand, "expected no custom command to be forwarded")
}
