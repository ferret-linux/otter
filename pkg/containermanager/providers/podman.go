//nolint:goconst // CLI flag strings are intentionally repeated per-provider; they may diverge independently
package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ferret-linux/otter/internal/insidecontainer"
	"github.com/ferret-linux/otter/internal/userenv"
	"github.com/ferret-linux/otter/pkg/containermanager"
	"github.com/ferret-linux/otter/pkg/ttyutil"
	"github.com/ferret-linux/otter/pkg/ui"
)

type Podman struct {
	command     podmanCommand
	binary      string
	root        bool
	sudoCommand string
}

// podmanCommand represents the executable name for the Podman provider.
type podmanCommand string

const (
	podmanCommandPodman podmanCommand = "podman"
)

var _ containermanager.ContainerManager = &Podman{}

func newPodman(command podmanCommand, root bool, sudoCommand string) *Podman {
	binary, err := exec.LookPath(string(command))
	if err != nil {
		binary = string(command)
	}
	return &Podman{
		command:     command,
		binary:      binary,
		sudoCommand: sudoCommand,
		root:        root,
	}
}

func NewPodman(root bool, sudoCommand string) *Podman {
	return newPodman(podmanCommandPodman, root, sudoCommand)
}

func (p *Podman) CloneAsRoot() containermanager.ContainerManager {
	cp := *p
	cp.root = true
	return &cp
}

func (p *Podman) Name() string {
	return string(p.command)
}

// podmanContainer represents the JSON output from `podman ps --format json`.
type podmanContainer struct {
	ID     string            `json:"ID"`
	Image  string            `json:"Image"`
	Names  []string          `json:"Names"`
	Status string            `json:"Status"`
	Labels map[string]string `json:"Labels"`
}

func (p *Podman) ListContainers(ctx context.Context) ([]containermanager.Container, error) {
	args := []string{"ps", "-a", "--no-trunc", "--format", "json"}
	out, err := p.run(ctx, args, runOptions{})
	if err != nil {
		return nil, err
	}
	return parsePodmanContainerList(out)
}

func (p *Podman) Create(
	ctx context.Context,
	opts containermanager.CreateOptions,
) error {
	userEnv := userenv.LoadUserEnvironment(ctx)

	scriptsDir, _, err := insidecontainer.ProvisionScripts(opts.ScriptsDir)
	if err != nil {
		return fmt.Errorf("failed to provision scripts: %w", err)
	}

	// ensure custom home dir exists, if needed
	if opts.ContainerUserCustomHome != "" && !containermanager.PathExists(opts.ContainerUserCustomHome) {
		//nolint:gosec // 0755 is the same as from distrobox v1, let's keep it for compatibility
		if err := os.MkdirAll(opts.ContainerUserCustomHome, 0755); err != nil {
			return fmt.Errorf("failed to create custom home directory: %w", err)
		}
	}

	userEnv.Shell = opts.ContainerShell

	cmd := p.makeCreateCommand(
		ctx,
		opts,
		userEnv,
		filepath.Join(scriptsDir, "otter-init"),
		filepath.Join(scriptsDir, "otter-export"),
		filepath.Join(scriptsDir, "otter-host-exec"),
		filepath.Join(scriptsDir, "otter"),
		filepath.Join(scriptsDir, "otter-subreaper"),
		filepath.Join(scriptsDir, "initialization-scripts"),
	)

	_, err = p.run(ctx, cmd, runOptions{})
	if err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}
	return nil
}

