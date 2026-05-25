package commands

import (
	"bufio"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ferret-linux/otter/pkg/config"
	"github.com/ferret-linux/otter/pkg/internal/testutil"
	"github.com/ferret-linux/otter/pkg/ui"
)

func newInternalTestEphemeralCommand(mock *testutil.MockContainerManager) *EphemeralCommand {
	progress := ui.NewDevNullProgress()
	prompter := ui.NewPrompter(*bufio.NewReader(strings.NewReader("")), io.Discard)
	return NewEphemeralCommand(&config.Values{}, mock, progress, prompter)
}

func TestEphemeralCommand_MakeUniqueRandomName_FirstAttemptUnique(t *testing.T) {
	mock := &testutil.MockContainerManager{}
	cmd := newInternalTestEphemeralCommand(mock)

	name, err := cmd.makeUniqueRandomName(context.Background(), false)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(name, "otter-"), "expected name to have otter- prefix, got %q", name)
	require.Len(t, mock.Spy.Exists, 1)
}

func TestEphemeralCommand_MakeUniqueRandomName_RetriesAfterCollisions(t *testing.T) {
	const collisions = 3

	mock := &testutil.MockContainerManager{}
	calls := 0
	mock.ExistsFn = func(_ string) bool {
		calls++
		return calls <= collisions
	}
	cmd := newInternalTestEphemeralCommand(mock)

	name, err := cmd.makeUniqueRandomName(context.Background(), false)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(name, "otter-"), "expected name to have otter- prefix, got %q", name)
	require.Len(t, mock.Spy.Exists, collisions+1)
}

func TestEphemeralCommand_MakeUniqueRandomName_AllCollideReturnsError(t *testing.T) {
	mock := &testutil.MockContainerManager{}
	mock.ExistsFn = func(_ string) bool {
		return true
	}
	cmd := newInternalTestEphemeralCommand(mock)

	name, err := cmd.makeUniqueRandomName(context.Background(), false)
	require.Error(t, err)
	assert.Empty(t, name)
	assert.Contains(t, err.Error(), "failed to generate unique ephemeral container name")
	require.Len(t, mock.Spy.Exists, ephemeralMaxNameGenAttempts)
}

func TestEphemeralCommand_MakeUniqueRandomName_DryRunSkipsExistenceCheck(t *testing.T) {
	mock := &testutil.MockContainerManager{}
	mock.ExistsFn = func(_ string) bool {
		return true
	}
	cmd := newInternalTestEphemeralCommand(mock)

	name, err := cmd.makeUniqueRandomName(context.Background(), true)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(name, "otter-"), "expected name to have otter- prefix, got %q", name)
	assert.Empty(t, mock.Spy.Exists, "Exists must not be called in dry-run mode")
}
