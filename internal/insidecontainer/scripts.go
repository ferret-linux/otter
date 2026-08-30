package insidecontainer

import (
	"crypto/sha256"
	"embed"
	"errors"
	"fmt"
	"io/fs"
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

//go:embed assets/otter-doctor
var doctorScript string

//go:embed assets/otter-redirect
var redirectScript string

//go:embed assets/otter-subreaper
var subreaperScript string

//go:embed assets/initialization-scripts
var initScriptsFS embed.FS

// ProvisionScripts ensures that all necessary scripts are created in the given host directory.
// It returns the path to the directory where the scripts are stored, and whether any scripts were updated.
func ProvisionScripts(scriptsDir string) (string, bool, error) {
	if scriptsDir == "" {
		return "", false, errors.New("scripts directory could not be determined: set 'scripts-dir' in otter.conf or ensure HOME is set")
	}

	dir := scriptsDir
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
		{"otter-doctor", doctorScript},
		{"otter-redirect", redirectScript},
		{"otter-subreaper", subreaperScript},
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

	const initScriptsSrcDir = "assets/initialization-scripts"
	initScriptsDestDir := filepath.Join(dir, "initialization-scripts")

	walkErr := fs.WalkDir(initScriptsFS, initScriptsSrcDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(initScriptsSrcDir, path)
		if err != nil {
			return err
		}
		destPath := filepath.Join(initScriptsDestDir, relPath)

		if entry.IsDir() {
			//nolint:gosec // 0755 is correct for directories: executable bit grants traversal permission to all users
			if err := os.MkdirAll(destPath, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", destPath, err)
			}
			return nil
		}

		content, err := fs.ReadFile(initScriptsFS, path)
		if err != nil {
			return fmt.Errorf("failed to read embedded script %s: %w", path, err)
		}

		newHash := sha256.Sum256(content)
		if existing, err := os.ReadFile(destPath); err == nil {
			if sha256.Sum256(existing) == newHash {
				return nil
			}
		}
		//nolint:gosec // 0755 is required: scripts must be executable by the container runtime
		if err := os.WriteFile(destPath, content, 0755); err != nil {
			return fmt.Errorf("failed to write script %s: %w", destPath, err)
		}
		updated = true
		return nil
	})
	if walkErr != nil {
		return "", false, fmt.Errorf("failed to provision initialization-scripts: %w", walkErr)
	}

	return dir, updated, nil
}