// makeCreateCommand builds the podman create command with all necessary options.
//
//nolint:gocognit,funlen,gocyclo,cyclop // ignore cognitive complexity here, the function is mostly imperative option appending
func (p *Podman) makeCreateCommand(
	ctx context.Context,
	opts containermanager.CreateOptions,
	userEnv *userenv.UserEnvironment,
	otterInitPath string,
	otterExportPath string,
	otterHostexecPath string,
	otterPath string,
	otterSubreaperPath string,
	initScriptsPath string,
) []string {
	containerName := opts.ContainerName
	containerImage := opts.ContainerImage
	containerAdditionalFlags := opts.AdditionalFlags
	containerHostname := opts.ContainerHostname
	hostnameExplicit := opts.ContainerHostnameExplicit
	containerAdditionalPackages := opts.AdditionalPackages
	containerAdditionalVolumes := opts.AdditionalVolumes
	customHome := opts.ContainerUserCustomHome
	containerPlatform := opts.ContainerPlatform
	nopasswd := opts.Nopasswd
	init := opts.Init
	containerPreInitHook := opts.ContainerPreInitHook
	containerInitHook := opts.ContainerInitHook
	gpu := opts.GPU
	noUsernsLimit := opts.NoUsernsLimit
	memory := opts.Memory
	cpuThreads := opts.CPUThreads
	unshareDevsys := opts.UnshareDevsys
	unshareGroups := opts.UnshareGroups
	unshareIPC := opts.UnshareIPC
	unshareNetNS := opts.UnshareNetNS
	unshareProcess := opts.UnshareProcess

	containerUserName := userEnv.User
	containerUserUID := userEnv.UserID
	containerUserGID := userEnv.GroupID
	shellFilepath := filepath.Base(userEnv.Shell)

	// hostHome is the real host home directory (e.g. /home/alice, or
	// /var/home/alice on ostree-based systems).
	//
	// customHome is an optional host-side override path for this
	// container's home (empty string means "no override, use hostHome").
	//
	// effectiveHome is what $HOME actually resolves to for THIS container:
	// the canonical /home/<user> path when a custom home is mounted there,
	// otherwise hostHome unchanged. It's computed once here and reused
	// everywhere HOME is needed (the --env HOME= below, the custom-home
	// mount target, and the otter-init --home arg) so it's never
	// set/computed more than once.
	hostHome := userEnv.Home
	effectiveHome := hostHome
	if customHome != "" {
		effectiveHome = fmt.Sprintf("/home/%s", containerUserName)
	}

	var options []string

	if containerPlatform != "" {
		options = append(options, "--platform="+containerPlatform)
	}
	if memory != "" {
		options = append(options, "--memory", memory)
	}
	if cpuThreads > 0 {
		options = append(options, "--cpus", strconv.Itoa(cpuThreads))
	}
	options = append(options, "--hostname", containerHostname)
	options = append(options, "--name", containerName)
	options = append(options, "--privileged")
	options = append(options, "--security-opt", "label=disable")
	options = append(options, "--security-opt", "apparmor=unconfined")
	options = append(options, "--pids-limit=-1")
	options = append(options, "--user", "root:root")

	if !unshareIPC {
		options = append(options, "--ipc", "host")
	}

	if !unshareNetNS {
		options = append(options, "--network", "host")
	}

	if !unshareProcess {
		options = append(options, "--pid", "host")
	}

	// --gpu=nvidia-toolkit delegates GPU access to the NVIDIA Container
	// Toolkit's CDI injection instead of otter's own driver-mirroring
	// (setup_nvidia_gpu.sh, which only runs when --gpu=nvidia is set). No
	// /run/host involvement here. Podman's CDI device syntax differs from
	// Docker/nerdctl's --gpus flag.
	if gpu == "nvidia-toolkit" {
		options = append(options, "--device", "nvidia.com/gpu=all")
	}

	// Mount useful stuff inside the container.
	// We also mount host's root filesystem to /run/host, to be able to syphon
	// dynamic configurations from the host.
	//
	// Mount user home, dev and host's root inside container.
	// This grants access to external devices like usb webcams, disks and so on.
	//
	// Mount also the otter-init utility as the container entrypoint.
	// Also mount in the container the otter-export and otter-host-exec
	// utilities.

	options = append(options, "--label", "manager=otter")
	options = append(options, "--label", "otter.managed_container=1")
	options = append(options, "--label", fmt.Sprintf("otter.unshare_groups=%d", containermanager.Btoi(unshareGroups)))
	options = append(options, "--label", fmt.Sprintf("otter.unshare_ipc=%d", containermanager.Btoi(unshareIPC)))
	options = append(options, "--label", fmt.Sprintf("otter.unshare_netns=%d", containermanager.Btoi(unshareNetNS)))
	options = append(options, "--label", fmt.Sprintf("otter.unshare_process=%d", containermanager.Btoi(unshareProcess)))
	options = append(options, "--label", fmt.Sprintf("otter.unshare_devsys=%d", containermanager.Btoi(unshareDevsys)))
	options = append(options, "--label", fmt.Sprintf("otter.init=%d", containermanager.Btoi(init)))
	options = append(options, "--label", "otter.gpu="+gpu)
	options = append(options, "--label", fmt.Sprintf("otter.rootful=%d", containermanager.Btoi(p.root)))
	options = append(options, "--label", fmt.Sprintf("otter.userns_nolimit=%d", containermanager.Btoi(noUsernsLimit)))
	options = append(options, "--env", fmt.Sprintf("SHELL=%s", shellFilepath))
	options = append(options, "--env", fmt.Sprintf("HOME=%s", effectiveHome))
	options = append(options, "--env", "container=podman")
	options = append(
		options,
		"--env",
		"TERMINFO_DIRS=/usr/share/terminfo:/run/host/usr/share/terminfo",
	)
	options = append(options, "--env", fmt.Sprintf("CONTAINER_ID=%s", containerName))
	options = append(options, "--env", fmt.Sprintf("OTTER_HOST_UID=%s", containerUserUID))
	options = append(options, "--volume", "/tmp:/tmp"+containermanager.BindPropagation())
	options = append(options, "--volume", fmt.Sprintf("%s:%s", otterExportPath, "/usr/lib/otter/scripts/otter-export:ro"))
	options = append(
		options,
		"--volume",
		fmt.Sprintf("%s:%s", otterHostexecPath, "/usr/lib/otter/scripts/otter-host-exec:ro"),
	)
	options = append(
		options,
		"--volume",
		fmt.Sprintf("%s:%s", otterSubreaperPath, "/usr/lib/otter/scripts/otter-subreaper:ro"),
	)
	options = append(options, "--volume", fmt.Sprintf("%s:%s", otterPath, "/usr/bin/otter:ro"))
	if customHome == "" {
		options = append(options, "--volume", fmt.Sprintf("%s:%s%s", hostHome, hostHome, containermanager.BindPropagation()))
	}

	// Due to breaking change in https://github.com/opencontainers/runc/commit/d4b670fca6d0ac606777376440ffe49686ce15f4
	// now we cannot mount /:/run/host as before, as it will try to mount RO partitions as RW thus breaking things.
	// This will ensure we will mount directories one-by-one thus avoiding this problem.
	//
	// This happens ONLY with podman+runc, docker & nerdctl is unaffected,
	// so let's do this only if we have podman AND runc.
	//
	// Note: hostRootMountsForRunc enumerates the host's top-level directories once,
	// at container creation time, and bakes one --volume flag per directory into the
	// container. If a new top-level directory is added under host / afterward (e.g. a
	// new drive mounted at /data), it will NOT be visible inside an existing
	// Podman+runc container at /run/host/data — the container must be recreated to
	// pick it up. Docker/nerdctl/Podman-without-runc, which use a single
	// /:/run/host mount, don't have this limitation.
	if p.usesRunc(ctx) {
		options = append(options, hostRootMountsForRunc(ctx)...)
	} else {
		options = append(options, "--volume", "/:/run/host/"+containermanager.BindPropagation())
	}

	if !unshareDevsys {
		options = append(options, "--volume", "/dev:/dev"+containermanager.BindPropagation())
		options = append(options, "--volume", "/sys:/sys"+containermanager.BindPropagation())
	}

	// This fix is needed so that the container can have a separate devpts instance
	// inside
	// This will mount an empty /dev/pts, and the init will take care of mounting
	// a new devpts with the proper flags set
	// Mounting an empty volume there, is needed in order to ensure that no package
	// manager tries to fiddle with /dev/pts/X that would not be writable by them
	//
	// This implementation is done this way in order to be compatible with both
	// docker and podman
	if !unshareDevsys {
		options = append(options, "--volume", "/dev/pts")
		options = append(options, "--volume", "/dev/null:/dev/ptmx")
	}

	// This fix is needed as on Selinux systems, the host's selinux sysfs directory
	// will be mounted inside the rootless container.
	//
	// This works around this and allows the rootless container to work when selinux
	// policies are installed inside it.
	//
	// Ref. Podman issue 4452:
	//    https://github.com/containers/podman/issues/4452
	if containermanager.PathExists("/sys/fs/selinux") {
		options = append(options, "--volume", "/sys/fs/selinux")
	}

	// This fix is needed as systemd (or journald) will try to set ACLs on this
	// path. For now overlayfs and fuse.overlayfs are not compatible with ACLs
	//
	// This works around this using an unnamed volume so that this path will be
	// mounted with a normal non-overlay FS, allowing ACLs and preventing errors.
	//
	// This work around works in conjunction with otter-init's package manager
	// setups.
	// So that we can use pre/post hooks for package managers to present to the
	// systemd install script a blank path to work with, and mount the host's
	// journal path afterwards.
	options = append(options, "--volume", "/var/log/journal")

	// In some systems, for example using sysvinit, /dev/shm is a symlink
	// to /run/shm, instead of the other way around.
	// Resolve this detecting if /dev/shm is a symlink and mount original
	// source also in the container.
	if containermanager.IsSymlink("/dev/shm") && !unshareIPC {
		realPath, err := filepath.EvalSymlinks("/dev/shm")
		if err == nil {
			options = append(options, "--volume", fmt.Sprintf("%s:%s", realPath, realPath))
		}
	}

	// Ensure support forwarding of RedHat subscription-manager
	// This is needed in order to have a working subscription forwarded into the container,
	// this will ensure that rhel-9-for-x86_64-appstream-rpms and rhel-9-for-x86_64-baseos-rpms repos
	// will be available in the container, so that otter-init will be able to
	// install properly all the dependencies like mesa drivers.
	//
	// /run/secrets is a standard location for RHEL containers, that is being pointed by
	// /etc/rhsm-host by default.
	rhelSubscriptionFiles := []string{
		"/etc/pki/entitlement/:/run/secrets/etc-pki-entitlement:ro",
		"/etc/rhsm/:/run/secrets/rhsm:ro",
		"/etc/yum.repos.d/redhat.repo:/run/secrets/redhat.repo:ro",
	}
	for _, rhelFile := range rhelSubscriptionFiles {
		parts := strings.Split(rhelFile, ":")
		if containermanager.PathExists(parts[0]) {
			options = append(options, "--volume", rhelFile)
		}
	}

	// If we have a custom home to use,
	//	1- export OTTER_HOST_HOME pointing to the default (non-custom) host home
	//	2- export OTTER_CUSTOM_HOME pointing to the real host source of the custom
	//	   home, so that `otter rm --rm-home` can find the correct host path later
	//	3- mount the custom home's host directory at effectiveHome (the canonical
	//	   in-container path)
	// HOME itself was already set to effectiveHome above, so it does not need
	// to be set again here.
	if customHome != "" {
		options = append(options, "--env", fmt.Sprintf("OTTER_HOST_HOME=%s", hostHome))
		options = append(options, "--env", fmt.Sprintf("OTTER_CUSTOM_HOME=%s", customHome))
		options = append(
			options,
			"--volume",
			fmt.Sprintf("%s:%s%s", customHome, effectiveHome, containermanager.BindPropagation()),
		)
	}

	// Mount also the /var/home dir on ostree based systems
	// do this only if $HOME was not already set to /var/home/username, and only
	// when not using a custom home, since a custom home should stay isolated
	// from the real host home entirely.
	if customHome == "" {
		homePath := fmt.Sprintf("/var/home/%s", containerUserName)
		if hostHome != homePath && containermanager.PathExists(homePath) {
			options = append(options, "--volume", fmt.Sprintf("%s:%s%s", homePath, homePath, containermanager.BindPropagation()))
		}
	}

	// Mount also the XDG_RUNTIME_DIR to ensure functionality of the apps.
	// This is skipped in case of initful containers, so that a dedicated
	// systemd user session can be used.
	xdgRuntimeDir := fmt.Sprintf("/run/user/%s", containerUserUID)
	if containermanager.PathExists(xdgRuntimeDir) && !init {
		options = append(options, "--volume", fmt.Sprintf("%s:%s%s", xdgRuntimeDir, xdgRuntimeDir, containermanager.BindPropagation()))
	}

	// These are dynamic configs needed by the container to function properly
	// and integrate with the host
	//
	// We're doing this now instead of inside the init because some distros will
	// have symlinks places for these files that use absolute paths instead of
	// relative paths.
	// This is the bare minimum to ensure connectivity inside the container.
	// These files, will then be kept updated by the main loop every 15 seconds.
	if !unshareNetNS {
		netFiles := []string{
			"/etc/hosts",
			"/etc/resolv.conf",
		}

		// If the user explicitly passed --hostname, treat it as a fixed,
		// intentional choice and never bind-mount /etc/hostname over it —
		// otherwise the container's hostname would silently start tracking
		// the host's hostname if it's ever renamed. Only sync /etc/hostname
		// when the container's hostname was left at its computed default.
		//
		// Previously this was decided by comparing containerHostname against
		// the host's current hostname, which broke when an explicit
		// --hostname happened to coincide with the host's hostname at
		// creation time.
		if !hostnameExplicit {
			hostnameFile := "/etc/hostname"
			if resolved, err := filepath.EvalSymlinks(hostnameFile); err == nil {
				hostnameFile = resolved
			}
			options = append(options, "--volume", fmt.Sprintf("%s:/etc/hostname:ro", hostnameFile))
		}

		for _, netFile := range netFiles {
			if containermanager.PathExists(netFile) {
				options = append(options, "--volume", fmt.Sprintf("%s:%s:ro", netFile, netFile))
			}
		}
	}

	// if nopasswd, then let the init know via a mountpoint
	if nopasswd {
		options = append(options, "--volume", "/dev/null:/run/.nopasswd:ro")
	}

	// Add additional flags
	options = append(options, containerAdditionalFlags...)

	// Add additional volumes
	for _, vol := range containerAdditionalVolumes {
		options = append(options, "--volume", vol)
	}

	// Podman-specific flags
	// If possible, always prefer crun, as it allows keeping original groups.
	// useful for rootless containers.
	if commandExists("crun") {
		options = append(options, "--runtime=crun")
	}
	options = append(options, "--annotation", "run.oci.keep_original_groups=1")
	options = append(options, "--ulimit", "host")

	// For init containers, use podman's systemd support
	if init {
		options = append(options, "--systemd=always")
	}

	// Use keep-id only if going rootless.
	if !p.root {
		if !noUsernsLimit && p.supportsKeepIDSize(ctx, containerImage) {
			options = append(options, "--userns", "keep-id:size=65536")
		} else {
			options = append(options, "--userns", "keep-id")
		}
	}

	// Now execute the entrypoint, refer to `otter-init -h` for instructions
	// containerManager
	// Be aware that entrypoint corresponds to otter-init, the copying of it
	// inside the container is moved to otter-enter, in the start phase.
	// This is done to make init, export and host-exec location independent from
	// the host, and easier to upgrade.
	//
	// We set the entrypoint _before_ running the container image so that
	// we can override any user provided entrypoint if need be
	options = append(options, "--volume", fmt.Sprintf("%s:%s", otterInitPath, "/usr/lib/otter/scripts/otter-init:ro"))
	options = append(
		options,
		"--volume",
		fmt.Sprintf("%s:%s", initScriptsPath, "/usr/lib/otter/scripts/initialization-scripts:ro"),
	)
	options = append(options, "--entrypoint", "/usr/lib/otter/scripts/otter-init")

	// Build the rest of the arguments for otter-init
	//
	// The arguments will be passed to otter-init as the entrypoint
	args := []string{
		"--verbose",
		"--name", containerUserName,
		"--user", containerUserUID,
		"--group", containerUserGID,
		"--home", effectiveHome,
		"--init", strconv.Itoa(containermanager.Btoi(init)),
		"--nvidia", strconv.Itoa(containermanager.Btoi(gpu == "nvidia")),
		"--pre-init-hooks", containerPreInitHook,
		"--additional-packages", strings.Join(containerAdditionalPackages, " "),
	}
	if p.root {
		args = append(args, "--rootful")
	}
	args = append(args, "--", containerInitHook)

	// Final assembly of the command
	// podman create [options] image [args...]
	//nolint:mnd // 2 is fine here, it's "create" and image
	cmd := make([]string, 0, len(options)+len(args)+2)
	cmd = append(cmd, "create")
	cmd = append(cmd, options...)
	cmd = append(cmd, containerImage)
	cmd = append(cmd, args...)

	return cmd
}

