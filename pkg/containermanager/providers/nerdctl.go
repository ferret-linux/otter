//nolint:goconst // CLI flag strings are intentionally repeated per-provider; they may diverge independently
package providers

import (
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

type Nerdctl struct {
	binary      string
	root        bool
	sudoCommand string
}

var _ containermanager.ContainerManager = &Nerdctl{}

func NewNerdctl(root bool, sudoCommand string) *Nerdctl {
	binary, err := exec.LookPath("nerdctl")
	if err != nil {
		binary = "nerdctl"
	}
	return &Nerdctl{
		binary:      binary,
		sudoCommand: sudoCommand,
		root:        root,
	}
}

func (n *Nerdctl) CloneAsRoot() containermanager.ContainerManager {
	cp := *n
	cp.root = true
	return &cp
}

func (n *Nerdctl) Name() string {
	return "nerdctl"
}

func (n *Nerdctl) ListContainers(ctx context.Context) ([]containermanager.Container, error) {
	args := []string{"ps", "-a", "--no-trunc", "--format", "json"}
	out, err := n.run(ctx, args, runOptions{})
	if err != nil {
		return nil, err
	}
	return parseContainerList(out)
}

func (n *Nerdctl) Create(
	ctx context.Context,
	opts containermanager.CreateOptions,
) error {
	userEnv := userenv.LoadUserEnvironment(ctx)

	scriptsDir, _, err := insidecontainer.ProvisionScripts(opts.ScriptsDir)
	if err != nil {
		return fmt.Errorf("failed to provision scripts: %w", err)
	}

	if opts.ContainerUserCustomHome != "" && !containermanager.PathExists(opts.ContainerUserCustomHome) {
		//nolint:gosec // 0755 is the same as from distrobox v1, let's keep it for compatibility
		if err := os.MkdirAll(opts.ContainerUserCustomHome, 0755); err != nil {
			return fmt.Errorf("failed to create custom home directory: %w", err)
		}
	}

	userEnv.Shell = opts.ContainerShell

	cmd := n.makeCreateCommand(
		opts,
		userEnv,
		filepath.Join(scriptsDir, "otter-init"),
		filepath.Join(scriptsDir, "otter-export"),
		filepath.Join(scriptsDir, "otter-host-exec"),
		filepath.Join(scriptsDir, "otter"),
		filepath.Join(scriptsDir, "otter-subreaper"),
		filepath.Join(scriptsDir, "otter-doctor"),
		filepath.Join(scriptsDir, "otter-redirect"),
		filepath.Join(scriptsDir, "initialization-scripts"),
	)

	_, err = n.run(ctx, cmd, runOptions{})
	if err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}
	return nil
}

