package config

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// WithConfigLock serializes a config read-modify-write cycle across otter
// processes. SaveFileConfig's atomic rename prevents truncation but not a
// stale whole-file replacement: two editors that each load the config before
// the other writes would lose one of their updates. Locking the whole cycle
// makes the second reader wait and see the first writer's committed values.
//
// The lock file lives beside the user config (never in a system-wide path),
// contains no data, and is handed to the invoking user when otter runs under
// an escalator so later unprivileged invocations can open it too.
func WithConfigLock(fn func() error) error {
	dir := filepath.Dir(userConfigPath())
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	lockPath := filepath.Join(dir, ".otter-settings.lock")
	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return fmt.Errorf("failed to open config lock: %w", err)
	}
	defer func() { _ = file.Close() }()

	if uid, gid, ok := escalatedTargetOwner(); ok {
		if err := os.Chown(lockPath, uid, gid); err != nil {
			return fmt.Errorf("failed to hand config lock to target user: %w", err)
		}
	}

	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("failed to lock config: %w", err)
	}
	defer func() { _ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN) }()

	return fn()
}