func (p *Podman) Exists(ctx context.Context, containerName string) bool {
	args := []string{"inspect", "--type", "container", containerName}
	_, err := p.run(ctx, args, runOptions{})
	return err == nil
}

func (p *Podman) run(ctx context.Context, args []string, opts runOptions) (string, error) {
	command := p.binary
	if p.root {
		args = append([]string{command}, args...)
		command = p.sudoCommand
	}

	cmd := exec.CommandContext(ctx, command, args...)

	if opts.Interactive {
		cmd.Stdout = os.Stdout
		cmd.Stdin = os.Stdin
		cmd.Stderr = os.Stderr

		err := cmd.Run()
		if err != nil {
			return "", fmt.Errorf("error running the interactive command :%w", err)
		}
		return "", nil
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		captured := strings.TrimSpace(stderr.String())
		if captured != "" {
			return "", fmt.Errorf("command execution failed: %s", captured)
		}
		return "", fmt.Errorf("command execution failed: %w", err)
	}
	return stdout.String(), nil
}

func (p *Podman) Enter(
	ctx context.Context,
	options containermanager.EnterOptions,
) error {
	userEnv := userenv.LoadUserEnvironment(ctx)
	user := userEnv.User

	command, config, err := p.generateEnterCommand(
		ctx,
		options.ContainerName,
		options.AdditionalFlags,
		options.NoTTY,
		options.NoWorkDir,
		options.CleanPath,
		options.EmptyEnv,
		options.AddEnv,
	)
	if err != nil {
		return err
	}

	commandArgs := containermanager.BuildCommandArgs(options.CustomCommand, user, options.NoTTY, config.UnshareGroups)

	inspectResult, err := p.InspectContainer(ctx, options.ContainerName)
	if err != nil || inspectResult.ContainerStatus != containermanager.RunningStatus {
		return fmt.Errorf("container '%s' is not running, use 'otter start' first", options.ContainerName)
	}

	runOpt := runOptions{Interactive: true}
	if options.NoTTY {
		runOpt = runOptions{}
	}
	if _, err := p.run(ctx, append(command, commandArgs...), runOpt); err != nil {
		return err
	}

	return nil
}