//nolint:gocognit,funlen,gocyclo,cyclop // ignore cognitive complexity here, the function is mostly imperative option appending
func (n *Nerdctl) makeCreateCommand(
	opts containermanager.CreateOptions,
	userEnv *userenv.UserEnvironment,
	otterInitPath string,
	otterExportPath string,
	otterHostexecPath string,
	otterPath string,
	otterSubreaperPath string,
	otterDoctorPath string,
	otterRedirectPath string,
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
	// /run/host involvement here.
	if gpu == "nvidia-toolkit" {
		options = append(options, "--gpus", "all")
	}

	options = append(options, "--label", "manager=otter")
	options = append(options, "--label", "otter.managed_container=1")
	options = append(options, "--label", fmt.Sprintf("otter.unshare_groups=%d", containermanager.Btoi(unshareGroups)))
	options = append(options, "--label", fmt.Sprintf("otter.unshare_ipc=%d", containermanager.Btoi(unshareIPC)))
	options = append(options, "--label", fmt.Sprintf("otter.unshare_netns=%d", containermanager.Btoi(unshareNetNS)))
	options = append(options, "--label", fmt.Sprintf("otter.unshare_process=%d", containermanager.Btoi(unshareProcess)))
	options = append(options, "--label", fmt.Sprintf("otter.unshare_devsys=%d", containermanager.Btoi(unshareDevsys)))
	options = append(options, "--label", fmt.Sprintf("otter.init=%d", containermanager.Btoi(init)))
	options = append(options, "--label", "otter.gpu="+gpu)
	options = append(options, "--label", fmt.Sprintf("otter.rootful=%d", containermanager.Btoi(n.root)))
	options = append(options, "--env", fmt.Sprintf("SHELL=%s", shellFilepath))
	options = append(options, "--env", fmt.Sprintf("HOME=%s", effectiveHome))
	options = append(options, "--env", "container=nerdctl")
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
	options = append(
		options,
		"--volume",
		fmt.Sprintf("%s:%s", otterDoctorPath, "/usr/lib/otter/scripts/otter-doctor:ro"),
	)
	options = append(
		options,
		"--volume",
		fmt.Sprintf("%s:%s", otterRedirectPath, "/usr/lib/otter/scripts/otter-redirect:ro"),
	)
	options = append(options, "--volume", fmt.Sprintf("%s:%s", otterPath, "/usr/bin/otter:ro"))
	if customHome == "" {
		options = append(options, "--volume", fmt.Sprintf("%s:%s%s", hostHome, hostHome, containermanager.BindPropagation()))
	}
	options = append(options, "--volume", "/:/run/host/"+containermanager.BindPropagation())

	if !unshareDevsys {
		options = append(options, "--volume", "/dev:/dev"+containermanager.BindPropagation())
		options = append(options, "--volume", "/sys:/sys"+containermanager.BindPropagation())
	}

	if init {
		options = append(options, "--cgroupns", "host")
		options = append(options, "--stop-signal", "SIGRTMIN+3")
		options = append(options, "--mount", "type=tmpfs,destination=/run")
		options = append(options, "--mount", "type=tmpfs,destination=/run/lock")
		options = append(options, "--mount", "type=tmpfs,destination=/var/lib/journal")
	}

	if !unshareDevsys {
		options = append(options, "--volume", "/dev/pts")
		options = append(options, "--volume", "/dev/null:/dev/ptmx")
	}

	if containermanager.PathExists("/sys/fs/selinux") {
		options = append(options, "--volume", "/sys/fs/selinux")
	}

	options = append(options, "--volume", "/var/log/journal")

	if containermanager.IsSymlink("/dev/shm") && !unshareIPC {
		realPath, err := filepath.EvalSymlinks("/dev/shm")
		if err == nil {
			options = append(options, "--volume", fmt.Sprintf("%s:%s", realPath, realPath))
		}
	}

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

	xdgRuntimeDir := fmt.Sprintf("/run/user/%s", containerUserUID)
	if containermanager.PathExists(xdgRuntimeDir) && !init {
		options = append(options, "--volume", fmt.Sprintf("%s:%s%s", xdgRuntimeDir, xdgRuntimeDir, containermanager.BindPropagation()))
	}

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

	if nopasswd {
		options = append(options, "--volume", "/dev/null:/run/.nopasswd:ro")
	}

	// nerdctl's default runtime (runc) has no run.oci.keep_original_groups
	// equivalent (that's a crun-specific annotation), so device-node access
	// via supplementary groups needs the same explicit forwarding as docker.
	if groups, err := os.Getgroups(); err == nil {
		for _, gid := range groups {
			options = append(options, "--group-add", strconv.Itoa(gid))
		}
	}

	options = append(options, containerAdditionalFlags...)

	for _, vol := range containerAdditionalVolumes {
		options = append(options, "--volume", vol)
	}

	options = append(options, "--volume", fmt.Sprintf("%s:%s", otterInitPath, "/usr/lib/otter/scripts/otter-init:ro"))
	options = append(
		options,
		"--volume",
		fmt.Sprintf("%s:%s", initScriptsPath, "/usr/lib/otter/scripts/initialization-scripts:ro"),
	)
	options = append(options, "--entrypoint", "/usr/lib/otter/scripts/otter-init")

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
	if n.root {
		args = append(args, "--rootful")
	}
	args = append(args, "--", containerInitHook)

	//nolint:mnd // 2 is fine here, it's "create" and image
	cmd := make([]string, 0, len(options)+len(args)+2)
	cmd = append(cmd, "create")
	cmd = append(cmd, options...)
	cmd = append(cmd, containerImage)
	cmd = append(cmd, args...)

	return cmd
}

func (n *Nerdctl) Exists(ctx context.Context, containerName string) bool {
	args := []string{"container", "inspect", containerName}
	_, err := n.run(ctx, args, runOptions{})
	return err == nil
}

