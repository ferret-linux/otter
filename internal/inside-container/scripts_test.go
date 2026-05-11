package insideContainer_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	insideContainer "github.com/ferret-linux/otter/internal/inside-container"
)

func TestProvisionScripts_CustomDir(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("OTR_SCRIPTS_DIR", tmpDir)

	scriptsDir, err := insideContainer.ProvisionScripts()
	require.NoError(t, err, "ProvisionScripts failed")
	defer os.RemoveAll(scriptsDir)

	require.Equal(t, tmpDir, scriptsDir)

	expectedScripts := []string{
		"otter-host-exec",
		"otter-init",
		"otter-export",
	}

	for _, scriptName := range expectedScripts {
		scriptPath := filepath.Join(scriptsDir, scriptName)
		assert.FileExists(t, scriptPath, "Expected script %s to exist", scriptName)
	}
}

func TestProvisionScripts_HomeDir(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	scriptsDir, err := insideContainer.ProvisionScripts()
	require.NoError(t, err, "ProvisionScripts failed")
	defer os.RemoveAll(scriptsDir)

	expected := filepath.Join(tmpDir, ".local", "share", "otter", "v2")
	require.Equal(t, expected, scriptsDir)

	expectedScripts := []string{
		"otter-host-exec",
		"otter-init",
		"otter-export",
	}

	for _, scriptName := range expectedScripts {
		scriptPath := filepath.Join(scriptsDir, scriptName)
		assert.FileExists(t, scriptPath, "Expected script %s to exist", scriptName)
	}
}
