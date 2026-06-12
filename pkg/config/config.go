package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type containerConfig struct {
	Hostname string `toml:"hostname"`
	Image    string `toml:"image"`
	Name     string `toml:"name"`
}

type settingsConfig struct {
	Shell      string `toml:"shell"`
	InitSystem bool   `toml:"init-system"`
	Rootful    bool   `toml:"rootful"`
}

type preferencesConfig struct {
	ContainerManager string `toml:"container-manager"`
	SudoProgram      string `toml:"sudo-program"`
	NoEntry          bool   `toml:"no-entry"`
}

type fileConfig struct {
	Container   containerConfig   `toml:"container"`
	Settings    settingsConfig    `toml:"settings"`
	Preferences preferencesConfig `toml:"preferences"`
}

type Values struct {
	ContainerManagerType  string
	SudoProgram           string
	DefaultContainerImage string
	DefaultContainerName  string
	DefaultHostname       string
	DefaultShell          string
	DefaultInitSystem     bool
	DefaultRootful        bool
	DefaultNoEntry        bool
}

func defaults() fileConfig {
	return fileConfig{
		Container: containerConfig{
			Image: "ghcr.io/ferret-linux/ubuntu-otr:lts",
			Name:  "my-container",
		},
		Preferences: preferencesConfig{
			ContainerManager: "autodetect",
			SudoProgram:      "autodetect",
		},
	}
}

func DefaultValues() *Values {
	return toValues(defaults())
}

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

func toValues(cfg fileConfig) *Values {
	return &Values{
		ContainerManagerType:  cfg.Preferences.ContainerManager,
		SudoProgram:           cfg.Preferences.SudoProgram,
		DefaultContainerImage: cfg.Container.Image,
		DefaultContainerName:  cfg.Container.Name,
		DefaultHostname:       cfg.Container.Hostname,
		DefaultShell:          cfg.Settings.Shell,
		DefaultInitSystem:     cfg.Settings.InitSystem,
		DefaultRootful:        cfg.Settings.Rootful,
		DefaultNoEntry:        cfg.Preferences.NoEntry,
	}
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

	// Source configuration files, this is done in an hierarchy so local files have
	// priority over system defaults
	// leave priority to environment variables.
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