func (n *Nerdctl) ImageExists(ctx context.Context, imageName string) bool {
	args := []string{"image", "inspect", "--format", "json", imageName}
	output, err := n.run(ctx, args, runOptions{})
	if err != nil {
		return false
	}

	var inspects []inspectOutput
	if err := json.Unmarshal([]byte(output), &inspects); err != nil {
		return false
	}

	return len(inspects) > 0
}

func (n *Nerdctl) inspectImage(ctx context.Context, imageID string) (*InspectImageOutput, error) {
	out, err := n.run(ctx, []string{"image", "inspect", "--format", "json", imageID}, runOptions{})
	if err != nil {
		return nil, err
	}
	var images []InspectImageOutput
	if err := json.Unmarshal([]byte(out), &images); err != nil || len(images) == 0 {
		return nil, errors.New("failed to parse image inspect output")
	}
	return &images[0], nil
}

func (n *Nerdctl) ImageLabel(ctx context.Context, imageName, key string) (string, bool) {
	info, err := n.inspectImage(ctx, imageName)
	if err != nil {
		return "", false
	}
	value, ok := info.Config.Labels[key]
	if !ok {
		return "", false
	}
	return value, true
}

func (n *Nerdctl) ImageID(ctx context.Context, imageName string) (string, bool) {
	info, err := n.inspectImage(ctx, imageName)
	if err != nil || info.ID == "" {
		return "", false
	}
	return info.ID, true
}

func (n *Nerdctl) ContainerImageID(ctx context.Context, containerName string) (string, bool) {
	args := []string{"container", "inspect", "--format", "json", containerName}
	output, err := n.run(ctx, args, runOptions{})
	if err != nil {
		return "", false
	}

	var inspect inspectOutput
	// nerdctl container inspect returns a single object, not an array.
	if err := json.Unmarshal([]byte(output), &inspect); err != nil || inspect.Image == "" {
		return "", false
	}

	return inspect.Image, true
}

func (n *Nerdctl) PullImage(ctx context.Context, imageName string, platform string, out containermanager.PullOutput) error {
	var args []string
	if platform != "" {
		args = []string{"pull", "--platform", platform, imageName}
	} else {
		args = []string{"pull", imageName}
	}

	if out == nil {
		_, err := n.run(ctx, args, runOptions{})
		return err
	}

	command := n.binary
	if n.root {
		args = append([]string{command}, args...)
		command = n.sudoCommand
	}
	cmd := exec.CommandContext(ctx, command, args...)
	if err := containermanager.RunWithPullOutput(cmd, out); err != nil {
		return fmt.Errorf("failed to pull image: %w", err)
	}
	return nil
}

func (n *Nerdctl) RemoveImage(ctx context.Context, imageName string, force bool) error {
	args := []string{"rmi", imageName}
	if force {
		args = []string{"rmi", "--force", imageName}
	}
	_, err := n.run(ctx, args, runOptions{})
	return err
}

func (n *Nerdctl) Remove(
	ctx context.Context,
	containerName string,
	options containermanager.RmOptions,
) error {
	args := []string{"rm"}
	if options.Force {
		args = append(args, "--force")
	}

	args = append(args, []string{"--volumes", containerName}...)

	_, err := n.run(ctx, args, runOptions{})
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

func (n *Nerdctl) Stop(ctx context.Context, containerNames []string, force bool) error {
	for _, name := range containerNames {
		inspectResult, err := n.InspectContainer(ctx, name)
		if err != nil {
			return fmt.Errorf("container '%s' not found", name)
		}
		if inspectResult.ContainerStatus == containermanager.PausedStatus {
			if err := n.Unpause(ctx, name); err != nil {
				return err
			}
		}
	}

	args := []string{"stop"}
	if force {
		args = append(args, "-t", "0")
	}
	args = append(args, containerNames...)

	_, err := n.run(ctx, args, runOptions{})
	if err != nil {
		return fmt.Errorf("error stopping containers: %w", err)
	}
	return nil
}

func (n *Nerdctl) Commit(ctx context.Context, containerID string, tag string) error {
	_, err := n.run(ctx, []string{"container", "commit", containerID, tag}, runOptions{})
	return err
}

func (n *Nerdctl) CopyFromContainer(ctx context.Context, containerName string, srcPath string, destPath string) error {
	_, err := n.run(ctx, []string{"cp", containerName + ":" + srcPath, destPath}, runOptions{})
	return err
}

func (n *Nerdctl) WriteToContainer(ctx context.Context, containerName string, srcPath string, destPath string) error {
	idx := strings.LastIndex(destPath, "/")
	if idx == -1 {
		return fmt.Errorf("invalid destPath %q: must be an absolute path", destPath)
	}
	dir := destPath[:idx]
	if _, err := n.run(ctx, []string{"exec", containerName, "mkdir", "-p", dir}, runOptions{}); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}
	_, err := n.run(ctx, []string{"cp", srcPath, containerName + ":" + destPath}, runOptions{})
	return err
}