func (p *Podman) ImageExists(ctx context.Context, imageName string) bool {
	args := []string{"inspect", "--type", "image", "--format", "json", imageName}
	output, err := p.run(ctx, args, runOptions{})
	if err != nil {
		return false
	}

	var inspects []inspectOutput
	if err := json.Unmarshal([]byte(output), &inspects); err != nil {
		return false
	}

	if len(inspects) == 0 {
		return false
	}

	return true
}

func (p *Podman) inspectImage(ctx context.Context, imageID string) (*InspectImageOutput, error) {
	out, err := p.run(ctx, []string{"image", "inspect", "--format", "json", imageID}, runOptions{})
	if err != nil {
		return nil, err
	}
	var images []InspectImageOutput
	if err := json.Unmarshal([]byte(out), &images); err != nil || len(images) == 0 {
		return nil, errors.New("failed to parse image inspect output")
	}
	return &images[0], nil
}

func (p *Podman) ImageLabel(ctx context.Context, imageName, key string) (string, bool) {
	info, err := p.inspectImage(ctx, imageName)
	if err != nil {
		return "", false
	}
	value, ok := info.Config.Labels[key]
	if !ok {
		return "", false
	}
	return value, true
}

func (p *Podman) ImageID(ctx context.Context, imageName string) (string, bool) {
	info, err := p.inspectImage(ctx, imageName)
	if err != nil || info.ID == "" {
		return "", false
	}
	return info.ID, true
}

