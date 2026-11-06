package userenv

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
)

// LoadUserEnvironment loads the user environment variables
// Data is sourced from environment variables, and falls back to system calls
// if the environment variables are not set
// Expected variables are:
// - USER
// - HOME
// - SHELL
func LoadUserEnvironment(ctx context.Context) *UserEnvironment {
	env := &UserEnvironment{}

	// USER
	env.User = os.Getenv("USER")
	if env.User == "" {
		// Try id -run (most reliable)
		if output, err := exec.CommandContext(ctx, "id", "-run").Output(); err == nil {
			env.User = strings.TrimSpace(string(output))
		} else if u, err := user.Current(); err == nil {
			// Fallback to os/user
			env.User = u.Username
		} else {
			env.User = "nobody"
		}
	}

	// Get passwd entry once for HOME and SHELL
	passwdEntry := getPasswdFields(ctx, env.User)

	// HOME
	env.Home = os.Getenv("HOME")
	if env.Home == "" {
		if len(passwdEntry) > 5 && passwdEntry[5] != "" {
			env.Home = passwdEntry[5]
		} else if u, err := user.Lookup(env.User); err == nil {
			env.Home = u.HomeDir
		} else {
			env.Home = fmt.Sprintf("/home/%s", env.User)
		}
	}

	// SHELL
	env.Shell = os.Getenv("SHELL")
	if env.Shell == "" {
		if len(passwdEntry) > 6 && passwdEntry[6] != "" {
			env.Shell = passwdEntry[6]
		} else {
			env.Shell = "/bin/sh"
		}
	}

	// USER ID / GROUP ID
	// os.Getuid/os.Getgid only return -1 on Windows; otter is Linux-only,
	// so they always succeed.
	env.UserID = strconv.Itoa(os.Getuid())
	env.GroupID = strconv.Itoa(os.Getgid())

	// DESKTOP ENTRY DIR
	//
	// otter always uses the canonical ~/.local/share base for its own
	// integration files (desktop entries, export index, icons). Honouring
	// XDG_DATA_HOME here would split otter's bookkeeping across two roots
	// depending on how the command was invoked.
	env.DesktopEntryBaseDir = filepath.Join(env.Home, ".local", "share")

	return env
}

// LoadTargetUserEnvironment loads the environment of the user on whose behalf
// otter is acting. When otter runs through sudo (effective uid 0 with SUDO_UID
// set), that is the invoking user, not root. Export scanning, rebuilding, and
// pruning must always touch the target user's files.
func LoadTargetUserEnvironment(ctx context.Context) *UserEnvironment {
	if os.Geteuid() == 0 {
		if uid := os.Getenv("SUDO_UID"); uid != "" {
			if env := loadPasswdEnvironment(ctx, uid); env != nil {
				return env
			}
		}
	}

	return LoadUserEnvironment(ctx)
}

// loadPasswdEnvironment resolves a user directly from a passwd entry by UID.
// It returns nil when the UID cannot be resolved, so callers can fall back to
// the normal user environment lookup.
func loadPasswdEnvironment(ctx context.Context, uid string) *UserEnvironment {
	output, err := exec.CommandContext(ctx, "getent", "passwd", uid).Output()
	if err != nil {
		return nil
	}

	fields := strings.Split(strings.TrimSpace(string(output)), ":")
	if len(fields) < 7 {
		return nil
	}

	env := &UserEnvironment{
		User:    fields[0],
		UserID:  fields[2],
		GroupID: fields[3],
		Home:    fields[5],
		Shell:   fields[6],
	}
	if env.User == "" || env.Home == "" {
		return nil
	}
	if env.Shell == "" {
		env.Shell = "/bin/sh"
	}

	env.DesktopEntryBaseDir = filepath.Join(env.Home, ".local", "share")
	return env
}

// getPasswdFields returns all fields from passwd entry for a user
func getPasswdFields(ctx context.Context, userName string) []string {
	// Use getent passwd (handles LDAP, NIS, local users)
	//nolint:gosec // G702: userName comes from the OS user lookup, not untrusted user input
	output, err := exec.CommandContext(ctx, "getent", "passwd", userName).Output()
	if err != nil {
		return nil
	}

	return strings.Split(strings.TrimSpace(string(output)), ":")
}
