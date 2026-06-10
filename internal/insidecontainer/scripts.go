package insidecontainer

import (
	"crypto/sha256"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed assets/otter-host-exec
var hostExecScript string

//go:embed assets/otter-init
var initScript string

//go:embed assets/otter-export
var exportScripts string

//go:embed assets/otter
var otterScript string

// ProvisionScripts ensures that all necessary scripts are created in the host directory.
// It returns the path to the directory where the scripts are stored, and whether any scripts were updated.
func ProvisionScripts() (string, bool, error) {
	dir := hostDir()
	//nolint:gosec // 0755 is correct for directories: executable bit grants traversal permission to all users
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", false, fmt.Errorf("failed to create scripts directory: %w", err)
	}

	scripts := []struct {
		name    string
		content string
	}{
		{"otter-host-exec", hostExecScript},
		{"otter-init", initScript},
		{"otter-export", exportScripts},
		{"otter", otterScript},
	}

	updated := false
	for _, script := range scripts {
		destFilePath := filepath.Join(dir, script.name)
		newHash := sha256.Sum256([]byte(script.content))
		if existing, err := os.ReadFile(destFilePath); err == nil {
			if sha256.Sum256(existing) == newHash {
				continue
			}
		}
		//nolint:gosec // 0755 is required: scripts must be executable by the container runtime
		if err := os.WriteFile(destFilePath, []byte(script.content), 0755); err != nil {
			return "", false, fmt.Errorf("failed to write script %s: %w", script.name, err)
		}
		updated = true
	}

	return dir, updated, nil
}

// hostDir returns the directory path where the scripts should be stored.
// Evaluates OTR_SCRIPTS_DIR env var first, then HOME env var, and falls back to default path.
func hostDir() string {
	// First check OTR_SCRIPTS_DIR env var
	if dir := os.Getenv("OTR_SCRIPTS_DIR"); dir != "" {
		return dir
	}

	// Then, check HOME env var
	if home := os.Getenv("HOME"); home != "" {
		return filepath.Join(home, ".local", "share", "otter")
	}

	// Fallback to default path
	return "/var/lib/otter"
}