func (p *Podman) ContainerImageID(ctx context.Context, containerName string) (string, bool) {
	args := []string{"inspect", "--type", "container", "--format", "json", containerName}
	output, err := p.run(ctx, args, runOptions{})
	if err != nil {
		return "", false
	}

	var inspects []inspectOutput
	if err := json.Unmarshal([]byte(output), &inspects); err != nil || len(inspects) == 0 {
		return "", false
	}

	if inspects[0].Image == "" {
		return "", false
	}
	return inspects[0].Image, true
}

func (p *Podman) PullImage(ctx context.Context, imageName string, platform string, out containermanager.PullOutput) error {
	var args []string
	if platform != "" {
		args = []string{"pull", "--platform", platform, imageName}
	} else {
		args = []string{"pull", imageName}
	}

	if out == nil {
		_, err := p.run(ctx, args, runOptions{})
		return err
	}

	command := p.binary
	if p.root {
		args = append([]string{command}, args...)
		command = p.sudoCommand
	}
	cmd := exec.CommandContext(ctx, command, args...)
	if err := containermanager.RunWithPullOutput(cmd, out); err != nil {
		return fmt.Errorf("failed to pull image: %w", err)
	}
	return nil
}

func (p *Podman) RemoveImage(ctx context.Context, imageName string, force bool) error {
	args := []string{"rmi", imageName}
	if force {
		args = []string{"rmi", "--force", imageName}
	}
	_, err := p.run(ctx, args, runOptions{})
	return err
}

func (p *Podman) Remove(
	ctx context.Context,
	containerName string,
	options containermanager.RmOptions,
) error {
	args := []string{"rm"}
	if options.Force {
		args = append(args, "--force")
	}

	args = append(args, []string{"--volumes", containerName}...)

	_, err := p.run(ctx, args, runOptions{})
	if err != nil {
		return fmt.Errorf("error removing the container: %w", err)
	}

	if options.RemoveHome {
		err = os.RemoveAll(options.ContainerHome)
		if err != nil {
			return fmt.Errorf("error removing home directory %s: %w", options.ContainerHome, err)
		}
	}

	return nil
}

func (p *Podman) Stop(ctx context.Context, containerNames []string, force bool) error {
	for _, name := range containerNames {
		inspectResult, err := p.InspectContainer(ctx, name)
		if err != nil {
			return fmt.Errorf("container '%s' not found", name)
		}
		if inspectResult.ContainerStatus == containermanager.PausedStatus {
			if err := p.Unpause(ctx, name); err != nil {
				return err
			}
		}
	}

	args := []string{"stop"}
	if force {
		args = append(args, "-t", "0")
	}
	args = append(args, containerNames...)

	_, err := p.run(ctx, args, runOptions{})
	if err != nil {
		return fmt.Errorf("error stopping containers: %w", err)
	}
	return nil
}

