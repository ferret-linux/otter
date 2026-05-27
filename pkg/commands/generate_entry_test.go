package commands_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ferret-linux/otter/pkg/commands"
	"github.com/ferret-linux/otter/pkg/config"
	"github.com/ferret-linux/otter/pkg/internal/testutil"
)

func TestGenerateEntryCommand_Execute(t *testing.T) {
	ctx := context.Background()

	tempDir := t.TempDir()
	defer os.RemoveAll(tempDir)

	// create the list command
	containerManager := &testutil.MockContainerManager{}
	listCmd := commands.NewListCommand(&config.Values{}, containerManager)

	//
	// Generate the entry
	//

	generateEntryCmd := commands.NewGenerateEntryCommand(&config.Values{}, listCmd, containerManager)

	opts := &commands.GenerateEntryOptions{
		ContainerName:       "test-container",
		Verbose:             true,
		Delete:              false,
		Icon:                "https://github.com/ferret-project/otter/raw/refs/heads/main/icons/terminal-otter-icon.svg",
		Root:                false,
		DesktopEntryBaseDir: fmt.Sprintf("%s/.local/share/", tempDir),
		OtterPath:           "/usr/bin/otter",
	}

	err := generateEntryCmd.Execute(ctx, opts)
	require.NoError(t, err, "GenerateEntryCommand.Execute()")

	expectedEntryPath := fmt.Sprintf("%s/.local/share/applications/test-container.desktop", tempDir)
	assert.FileExists(t, expectedEntryPath)

	expectedContent := `[Desktop Entry]
Name=Test-container
GenericName=Terminal entering Test-container
Comment=Terminal entering Test-container
Categories=Otter;System;Utility
Exec=/usr/bin/otter enter test-container
Icon=https://github.com/ferret-project/otter/raw/refs/heads/main/icons/terminal-otter-icon.svg
Keywords=otter;
NoDisplay=false
Terminal=true
TryExec=/usr/bin/otter
Type=Application
Actions=Remove;

[Desktop Action Remove]
Name=Remove Test-container from system
Exec=/usr/bin/otter rm test-container
`

	content, err := os.ReadFile(expectedEntryPath)
	require.NoError(t, err, "Failed to read desktop entry file")
	assert.Equal(t, expectedContent, string(content), "Desktop entry content mismatch")

	// Delete the entry
	opts.Delete = true
	err = generateEntryCmd.Execute(ctx, opts)
	require.NoError(t, err, "GenerateEntryCommand.Execute() on delete")

	assert.NoFileExists(t, expectedEntryPath)

	// Try deleting a non-existing entry
	err = generateEntryCmd.Execute(ctx, opts)
	assert.NoError(t, err, "GenerateEntryCommand.Execute() on delete non-existing")
}

func TestGenerateEntryCommand_Execute_Root(t *testing.T) {
	ctx := context.Background()

	tempDir := t.TempDir()
	defer os.RemoveAll(tempDir)

	containerManager := &testutil.MockContainerManager{}
	listCmd := commands.NewListCommand(&config.Values{}, containerManager)

	generateEntryCmd := commands.NewGenerateEntryCommand(&config.Values{}, listCmd, containerManager)

	opts := &commands.GenerateEntryOptions{
		ContainerName:       "test-container",
		Verbose:             true,
		Delete:              false,
		Icon:                "https://github.com/ferret-project/otter/raw/refs/heads/main/icons/terminal-otter-icon.svg",
		Root:                true,
		DesktopEntryBaseDir: fmt.Sprintf("%s/.local/share/", tempDir),
		OtterPath:           "/usr/bin/otter",
	}

	err := generateEntryCmd.Execute(ctx, opts)
	require.NoError(t, err, "GenerateEntryCommand.Execute()")

	expectedEntryPath := fmt.Sprintf("%s/.local/share/applications/test-container.desktop", tempDir)
	assert.FileExists(t, expectedEntryPath)

	expectedContent := `[Desktop Entry]
Name=Test-container (rootful)
GenericName=Terminal entering Test-container
Comment=Terminal entering Test-container
Categories=Otter;System;Utility
Exec=/usr/bin/otter enter --root test-container
Icon=https://github.com/ferret-project/otter/raw/refs/heads/main/icons/terminal-otter-icon.svg
Keywords=otter;
NoDisplay=false
Terminal=true
TryExec=/usr/bin/otter
Type=Application
Actions=Remove;

[Desktop Action Remove]
Name=Remove Test-container from system
Exec=/usr/bin/otter rm --root test-container
`

	content, err := os.ReadFile(expectedEntryPath)
	require.NoError(t, err, "Failed to read desktop entry file")
	assert.Equal(t, expectedContent, string(content), "Desktop entry content mismatch")
}

func TestGenerateAllEntriesCommand_Execute(t *testing.T) {
	ctx := context.Background()

	// tempDir is the test directory where we expect to create desktop entries
	tempDir := t.TempDir()
	defer os.RemoveAll(tempDir)

	// create the list command
	containerManager := &testutil.MockContainerManager{}
	listCmd := commands.NewListCommand(&config.Values{}, containerManager)

	// create the generate all entries command
	genAllEntriesCmd := commands.NewGenerateEntryCommand(&config.Values{}, listCmd, containerManager)

	//
	// Generate the entries
	//

	opts := &commands.GenerateEntryOptions{
		All:                 true,
		Verbose:             false,
		Delete:              false,
		DesktopEntryBaseDir: fmt.Sprintf("%s/.local/share/", tempDir),
		OtterPath:           "/usr/bin/otter",
	}
	err := genAllEntriesCmd.Execute(ctx, opts)
	require.NoError(t, err, "GenerateAllEntriesCommand.Execute()")

	// retrieve the list of containers to verify entries were created
	listResult, err := listCmd.Execute(ctx)
	require.NoError(t, err, "ListCommand.Execute()")

	// verify that each container has a corresponding desktop entry
	for _, container := range listResult.Containers {
		expectedEntryPath := fmt.Sprintf("%s/.local/share/applications/%s.desktop", tempDir, container.Name)
		assert.FileExists(t, expectedEntryPath)
	}

	//
	// Delete the entries
	//

	opts.Delete = true
	err = genAllEntriesCmd.Execute(ctx, opts)
	require.NoError(t, err, "GenerateAllEntriesCommand.Execute() on delete")

	// verify that each container's desktop entry has been deleted
	for _, container := range listResult.Containers {
		expectedEntryPath := fmt.Sprintf("%s/.local/share/applications/%s.desktop", tempDir, container.Name)
		assert.NoFileExists(t, expectedEntryPath)
	}
}