func (n *Nerdctl) DeleteFromContainer(ctx context.Context, containerName string, filePath string) error {
	_, err := n.run(ctx, []string{"exec", containerName, "rm", "-rf", filePath}, runOptions{})
	return err
}

func (n *Nerdctl) InspectContainer(ctx context.Context, containerName string) (*containermanager.InspectResult, error) {
	config := containermanager.InspectResult{}
	args := []string{"container", "inspect", "--format", "json", containerName}
	output, err := n.run(ctx, args, runOptions{})
	if err != nil {
		return nil, err
	}

	var inspect inspectOutput
	// nerdctl container inspect returns a single object, not an array
	if err := json.Unmarshal([]byte(output), &inspect); err != nil {
		return nil, fmt.Errorf("error unmarshaling json into containerInspect: %w", err)
	}

	if inspect.ID == "" {
		return nil, errors.New("container not found")
	}
	config.ContainerID = inspect.ID
	config.ContainerStatus = inspect.State.Status
	config.ContainerCreated = inspect.Created
	config.ContainerImage = inspect.Config.Image
	config.ContainerHostname = inspect.Config.Hostname
	imageOut, err := n.inspectImage(ctx, inspect.Image)
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

func (n *Nerdctl) ListContainersByName(ctx context.Context) ([]containermanager.Container, error) {
	return n.ListContainers(ctx)
}

func (n *Nerdctl) Enter(
	ctx context.Context,
	options containermanager.EnterOptions,
) error {
	userEnv := userenv.LoadUserEnvironment(ctx)
	user := userEnv.User

	command, config, err := n.generateEnterCommand(
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

	inspectResult, err := n.InspectContainer(ctx, options.ContainerName)
	if err != nil || inspectResult.ContainerStatus != containermanager.RunningStatus {
		return fmt.Errorf("container '%s' is not running, use 'otter start' first", options.ContainerName)
	}

	runOpt := runOptions{Interactive: true}
	if options.NoTTY {
		runOpt = runOptions{}
	}
	if _, err := n.run(ctx, append(command, commandArgs...), runOpt); err != nil {
		return err
	}

	return nil
}

func (n *Nerdctl) run(ctx context.Context, args []string, opts runOptions) (string, error) {
	command := n.binary
	if n.root {
		args = append([]string{command}, args...)
		command = n.sudoCommand
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

	var stdout, stderr strings.Builder
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

func (n *Nerdctl) generateEnterCommand(
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
		// nerdctl exec has no --detach-keys equivalent (unlike docker/podman);
		// see nerdctl's own docs: "Unimplemented docker exec flags: --detach-keys".
	}

	containerConfig, err := n.InspectContainer(ctx, containerName)
	if err != nil {
		return nil, nil, fmt.Errorf("container '%s' not found — are you sure it was created?", containerName)
	}

	if containerConfig.UnshareGroups {
		cmd = append(cmd, "--user=root")
	} else {
		userEnv := userenv.LoadUserEnvironment(ctx)
		cmd = append(cmd, fmt.Sprintf("--user=%s", userEnv.User))
	}

	if !noTTY && ttyutil.IsTTY() {
		cmd = append(cmd, "--tty")
	}

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

	cmd = append(cmd, fmt.Sprintf("--env=CONTAINER_ID=%s", containerName))
	cmd = append(cmd, fmt.Sprintf("--env=OTTER_PATH=%s", executablePath))

	if !emptyEnv {
		for _, env := range containermanager.FilterEnvVars() {
			cmd = append(cmd, fmt.Sprintf("--env=%s", env))
		}
	}

	// PATH is deliberately left unset here so the exec inherits the
	// container's own native PATH (needed by BuildCommandArgs's
	// getent/cut shell resolution below, which only exists under the
	// image's own package-manager-specific paths on some images, e.g.
	// Nix). The host's PATH is passed separately as HOST_PATH and merged
	// in by otter_profile.sh/otter_config.fish once an interactive shell
	// actually starts, so host CLI tools stay reachable without
	// clobbering the container's own PATH first. --clean-path means "no
	// host paths added", so HOST_PATH is omitted entirely in that case.
	if !cleanPath {
		cmd = append(cmd, fmt.Sprintf("--env=HOST_PATH=%s", os.Getenv("PATH")))
	}

	xdgDataDirs := containermanager.BuildXDGPaths("XDG_DATA_DIRS", []string{"/usr/local/share", "/usr/share"})
	cmd = append(cmd, fmt.Sprintf("--env=XDG_DATA_DIRS=%s", xdgDataDirs))

	xdgConfigDirs := containermanager.BuildXDGPaths("XDG_CONFIG_DIRS", []string{"/etc/xdg"})
	cmd = append(cmd, fmt.Sprintf("--env=XDG_CONFIG_DIRS=%s", xdgConfigDirs))

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

	if len(additionalFlags) > 0 {
		cmd = append(cmd, strings.Fields(additionalFlags)...)
	}

	cmd = append(cmd, containerName)

	return cmd, containerConfig, nil
}

func (n *Nerdctl) Pause(ctx context.Context, containerName string) error {
	_, err := n.run(ctx, []string{"pause", containerName}, runOptions{})
	if err != nil {
		return fmt.Errorf("error pausing container '%s': %w", containerName, err)
	}
	return nil
}

func (n *Nerdctl) Unpause(ctx context.Context, containerName string) error {
	_, err := n.run(ctx, []string{"unpause", containerName}, runOptions{})
	if err != nil {
		return fmt.Errorf("error unpausing container '%s': %w", containerName, err)
	}
	return nil
}

func (n *Nerdctl) Start(ctx context.Context, containerName string) error {
	inspectResult, err := n.InspectContainer(ctx, containerName)
	if err != nil {
		return fmt.Errorf("container '%s' not found", containerName)
	}
	if inspectResult.ContainerStatus == containermanager.RunningStatus {
		ui.DefaultLogger.Info("container is already running", "container", containerName)
		return nil
	}
	if inspectResult.ContainerStatus == containermanager.PausedStatus {
		return n.Unpause(ctx, containerName)
	}

	progress := ui.NewProgress(os.Stderr)
	logTimestamp := containermanager.TimestampNow()

	_, err = n.run(ctx, []string{"start", containerName}, runOptions{Interactive: true})
	if err != nil {
		return err
	}

	inspectResult, err = n.InspectContainer(ctx, containerName)
	if err != nil || inspectResult.ContainerStatus != containermanager.RunningStatus {
		logs, err := n.run(ctx, []string{"logs", containerName}, runOptions{})
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

	if err := n.waitForSetup(ctx, containerName, logTimestamp, progress); err != nil {
		return err
	}

	progress.Finalize("container setup complete!")
	return nil
}

func (n *Nerdctl) waitForSetup(
	ctx context.Context,
	containerName string,
	since string,
	progress *ui.Progress,
) error {
	for {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("nerdctl: %w", err)
		}

		inspectResult, err := n.InspectContainer(ctx, containerName)
		if err != nil {
			return fmt.Errorf("container stopped during setup: %w", err)
		}
		if inspectResult.ContainerStatus != containermanager.RunningStatus {
			return fmt.Errorf(
				"container stopped during setup: status=%s",
				inspectResult.ContainerStatus,
			)
		}

		output, err := n.run(ctx, []string{"logs", "--since", since, containerName}, runOptions{})
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

func (n *Nerdctl) IsSetupDone(ctx context.Context, containerName string) bool {
	_, err := n.run(ctx, []string{"exec", containerName, "test", "-f", "/usr/lib/otter/container.ready"}, runOptions{})
	return err == nil
}

func (n *Nerdctl) Journal(ctx context.Context, containerName string, opts containermanager.JournalOptions) error {
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
	_, err := n.run(ctx, args, runOptions{Interactive: true})
	return err
}