func parsePodmanContainerList(output string) ([]containermanager.Container, error) {
	output = strings.TrimSpace(output)
	if output == "" {
		return nil, nil
	}

	var containers []containermanager.Container

	var pc []podmanContainer
	if err := json.Unmarshal([]byte(output), &pc); err != nil {
		return nil, fmt.Errorf("failed to parse container JSON: %w", err)
	}

	for _, c := range pc {
		name := ""
		if len(c.Names) > 0 {
			name = c.Names[0]
		}

		containers = append(containers, containermanager.Container{
			ID:     c.ID,
			Image:  c.Image,
			Name:   name,
			Status: c.Status,
			Labels: c.Labels,
		})
	}
	return containers, nil
}

func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// supportsKeepIDSize tests whether podman supports the keep-id:size= userns option
// by attempting a quick container run. Older podman versions do not support the size suboption.
func (p *Podman) supportsKeepIDSize(ctx context.Context, image string) bool {
	_, err := p.run(ctx, []string{"run", "--rm", "--userns=keep-id:size=65536", image, "/bin/true"}, runOptions{})
	if err == nil {
		return true
	}
	// If the error mentions "size" as unknown option, the feature is not supported.
	// Other errors (e.g., /bin/true not found exit 127) mean the option itself was accepted.
	return !strings.Contains(err.Error(), "unknown option specified: \"size\"")
}

// usesRunc detects whether podman is configured to use runc as the OCI runtime.
// Equivalent to shell: podman info 2>/dev/null | grep -q runc.
func (p *Podman) usesRunc(ctx context.Context) bool {
	out, err := p.run(ctx, []string{"info"}, runOptions{})
	if err != nil {
		return false
	}

	return strings.Contains(out, "runc")
}

// hostRootMountsForRunc returns per-directory volume mounts for /run/host,
// working around a runc breaking change that prevents mounting / as a whole.
func hostRootMountsForRunc(ctx context.Context) []string {
	var mounts []string

	entries, err := os.ReadDir("/")
	if err != nil {
		return mounts
	}

	for _, entry := range entries {
		// Skip hidden directories (shell glob /* doesn't match dotfiles)
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		rootdir := "/" + entry.Name()

		// Skip symlinks
		info, err := os.Lstat(rootdir)
		if err != nil || info.Mode()&os.ModeSymlink != 0 {
			continue
		}

		target := fmt.Sprintf("%s:/run/host%s", rootdir, rootdir)
		if isMountReadOnly(ctx, rootdir) {
			mounts = append(mounts, "--volume", target+containermanager.ReadOnlyBindPropagation())
		} else {
			mounts = append(mounts, "--volume", target+containermanager.BindPropagation())
		}
	}

	return mounts
}

// isMountReadOnly checks if the given path resides on a read-only mount
// by parsing findmnt output. Equivalent to shell:
// findmnt --notruncate --noheadings --list --output OPTIONS --target "$path" | tr ',' '\n' | grep -q "^ro$".
func isMountReadOnly(ctx context.Context, path string) bool {
	out, err := exec.CommandContext(
		ctx,
		"findmnt", "--notruncate", "--noheadings", "--list",
		"--output", "OPTIONS", "--target", path,
	).Output()
	if err != nil {
		return false
	}

	for _, opt := range strings.Split(string(out), ",") {
		if strings.TrimSpace(opt) == "ro" {
			return true
		}
	}

	return false
}

func (p *Podman) Commit(ctx context.Context, containerID string, tag string) error {
	_, err := p.run(ctx, []string{"container", "commit", containerID, tag}, runOptions{})
	return err
}

func (p *Podman) CopyFromContainer(ctx context.Context, containerName string, srcPath string, destPath string) error {
	_, err := p.run(ctx, []string{"cp", containerName + ":" + srcPath, destPath}, runOptions{})
	return err
}

func (p *Podman) WriteToContainer(ctx context.Context, containerName string, srcPath string, destPath string) error {
	idx := strings.LastIndex(destPath, "/")
	if idx == -1 {
		return fmt.Errorf("invalid destPath %q: must be an absolute path", destPath)
	}
	dir := destPath[:idx]
	if _, err := p.run(ctx, []string{"exec", containerName, "mkdir", "-p", dir}, runOptions{}); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}
	_, err := p.run(ctx, []string{"cp", srcPath, containerName + ":" + destPath}, runOptions{})
	return err
}

func (p *Podman) DeleteFromContainer(ctx context.Context, containerName string, filePath string) error {
	_, err := p.run(ctx, []string{"exec", containerName, "rm", "-rf", filePath}, runOptions{})
	return err
}

