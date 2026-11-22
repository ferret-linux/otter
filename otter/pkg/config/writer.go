package config

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/BurntSushi/toml"
)

// SaveFileConfig writes cfg to the user's own otter.conf
// ($XDG_CONFIG_HOME/otter/otter.conf, or ~/.config/otter/otter.conf if
// XDG_CONFIG_HOME is unset), overwriting it entirely. It never writes to
// /etc/otter/otter.conf or /usr/share/otter/otter.conf, regardless of
// --root or where a currently-effective value happened to be sourced from
// — those are system-wide defaults, not something a per-user settings
// editor should touch.
//
// This always regenerates the whole file from cfg rather than patching it
// in place, so any comments or formatting-style choices (dotted keys,
// inline tables, table headers — see docs/05.configuration.md) already in
// the user's file are lost; the parsed settings themselves are preserved,
// which is what `otter settings` round-trips.
func SaveFileConfig(cfg *FileConfig) error {
	path := userConfigPath()
	configDir := filepath.Dir(path)

	if err := os.MkdirAll(configDir, 0o750); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	if info, err := os.Lstat(configDir); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing to write config through symlinked directory %s", configDir)
	}

	data, err := toml.Marshal(*cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// O_NOFOLLOW keeps a pre-existing symlink at the config path from
	// redirecting the write to an arbitrary file.
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC|syscall.O_NOFOLLOW, 0o600)
	if err != nil {
		return fmt.Errorf("failed to open config file: %w", err)
	}
	_, writeErr := file.Write(data)
	closeErr := file.Close()
	if writeErr != nil {
		return fmt.Errorf("failed to write config file: %w", writeErr)
	}
	if closeErr != nil {
		return fmt.Errorf("failed to close config file: %w", closeErr)
	}

	return nil
}
