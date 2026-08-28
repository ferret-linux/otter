package containermanager

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/creack/pty"
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

//nolint:goconst // standard FHS paths are clearer inline than as named constants
func BuildContainerPath(cleanPath bool, hostPath string, containerPath string) string {
	standardPaths := []string{"/usr/local/sbin", "/usr/local/bin", "/usr/sbin", "/usr/bin", "/sbin", "/bin"}

	if cleanPath {
		return strings.Join(standardPaths, ":")
	}

	// If no host PATH, use the container's PATH if available
	if hostPath == "" {
		if containerPath != "" {
			return containerPath
		}
		return strings.Join(standardPaths, ":")
	}

	// Add standard paths not in host PATH
	hostSegments := strings.Split(hostPath, ":")
	var additionalPaths []string
	for _, sp := range standardPaths {
		if !slices.Contains(hostSegments, sp) {
			additionalPaths = append(additionalPaths, sp)
		}
	}

	merged := hostPath
	if len(additionalPaths) > 0 {
		merged = hostPath + ":" + strings.Join(additionalPaths, ":")
	}

	return reorderFHSPath(merged)
}

// reorderFHSPath ensures /usr/local/bin precedes /usr/bin and /usr/local/sbin
// precedes /usr/sbin, so otter wrappers in /usr/local/* win — mirroring the
// reference shell (distrobox-enter:478-512).
func reorderFHSPath(path string) string {
	var reordered []string
	for _, p := range strings.Split(path, ":") {
		switch p {
		case "/usr/local/bin", "/usr/local/sbin":
			// Skip here; re-inserted right before its /usr counterpart.
		case "/usr/bin":
			reordered = append(reordered, "/usr/local/bin", "/usr/bin")
		case "/usr/sbin":
			reordered = append(reordered, "/usr/local/sbin", "/usr/sbin")
		default:
			reordered = append(reordered, p)
		}
	}

	result := strings.Join(reordered, ":")

	// If /usr/bin or /usr/sbin were absent, their local counterparts were
	// skipped above; re-add any that went missing (prepended, like the shell).
	for _, lp := range []string{"/usr/local/bin", "/usr/local/sbin"} {
		if !slices.Contains(strings.Split(result, ":"), lp) {
			result = lp + ":" + result
		}
	}

	return result
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

	// Handle unshare_groups mode - use su to trigger proper login
	if unshareGroups {
		unshareArgs := []string{"su"}
		if !noTTY {
			unshareArgs = append(unshareArgs, "--pty")
		}
		unshareArgs = append(unshareArgs, "-m", "-s", "/bin/sh", "-c", `"$0" "$@"`, "--", user)
		unshareArgs = append(unshareArgs, args...)
		return unshareArgs
	}

	return args
}

func TimestampNow() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

// RunWithPullOutput runs cmd with its combined stdout/stderr streamed into
// out. If out implements PullOutputSizer, cmd is given a pseudo-terminal
// sized to match out's current size, kept in sync for the life of cmd via
// out.OnResize, so the pulling process renders its native live-progress
// output (which it otherwise suppresses when it detects a non-TTY pipe)
// scaled to fit the region out renders into. If out does not implement
// PullOutputSizer, cmd's stdout/stderr are connected to out directly, with
// the same non-TTY behavior as before.
func RunWithPullOutput(cmd *exec.Cmd, out PullOutput) error {
	sizer, ok := out.(PullOutputSizer)
	if !ok {
		cmd.Stdout = out
		cmd.Stderr = out
		//nolint:wrapcheck // callers wrap this error with their own pull-specific message
		return cmd.Run()
	}

	rows, cols := sizer.Size()
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: uint16(rows), Cols: uint16(cols)}) //nolint:gosec // terminal dims fit uint16
	if err != nil {
		return fmt.Errorf("failed to start pseudo-terminal: %w", err)
	}
	defer ptmx.Close()

	sizer.OnResize(func(rows, cols int) {
		_ = pty.Setsize(ptmx, &pty.Winsize{Rows: uint16(rows), Cols: uint16(cols)}) //nolint:gosec // terminal dims fit uint16
	})

	_, copyErr := io.Copy(out, ptmx)
	waitErr := cmd.Wait()

	// A closed pty on the child's exit surfaces as a read error from
	// io.Copy; that's expected once the child is done, not a real failure,
	// so only cmd.Wait's exit status is treated as the actual result.
	_ = copyErr

	//nolint:wrapcheck // callers wrap this error with their own pull-specific message
	return waitErr
}