func (p *Podman) InspectContainer(ctx context.Context, containerName string) (*containermanager.InspectResult, error) {
	config := containermanager.InspectResult{}
	args := []string{"inspect", "--type", "container", "--format", "json", containerName}
	output, err := p.run(ctx, args, runOptions{})
	if err != nil {
		return nil, err
	}

	var inspects []inspectOutput
	if err := json.Unmarshal([]byte(output), &inspects); err != nil {
		return nil, fmt.Errorf("error unmarshaling json into containerInspect: %w", err)
	}

	if len(inspects) == 0 {
		return nil, errors.New("container not found")
	}

	inspect := inspects[0]
	config.ContainerID = inspect.ID
	config.ContainerStatus = inspect.State.Status
	config.ContainerCreated = inspect.Created
	config.ContainerImage = inspect.Config.Image
	config.ContainerHostname = inspect.Config.Hostname
	imageOut, err := p.inspectImage(ctx, inspect.Image)
	if err == nil {
		config.ContainerPlatform = imageOut.Os + "/" + imageOut.Architecture
	}

	if inspect.HostConfig.Memory > 0 {
		config.Memory = fmt.Sprintf("%dmb", inspect.HostConfig.Memory/1024/1024)
	}
	if inspect.HostConfig.NanoCpus > 0 {
		config.CPUThreads = int(inspect.HostConfig.NanoCpus / 1e9)
	}

	labels := inspect.Config.Labels
	config.UnshareGroups = labels["otter.unshare_groups"] == "1"
	config.UnshareIPC = labels["otter.unshare_ipc"] == "1"
	config.UnshareNetNS = labels["otter.unshare_netns"] == "1"
	config.UnshareProcess = labels["otter.unshare_process"] == "1"
	config.UnshareDevsys = labels["otter.unshare_devsys"] == "1"
	config.Init = labels["otter.init"] == "1"
	config.GPU = labels["otter.gpu"]
	if config.GPU == "" {
		config.GPU = "mesa"
	}
	config.Rootful = labels["otter.rootful"] == "1"
	config.UsernsNoLimit = labels["otter.userns_nolimit"] == "1"

	for _, env := range inspect.Config.Env {
		switch {
		case strings.HasPrefix(env, "HOME="):
			config.ContainerHome = strings.TrimPrefix(env, "HOME=")
		case strings.HasPrefix(env, "OTTER_CUSTOM_HOME="):
			config.ContainerCustomHomeSource = strings.TrimPrefix(env, "OTTER_CUSTOM_HOME=")
		case strings.HasPrefix(env, "PATH="):
			config.ContainerPath = strings.TrimPrefix(env, "PATH=")
		case strings.HasPrefix(env, "SHELL="):
			config.ContainerShell = strings.TrimPrefix(env, "SHELL=")
		}
	}

	return &config, nil
}

func (p *Podman) generateEnterCommand(
	ctx context.Context,
	containerName string,
	additionalFlags string,
	noTTY bool,
	noWorkDir bool,
	cleanPath bool,
	emptyEnv bool,
	addEnv []string,
) ([]string, *containermanager.InspectResult, error) {
	cmd := []string{}

	cmd = append(cmd, "exec")
	if noTTY {
		cmd = append(cmd, "--detach")
	} else {
		cmd = append(cmd, "--interactive")
		cmd = append(cmd, "--detach-keys=")
	}

	containerConfig, err := p.InspectContainer(ctx, containerName)
	if err != nil {
		return nil, nil, fmt.Errorf("container '%s' not found — are you sure it was created?", containerName)
	}
	// User selection
	if containerConfig.UnshareGroups {
		cmd = append(cmd, "--user=root")
	} else {
		userEnv := userenv.LoadUserEnvironment(ctx)
		username := userEnv.User
		cmd = append(cmd, fmt.Sprintf("--user=%s", username))
	}

	// TTY allocation — auto-detect headless mode like the shell version:
	// if stdin or stdout is not a terminal, skip --tty.
	if !noTTY && ttyutil.IsTTY() {
		cmd = append(cmd, "--tty")
	}

	// Working directory
	hostHome := containerConfig.ContainerHome
	if containerConfig.ContainerCustomHomeSource != "" {
		hostHome = containerConfig.ContainerCustomHomeSource
	}
	workdir, err := containermanager.GetWorkDir(hostHome, containerConfig.ContainerHome, noWorkDir)
	if err != nil {
		return nil, nil, fmt.Errorf("error getting the workdir: %w", err)
	}

	cmd = append(cmd, fmt.Sprintf("--workdir=%s", workdir))
	cmd = append(cmd, fmt.Sprintf("--env=PWD=%s", workdir))

	executablePath, err := os.Executable()
	if err != nil {
		return nil, nil, fmt.Errorf("error getting the executable path: %w", err)
	}

	// Environment variables
	cmd = append(cmd, fmt.Sprintf("--env=CONTAINER_ID=%s", containerName))
	cmd = append(cmd, fmt.Sprintf("--env=OTTER_PATH=%s", executablePath))

	if !emptyEnv {
		for _, env := range containermanager.FilterEnvVars() {
			cmd = append(cmd, fmt.Sprintf("--env=%s", env))
		}
	}
	// PATH handling
	containerPaths := containermanager.BuildContainerPath(cleanPath, os.Getenv("PATH"), containerConfig.ContainerPath)
	cmd = append(cmd, fmt.Sprintf("--env=PATH=%s", containerPaths))

	// XDG_DATA_DIRS
	xdgDataDirs := containermanager.BuildXDGPaths("XDG_DATA_DIRS", []string{"/usr/local/share", "/usr/share"})
	cmd = append(cmd, fmt.Sprintf("--env=XDG_DATA_DIRS=%s", xdgDataDirs))

	// XDG_CONFIG_DIRS
	xdgConfigDirs := containermanager.BuildXDGPaths("XDG_CONFIG_DIRS", []string{"/etc/xdg"})
	cmd = append(cmd, fmt.Sprintf("--env=XDG_CONFIG_DIRS=%s", xdgConfigDirs))

	// XDG home directories
	cmd = append(cmd, fmt.Sprintf("--env=XDG_CACHE_HOME=%s/.cache", containerConfig.ContainerHome))
	cmd = append(cmd, fmt.Sprintf("--env=XDG_CONFIG_HOME=%s/.config", containerConfig.ContainerHome))
	cmd = append(cmd, fmt.Sprintf("--env=XDG_DATA_HOME=%s/.local/share", containerConfig.ContainerHome))
	cmd = append(cmd, fmt.Sprintf("--env=XDG_STATE_HOME=%s/.local/state", containerConfig.ContainerHome))

	// Explicit --add-env values are applied last so they take precedence
	// over otter's own auto-computed PATH/XDG_* values above.
	for _, env := range addEnv {
		if strings.Contains(env, "=") {
			cmd = append(cmd, fmt.Sprintf("--env=%s", env))
		} else if val, ok := os.LookupEnv(env); ok {
			cmd = append(cmd, fmt.Sprintf("--env=%s=%s", env, val))
		} else {
			ui.DefaultLogger.Warn("env-var not found on host, skipping", "env", env)
		}
	}

	// Additional flags
	if len(additionalFlags) > 0 {
		cmd = append(cmd, strings.Fields(additionalFlags)...)
	}

	// Container name
	cmd = append(cmd, containerName)

	return cmd, containerConfig, nil
}

