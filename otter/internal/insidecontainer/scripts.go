package insidecontainer //nolint:cyclop // package average reflects explicit provisioning helpers, not a single complex function

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

//go:embed assets/otter-service
var serviceScript string

//go:embed assets/otter-subreaper
var subreaperScript string

//go:embed assets/initialization-scripts
var initScriptsFS embed.FS

// ProvisionScripts ensures that all necessary scripts are created in the given host directory.
// It returns the path to the directory where the scripts are stored, and whether any scripts were updated.
//
//nolint:gocognit // provisioning flow: sequential steps with explicit error handling
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
		{"otter-service", serviceScript},
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
		if err := writeScriptAtomic(destFilePath, []byte(script.content)); err != nil {
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
			return fmt.Errorf("resolve script path %s: %w", path, err)
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
		if err := writeScriptAtomic(destPath, content); err != nil {
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

// writeScriptAtomic writes a script to a temp file in the same directory and
// renames it into place, so an interrupted write cannot leave a truncated
// executable behind.
func writeScriptAtomic(path string, content []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".otter-script-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp script file: %w", err)
	}
	tmpName := tmp.Name()
	committed := false
	defer func() {
		if !committed {
			_ = os.Remove(tmpName)
		}
	}()

	_, writeErr := tmp.Write(content)
	chmodErr := tmp.Chmod(0o755)
	syncErr := tmp.Sync()
	closeErr := tmp.Close()
	for _, stepErr := range []error{writeErr, chmodErr, syncErr, closeErr} {
		if stepErr != nil {
			return fmt.Errorf("write temp script file: %w", stepErr)
		}
	}

	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("rename temp script file: %w", err)
	}
	committed = true

	return nil
}
