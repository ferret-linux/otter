package config

import (
	"fmt"
	"os"
	"path/filepath"

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

	return writeConfigFileAtomic(path, data)
}

// writeConfigFileAtomic writes data to a temp file in the same directory and
// renames it into place, so an interrupted write cannot truncate the existing
// config. A symlink at the destination is refused rather than overwritten.
func writeConfigFileAtomic(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".otter.conf-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp config file: %w", err)
	}
	tmpName := tmp.Name()
	committed := false
	defer func() {
		if !committed {
			_ = os.Remove(tmpName)
		}
	}()

	_, writeErr := tmp.Write(data)
	chmodErr := tmp.Chmod(0o600)
	syncErr := tmp.Sync()
	closeErr := tmp.Close()
	for _, stepErr := range []error{writeErr, chmodErr, syncErr, closeErr} {
		if stepErr != nil {
			return fmt.Errorf("failed to write temp config file: %w", stepErr)
		}
	}

	if info, err := os.Lstat(path); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing to overwrite symlinked config file %s", path)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("failed to replace config file: %w", err)
	}
	committed = true

	return nil
}
