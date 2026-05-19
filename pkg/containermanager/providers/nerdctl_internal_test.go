package providers

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ferret-linux/otter/internal/userenv"
	"github.com/ferret-linux/otter/pkg/containermanager"
	"github.com/ferret-linux/otter/pkg/ui"
)

func TestNerdctl_makeCreateCommand(t *testing.T) {
	nerdctl := NewNerdctl(false, "sudo", false)

	userEnv := &userenv.UserEnvironment{
		User:    "user",
		UserID:  "1000",
		GroupID: "1000",
		Home:    "/home/user",
		Shell:   "/bin/sh",
	}

	containerAdditionalFlags := []string{}
	containerAdditionalPackages := []string{}
	containerAdditionalVolumes := []string{
		"/path/to/my-volume:/var/local/my-volume:ro",
	}

	cmd := nerdctl.makeCreateCommand(
		"my-container",              // containerName
		"my-image",                  // containerImage
		containerAdditionalFlags,    // containerAdditionalFlags
		"my-hostname",               // containerHostname
		containerAdditionalPackages, // containerAdditionalPackages
		containerAdditionalVolumes,  // containerAdditionalVolumes
		"",                          // containerUserCustomHome
		"",                          // containerPlatform
		false,                       // nopasswd
		true,                        // init
		"echo 'pre-init-hook'",      // containerPreInitHook
		"echo 'init-hook'",          // containerInitHook
		false,                       // nvidia
		false,                       // unshareDevsys
		false,                       // unshareGroups
		false,                       // unshareIPC
		false,                       // unshareNetNS
		false,                       // unshareProcess
		userEnv,                     // userEnv
		"/path/to/otter-init",       // otterInitPath
		"/path/to/otter-export",     // otterExportPath
		"/path/to/otter-hostexec",   // otterHostexecPath
	)

	selinuxVolume := ""
	if containermanager.PathExists("/sys/fs/selinux") {
		selinuxVolume = " --volume /sys/fs/selinux"
	}

	expected := oneline(`
 create
 --hostname my-hostname
 --name my-container
 --privileged
 --security-opt label=disable
 --security-opt apparmor=unconfined
 --user root:root
 --ipc host
 --network host
 --pid host
 --label manager=otter
 --label otter.unshare_groups=0
 --env SHELL=sh
 --env HOME=/home/user
 --env container=nerdctl
 --env TERMINFO_DIRS=/usr/share/terminfo:/run/host/usr/share/terminfo
 --env CONTAINER_ID=my-container
 --volume /tmp:/tmp:rslave
 --volume /path/to/otter-export:/usr/bin/otter-export:ro
 --volume /path/to/otter-hostexec:/usr/bin/otter-host-exec:ro
 --volume /home/user:/home/user:rslave
 --volume /:/run/host/:rslave
 --volume /dev:/dev:rslave
 --volume /sys:/sys:rslave
 --cgroupns host
 --stop-signal SIGRTMIN+3
 --mount type=tmpfs,destination=/run
 --mount type=tmpfs,destination=/run/lock
 --mount type=tmpfs,destination=/var/lib/journal
 --volume /dev/pts
 --volume /dev/null:/dev/ptmx
 ` + selinuxVolume + `
 --volume /var/log/journal
 --volume /etc/hosts:/etc/hosts:ro
 --volume /etc/resolv.conf:/etc/resolv.conf:ro
 --volume /path/to/my-volume:/var/local/my-volume:ro
 --volume /path/to/otter-init:/usr/bin/entrypoint:ro
 --entrypoint /usr/bin/entrypoint
 my-image
 --verbose
 --name user
 --user 1000
 --group 1000
 --home /home/user
 --init 1
 --nvidia 0
 --pre-init-hooks echo 'pre-init-hook'
 --additional-packages
 -- echo 'init-hook'
`)

	got := oneline(strings.Join(cmd, " "))

	assert.Equal(t, expected, got)
}

func TestNerdctlEnterPropagatesStartError(t *testing.T) {
	installFakeNerdctlRuntime(t)
	t.Setenv("FAKE_INSPECT_STDOUT", nerdctlFakeInspectJSON("exited"))
	t.Setenv("FAKE_START_EXIT", "9")
	t.Setenv("FAKE_START_STDERR", "start failed")

	err := NewNerdctl(false, "sudo", false).Enter(
		t.Context(),
		containermanager.EnterOptions{
			ContainerName: "box",
			NoTTY:         true,
			NoWorkDir:     true,
		},
		ui.NewDevNullProgress(),
		ui.NewPrinter(io.Discard, false),
	)

	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to start container")
}

func TestNerdctlEnterPropagatesExecError(t *testing.T) {
	installFakeNerdctlRuntime(t)
	t.Setenv("FAKE_INSPECT_STDOUT", nerdctlFakeInspectJSON("running"))
	t.Setenv("FAKE_EXEC_EXIT", "7")
	t.Setenv("FAKE_EXEC_STDERR", "exec failed")

	err := NewNerdctl(false, "sudo", false).Enter(
		t.Context(),
		containermanager.EnterOptions{
			ContainerName: "box",
			NoTTY:         true,
			NoWorkDir:     true,
		},
		ui.NewDevNullProgress(),
		ui.NewPrinter(io.Discard, false),
	)

	require.Error(t, err)
	assert.ErrorContains(t, err, "exit status 7")
}

func installFakeNerdctlRuntime(t *testing.T) {
	t.Helper()

	tmpDir := t.TempDir()
	runtimePath := filepath.Join(tmpDir, "nerdctl")

	// nerdctl uses subcommand form: nerdctl container inspect, nerdctl image inspect
	const script = `#!/bin/sh
cmd="$1"
subcmd="$2"
shift 2
case "$cmd" in
  container)
    case "$subcmd" in
      inspect)
        if [ -n "$FAKE_INSPECT_STDOUT" ]; then
          printf "%s" "$FAKE_INSPECT_STDOUT"
        fi
        exit "${FAKE_INSPECT_EXIT:-0}"
        ;;
    esac
    ;;
  start)
    if [ -n "$FAKE_START_STDERR" ]; then
      printf "%s" "$FAKE_START_STDERR" >&2
    fi
    exit "${FAKE_START_EXIT:-0}"
    ;;
  logs)
    if [ -n "$FAKE_LOGS_STDOUT" ]; then
      printf "%s" "$FAKE_LOGS_STDOUT"
    fi
    exit "${FAKE_LOGS_EXIT:-0}"
    ;;
  exec)
    if [ -n "$FAKE_EXEC_STDERR" ]; then
      printf "%s" "$FAKE_EXEC_STDERR" >&2
    fi
    exit "${FAKE_EXEC_EXIT:-0}"
    ;;
  *)
    exit 0
    ;;
esac
`

	err := os.WriteFile(runtimePath, []byte(script), 0o755)
	require.NoError(t, err)

	t.Setenv("PATH", tmpDir+":"+os.Getenv("PATH"))
	t.Setenv("USER", "testuser")
	t.Setenv("HOME", "/home/testuser")
	t.Setenv("SHELL", "/bin/sh")
}

func nerdctlFakeInspectJSON(status string) string {
	return `[{"Id":"container-id","State":{"Status":"` + status + `"},"Config":{"Labels":{"otter.unshare_groups":"0"},"Env":["HOME=/home/testuser","PATH=/usr/bin:/bin"]}}]`
}
