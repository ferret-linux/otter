package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

func LoadValues() (*Values, error) {
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

	return toValues(cfg), nil
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

	xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")
	if xdgConfigHome == "" {
		xdgConfigHome = filepath.Join(os.Getenv("HOME"), ".config")
	}

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
		filepath.Join(xdgConfigHome, "otter", "otter.conf"),
	}, nil
}

func readConfigFile(filePath string, cfg *fileConfig) error {
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
