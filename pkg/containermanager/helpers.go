package containermanager

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"time"
)

// bindGOOS is the OS used to decide bind-mount propagation. It is a package
// variable so tests can exercise the macOS branch without running on darwin.
//
//nolint:gochecknoglobals // deliberate test seam for the macOS bind-propagation branch
var bindGOOS = runtime.GOOS

// BindPropagation returns the propagation suffix for shared bind mounts.
//
// macOS (Docker Desktop / Colima) runs containers inside a Linux VM that mounts
// host paths as private, so rslave/rshared propagation is unsupported and must
// be dropped there — mirroring the reference shell (distrobox-create:677-685).
func BindPropagation() string {
	if bindGOOS == "darwin" {
		return ""
	}
	return ":rslave"
}

// ReadOnlyBindPropagation is BindPropagation for read-only mounts.
func ReadOnlyBindPropagation() string {
	if bindGOOS == "darwin" {
		return ":ro"
	}
	return ":ro,rslave"
}

func PathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// NvidiaToolkitAvailable reports whether the NVIDIA Container Toolkit is
// actually usable on this host: nvidia-ctk on PATH (the real binary, not an
// inference from OS/distro), and a generated CDI spec present at one of the
// two locations nvidia-ctk writes to by default.
func NvidiaToolkitAvailable() bool {
	if _, err := exec.LookPath("nvidia-ctk"); err != nil {
		return false
	}
	return PathExists("/etc/cdi/nvidia.yaml") || PathExists("/var/run/cdi/nvidia.yaml")
}

func IsSymlink(path string) bool {
	info, err := os.Lstat(path)
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeSymlink != 0
}

func Btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}

func BuildXDGPaths(envVar string, standardPaths []string) string {
	containerPaths := os.Getenv(envVar)

	for _, sp := range standardPaths {
		if containerPaths == "" {
			containerPaths = sp
		} else if !slices.Contains(strings.Split(containerPaths, ":"), sp) {
			containerPaths = containerPaths + ":" + sp
		}
	}

	return containerPaths
}

func FilterEnvVars() []string {
	result := []string{}

	// Compile regex for XDG_.*_DIRS pattern
	xdgDirsPattern := regexp.MustCompile(`^XDG_.*_DIRS$`)

	// Excluded prefixes
	excludedPrefixes := []string{
		"CONTAINER_ID",
		"FPATH",
		"HOST",
		"HOSTNAME",
		"HOME",
		"PATH",
		"PROFILEREAD",
		"PWD", // host PWD is replaced by --env=PWD=<workdir>
		"SHELL",
		"XDG_SEAT",
		"XDG_VTNR",
		"_", // Variables starting with underscore
	}

	for _, env := range os.Environ() {
		// Must contain '='
		if !strings.Contains(env, "=") {
			continue
		}

		// Exclude if contains ", `, or $
		if strings.ContainsAny(env, "\"`$") {
			continue
		}

		// Split into key and value
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := parts[0]

		// Check excluded prefixes
		excluded := false
		for _, prefix := range excludedPrefixes {
			if strings.HasPrefix(key, prefix) {
				excluded = true
				break
			}
		}

		if excluded || xdgDirsPattern.MatchString(key) {
			continue
		}

		result = append(result, env)
	}

	return result
}

func GetWorkDir(hostHome, containerHome string, noWorkDir bool) (string, error) {
	workDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("error getting working dir: %w", err)
	}

	if noWorkDir {
		return containerHome, nil
	}

	if workDir == "" && hostHome == "" {
		return "/", nil
	}

	if workDir == "" {
		return containerHome, nil
	}

	if workDir == hostHome {
		return containerHome, nil
	}

	if prefix := hostHome + "/"; strings.HasPrefix(workDir, prefix) {
		return containerHome + "/" + strings.TrimPrefix(workDir, prefix), nil
	}

	return "/run/host" + workDir, nil
}

func BuildCommandArgs(customCommand []string, user string, noTTY bool, unshareGroups bool) []string {
	args := customCommand
	if len(args) == 0 {
		// Default: execute user's shell with login
		args = []string{"/bin/sh", "-c", fmt.Sprintf("$(getent passwd '%s' | cut -f 7 -d :) -l", user)}
	}

	// Handle unshare_groups mode - use su to trigger proper login.
	// Deliberately not passing -m/--preserve-environment: a real login is
	// exactly what should resolve $HOME (and other login-context vars) from
	// the target user's own passwd entry, rather than inheriting whatever
	// the outer (typically root) exec's environment happened to have.
	//
	// The whole su invocation is run through otter-subreaper first: it marks
	// itself as a child subreaper (prctl PR_SET_CHILD_SUBREAPER) and execs
	// into su, so any process the session double-forks and detaches (e.g.
	// fish's own fish_update_completions worker) reparents to this session
	// instead of falling through to the container's real PID 1.
	if unshareGroups {
		unshareArgs := []string{"python3", "/usr/lib/otter/scripts/otter-subreaper", "su"}
		if !noTTY {
			unshareArgs = append(unshareArgs, "--pty")
		}
		unshareArgs = append(unshareArgs, "-s", "/bin/sh", "-c", `"$0" "$@"`, "--", user)
		unshareArgs = append(unshareArgs, args...)
		return unshareArgs
	}

	return args
}

func TimestampNow() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}