func (p *Podman) Pause(ctx context.Context, containerName string) error {
	_, err := p.run(ctx, []string{"pause", containerName}, runOptions{})
	if err != nil {
		return fmt.Errorf("error pausing container '%s': %w", containerName, err)
	}
	return nil
}

func (p *Podman) Unpause(ctx context.Context, containerName string) error {
	_, err := p.run(ctx, []string{"unpause", containerName}, runOptions{})
	if err != nil {
		return fmt.Errorf("error unpausing container '%s': %w", containerName, err)
	}
	return nil
}

func (p *Podman) Start(ctx context.Context, containerName string) error {
	inspectResult, err := p.InspectContainer(ctx, containerName)
	if err != nil {
		return fmt.Errorf("container '%s' not found", containerName)
	}
	if inspectResult.ContainerStatus == containermanager.RunningStatus {
		ui.DefaultLogger.Info("container is already running", "container", containerName)
		return nil
	}
	if inspectResult.ContainerStatus == containermanager.PausedStatus {
		return p.Unpause(ctx, containerName)
	}

	progress := ui.NewProgress(os.Stderr)
	logTimestamp := containermanager.TimestampNow()

	_, err = p.run(ctx, []string{"start", containerName}, runOptions{Interactive: true})
	if err != nil {
		return err
	}

	inspectResult, err = p.InspectContainer(ctx, containerName)
	if err != nil || inspectResult.ContainerStatus != containermanager.RunningStatus {
		logs, err := p.run(ctx, []string{"logs", containerName}, runOptions{})
		if err != nil {
			return fmt.Errorf("could not inspect container logs: %w", err)
		}
		return fmt.Errorf("could not start entrypoint.\n%s", logs)
	}

	progress.Next("starting container...")

	userEnv := userenv.LoadUserEnvironment(ctx)

	cacheDir := os.Getenv("XDG_CACHE_HOME")
	if cacheDir == "" {
		cacheDir = filepath.Join(userEnv.Home, ".cache")
	}
	cacheDir = filepath.Join(cacheDir, "otter")

	if err := os.MkdirAll(cacheDir, 0755); err != nil { //nolint:gosec // we need this writable by everybody
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	if err := p.waitForSetup(ctx, containerName, logTimestamp, progress); err != nil {
		return err
	}

	progress.Finalize("container setup complete!")
	return nil
}

func (p *Podman) waitForSetup(
	ctx context.Context,
	containerName string,
	since string,
	progress *ui.Progress,
) error {
	for {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("podman: %w", err)
		}

		// Check container is still running
		inspectResult, err := p.InspectContainer(ctx, containerName)
		if err != nil {
			return fmt.Errorf("container stopped during setup: %w", err)
		}
		if inspectResult.ContainerStatus != containermanager.RunningStatus {
			return fmt.Errorf(
				"container stopped during setup: status=%s",
				inspectResult.ContainerStatus,
			)
		}

		// Get logs
		output, err := p.run(ctx, []string{"logs", "--since", since, containerName}, runOptions{})
		if err != nil {
			time.Sleep(logsRetryInterval)
			continue
		}
		since = containermanager.TimestampNow()

		lines := strings.Split(output, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)

			switch {
			case strings.HasPrefix(line, "+"):
				// Ignore logging commands
				continue

			case strings.HasPrefix(line, "Error:"):
				progress.Fail()
				return fmt.Errorf("container setup error: %s", line)

			case strings.HasPrefix(line, "Warning:"):
				ui.DefaultLogger.Warn(line)

			case strings.HasPrefix(line, "otter:"):
				parts := strings.SplitN(line, " ", 2)
				if len(parts) > 1 {
					progress.Done()
					progress.Next("%s", parts[1])
				}

			case strings.HasPrefix(line, "container_setup_done"):
				return nil
			}
		}

		time.Sleep(setupPollInterval)
	}
}

func (p *Podman) IsSetupDone(ctx context.Context, containerName string) bool {
	_, err := p.run(ctx, []string{"exec", containerName, "test", "-f", "/usr/lib/otter/container.ready"}, runOptions{})
	return err == nil
}

func (p *Podman) Journal(ctx context.Context, containerName string, opts containermanager.JournalOptions) error {
	args := []string{"logs"}
	if opts.Follow {
		args = append(args, "--follow")
	}
	if opts.Since != "" {
		args = append(args, "--since", opts.Since)
	}
	if opts.Until != "" {
		args = append(args, "--until", opts.Until)
	}
	if opts.Timestamps {
		args = append(args, "--timestamps")
	}
	if opts.Tail >= 0 {
		args = append(args, "--tail", strconv.Itoa(opts.Tail))
	}
	args = append(args, containerName)
	_, err := p.run(ctx, args, runOptions{Interactive: true})
	return err
}
