package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// LoadFileConfig loads and merges otter.conf from all four layered
// locations (see getConfigFilePaths), returning the raw FileConfig shape
// rather than the derived Values. This is what the settings TUI edits
// directly, so it starts from the actual effective on-disk values rather
// than zero values or Values' resolved defaults.
func LoadFileConfig() (*FileConfig, error) {
	files, err := getConfigFilePaths()
	if err != nil {
		return nil, fmt.Errorf("failed to get config file paths: %w", err)
	}

	cfg := defaults()

	for _, file := range files {
		if err := readConfigFile(file, &cfg); err != nil {
			return nil, fmt.Errorf("failed to read config file %q: %w", file, err)
		}
	}

	return &cfg, nil
}

func LoadValues() (*Values, error) {
	cfg, err := LoadFileConfig()
	if err != nil {
		return nil, err
	}

	return toValues(*cfg), nil
}

// userConfigPath returns the path to the user's own otter.conf, under
// XDG_CONFIG_HOME (or ~/.config if that's unset). This is the only
// location SaveFileConfig is ever allowed to write to, and the last
// (highest-priority) entry getConfigFilePaths returns.
func userConfigPath() string {
	xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")
	if xdgConfigHome == "" {
		xdgConfigHome = filepath.Join(os.Getenv("HOME"), ".config")
	}
	return filepath.Join(xdgConfigHome, "otter", "otter.conf")
}

func getConfigFilePaths() ([]string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to get executable path: %w", err)
	}

	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate symlinks for executable path: %w", err)
	}

	selfDir := filepath.Dir(execPath)

	// Source configuration files in increasing priority order: each later
	// file's values override earlier ones for any key it sets, giving the
	// user's own local config priority over system-wide defaults.
	//
	// On NixOS, for the otter derivation to pick up a static config file shipped
	// by the package maintainer the path must be relative to the script itself.
	return []string{
		filepath.Join(selfDir, "..", "share", "otter", "otter.conf"), // for NixOS
		"/usr/share/otter/otter.conf",
		"/etc/otter/otter.conf",
		userConfigPath(),
	}, nil
}

func readConfigFile(filePath string, cfg *FileConfig) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read config file: %w", err)
	}

	if err := toml.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("failed to parse TOML: %w", err)
	}

	return nil
}
