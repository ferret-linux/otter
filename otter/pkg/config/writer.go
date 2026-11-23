package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/BurntSushi/toml"

	"github.com/ferret-linux/otter/internal/userenv"
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

	// Record directories we create so an escalated invocation can hand them
	// back to the user it is acting on behalf of.
	createdDirs := missingConfigDirs(configDir)

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

	if err := writeConfigFileAtomic(path, data); err != nil {
		return err
	}

	// When otter runs through an escalator the path points at the invoking
	// user's home, but the directories and file were created as root. Without
	// chowning them the user cannot read or edit their own config.
	uid, gid, ok := escalatedTargetOwner()
	if !ok {
		return nil
	}
	for _, dir := range append(createdDirs, path) {
		if err := os.Chown(dir, uid, gid); err != nil {
			return fmt.Errorf("failed to hand config path %s to target user: %w", dir, err)
		}
	}
	return nil
}

// escalatedTargetOwner returns the uid/gid of the user otter is acting on
// behalf of, when running root with an escalator (sudo/doas/pkexec). It
// returns ok=false for ordinary invocations and for real root.
func escalatedTargetOwner() (int, int, bool) {
	if os.Geteuid() != 0 {
		return 0, 0, false
	}
	env := userenv.LoadTargetUserEnvironment(context.Background())
	uid, uidErr := strconv.Atoi(env.UserID)
	gid, gidErr := strconv.Atoi(env.GroupID)
	if uidErr != nil || gidErr != nil {
		return 0, 0, false
	}
	if uid == os.Geteuid() && gid == os.Getegid() {
		return 0, 0, false
	}
	return uid, gid, true
}

// missingConfigDirs lists the path components of dir (deepest first) that do
// not exist yet, stopping at the first existing ancestor.
func missingConfigDirs(dir string) []string {
	var missing []string
	for current := dir; current != "" && current != string(os.PathSeparator); current = filepath.Dir(current) {
		if _, err := os.Lstat(current); err == nil {
			break
		}
		missing = append(missing, current)
	}
	return missing
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
