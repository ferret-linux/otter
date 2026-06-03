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

	insideContainer "github.com/ferret-linux/otter/internal/inside-container"
	"github.com/ferret-linux/otter/internal/userenv"
	"github.com/ferret-linux/otter/pkg/containermanager"
	"github.com/ferret-linux/otter/pkg/ui"
)

type Nerdctl struct {
	binary      string
	root        bool
	sudoCommand string
	verbose     bool
}

var _ containermanager.ContainerManager = &Nerdctl{}

func NewNerdctl(root bool, sudoCommand string, verbose bool) *Nerdctl {
	binary, err := exec.LookPath("nerdctl")
	if err != nil {
		binary = "nerdctl"
	}
	return &Nerdctl{
		binary:      binary,
		sudoCommand: sudoCommand,
		root:        root,
		verbose:     verbose,
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

	scriptsDir, _, err := insideContainer.ProvisionScripts()
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
		opts.ContainerName,
		opts.ContainerImage,
		opts.AdditionalFlags,
		opts.ContainerHostname,
		opts.AdditionalPackages,
		opts.AdditionalVolumes,
		opts.ContainerUserCustomHome,
		opts.ContainerPlatform,
		opts.Nopasswd,
		opts.Init,
		opts.ContainerPreInitHook,
		opts.ContainerInitHook,
		opts.Nvidia,
		opts.Memory,
		opts.CPUThreads,
		opts.UnshareDevsys,
		opts.UnshareGroups,
		opts.UnshareIPC,
		opts.UnshareNetNS,
		opts.UnshareProcess,
		userEnv,
		filepath.Join(scriptsDir, "otter-init"),
		filepath.Join(scriptsDir, "otter-export"),
		filepath.Join(scriptsDir, "otter-host-exec"),
	)

	_, err = n.run(ctx, cmd, runOptions{DryRun: opts.DryRun})
	if err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}
	return nil
}

//nolint:gocognit,funlen // ignore cognitive complexity here, the function is mostly imperative option appending
func (n *Nerdctl) makeCreateCommand(
	containerName string,
	containerImage string,
	containerAdditionalFlags []string,
	containerHostname string,
	containerAdditionalPackages []string,
	containerAdditionalVolumes []string,
	containerUserCustomHome string,
	containerPlatform string,
	nopasswd bool,
	init bool,
	containerPreInitHook string,
	containerInitHook string,
	nvidia bool,
	memory string,
	cpuThreads int,
	unshareDevsys bool,
	unshareGroups bool,
	unshareIPC bool,
	unshareNetNS bool,
	unshareProcess bool,
	userEnv *userenv.UserEnvironment,
	otterInitPath string,
	otterExportPath string,
	otterHostexecPath string,
) []string {
	containerUserHome := userEnv.Home
	containerUserName := userEnv.User
	containerUserUID := userEnv.UserID
	containerUserGID := userEnv.GroupID
	shellFilepath := filepath.Base(userEnv.Shell)

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

	options = append(options, "--label", "manager=otter")
	options = append(options, "--label", "otter.managed_container=1")
	options = append(
		options,
		"--label",
		fmt.Sprintf("otter.unshare_groups=%d", containermanager.Btoi(unshareGroups)),
	)
	options = append(options, "--env", fmt.Sprintf("SHELL=%s", shellFilepath))
	options = append(options, "--env", fmt.Sprintf("HOME=%s", containerUserHome))
	options = append(options, "--env", "container=nerdctl")
	options = append(
		options,
		"--env",
		"TERMINFO_DIRS=/usr/share/terminfo:/run/host/usr/share/terminfo",
	)
	options = append(options, "--env", fmt.Sprintf("CONTAINER_ID=%s", containerName))
	options = append(options, "--volume", "/tmp:/tmp:rslave")
	options = append(options, "--volume", fmt.Sprintf("%s:%s", otterExportPath, "/usr/bin/otter-export:ro"))
	options = append(
		options,
		"--volume",
		fmt.Sprintf("%s:%s", otterHostexecPath, "/usr/bin/otter-host-exec:ro"),
	)
	options = append(options, "--volume", fmt.Sprintf("%s:%s:rslave", containerUserHome, containerUserHome))
	options = append(options, "--volume", "/:/run/host/:rslave")

	if !unshareDevsys {
		options = append(options, "--volume", "/dev:/dev:rslave")
		options = append(options, "--volume", "/sys:/sys:rslave")
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

	if containerUserCustomHome != "" {
		options = append(options, "--env", fmt.Sprintf("HOME=%s", containerUserCustomHome))
		options = append(options, "--env", fmt.Sprintf("OTTER_HOST_HOME=%s", containerUserHome))
		options = append(
			options,
			"--volume",
			fmt.Sprintf("%s:%s:rslave", containerUserCustomHome, containerUserCustomHome),
		)
	}

	homePath := fmt.Sprintf("/var/home/%s", containerUserName)
	if containerUserHome != homePath && containermanager.PathExists(homePath) {
		options = append(options, "--volume", fmt.Sprintf("%s:%s:rslave", homePath, homePath))
	}

	xdgRuntimeDir := fmt.Sprintf("/run/user/%s", containerUserUID)
	if containermanager.PathExists(xdgRuntimeDir) && !init {
		options = append(options, "--volume", fmt.Sprintf("%s:%s:rslave", xdgRuntimeDir, xdgRuntimeDir))
	}

	if !unshareNetNS {
		netFiles := []string{
			"/etc/hosts",
			"/etc/resolv.conf",
		}

		hostname, _ := os.Hostname()
		if containerHostname == hostname {
			hostnameFile := "/etc/hostname"
			if real, err := filepath.EvalSymlinks(hostnameFile); err == nil {
				hostnameFile = real
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

	options = append(options, containerAdditionalFlags...)

	for _, vol := range containerAdditionalVolumes {
		options = append(options, "--volume", vol)
	}

	options = append(options, "--volume", fmt.Sprintf("%s:%s", otterInitPath, "/usr/bin/entrypoint:ro"))
	options = append(options, "--entrypoint", "/usr/bin/entrypoint")

	homeToUse := containerUserHome
	if containerUserCustomHome != "" {
		homeToUse = containerUserCustomHome
	}
	args := []string{
		"--verbose",
		"--name", containerUserName,
		"--user", containerUserUID,
		"--group", containerUserGID,
		"--home", homeToUse,
		"--init", strconv.Itoa(containermanager.Btoi(init)),
		"--nvidia", strconv.Itoa(containermanager.Btoi(nvidia)),
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

	var inspect inspectOutput
	if err := json.Unmarshal([]byte(output), &inspect); err != nil {
		return false
	}

	return inspect.ID != ""
}

func (n *Nerdctl) PullImage(ctx context.Context, imageName string, platform string, dryRun bool) error {
	var args []string
	if platform != "" {
		args = []string{"pull", "--platform", platform, imageName}
	} else {
		args = []string{"pull", imageName}
	}
	_, err := n.run(ctx, args, runOptions{DryRun: dryRun})
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

	_, err := n.run(ctx, args, runOptions{DryRun: options.DryRun})
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

func (n *Nerdctl) Stop(ctx context.Context, containerNames []string, dryRun bool) error {
	args := []string{"stop"}
	args = append(args, containerNames...)

	_, err := n.run(ctx, args, runOptions{DryRun: dryRun})
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
	dir := destPath[:strings.LastIndex(destPath, "/")]
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

	if v, ok := inspect.Config.Labels["otter.unshare_groups"]; ok && v == "1" {
		config.UnshareGroups = true
	}

	for _, env := range inspect.Config.Env {
		if strings.HasPrefix(env, "HOME=") {
			config.ContainerHome = strings.TrimPrefix(env, "HOME=")
		} else if strings.HasPrefix(env, "PATH=") {
			config.ContainerPath = strings.TrimPrefix(env, "PATH=")
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
		options.Verbose,
	)
	if err != nil {
		return err
	}

	commandArgs := containermanager.BuildCommandArgs(options.CustomCommand, user, options.NoTTY, config.UnshareGroups)

	if options.DryRun {
		command = append(command, commandArgs...)
		ui.DefaultLogger.Info("%s %s", n.Name(), strings.Join(command, "\n"))

		return nil
	}

	inspectResult, err := n.InspectContainer(ctx, options.ContainerName)
	if err != nil || inspectResult.ContainerStatus != containermanager.RunningStatus {
		return fmt.Errorf("container '%s' is not running, use 'otter start' first", options.ContainerName)
	}

	runOpt := runOptions{Interactive: true}
	if options.NoTTY {
		runOpt = runOptions{Detach: true}
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

	if opts.DryRun {
		ui.DefaultLogger.Info("%s %s", command, strings.Join(args, " "))
		return "", nil
	}

	if n.verbose {
		ui.DefaultLogger.Info("%s %s", command, strings.Join(args, " "))
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
	verbose bool,
) ([]string, *containermanager.InspectResult, error) {
	cmd := []string{}

	if verbose {
		cmd = append(cmd, "--debug")
	}

	cmd = append(cmd, "exec")
	if noTTY {
		cmd = append(cmd, "--detach")
	} else {
		cmd = append(cmd, "--interactive")
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

	if !noTTY && containermanager.IsTTY() {
		cmd = append(cmd, "--tty")
	}

	workdir, err := containermanager.GetWorkDir(containerConfig.ContainerHome, noWorkDir)
	if err != nil {
		return nil, nil, fmt.Errorf("error getting the workdir: %w", err)
	}

	cmd = append(cmd, fmt.Sprintf("--workdir=%s", workdir))

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
	for _, env := range addEnv {
		if strings.Contains(env, "=") {
			cmd = append(cmd, fmt.Sprintf("--env=%s", env))
		} else if val, ok := os.LookupEnv(env); ok {
			cmd = append(cmd, fmt.Sprintf("--env=%s=%s", env, val))
		} else {
			ui.DefaultLogger.Warn("env-var '%s' not found on host, skipping", env)
		}
	}

	containerPaths := containermanager.BuildContainerPath(cleanPath, os.Getenv("PATH"), containerConfig.ContainerPath)
	cmd = append(cmd, fmt.Sprintf("--env=PATH=%s", containerPaths))

	xdgDataDirs := containermanager.BuildXDGPaths("XDG_DATA_DIRS", []string{"/usr/local/share", "/usr/share"})
	cmd = append(cmd, fmt.Sprintf("--env=XDG_DATA_DIRS=%s", xdgDataDirs))

	xdgConfigDirs := containermanager.BuildXDGPaths("XDG_CONFIG_DIRS", []string{"/etc/xdg"})
	cmd = append(cmd, fmt.Sprintf("--env=XDG_CONFIG_DIRS=%s", xdgConfigDirs))

	cmd = append(cmd, fmt.Sprintf("--env=XDG_CACHE_HOME=%s/.cache", containerConfig.ContainerHome))
	cmd = append(cmd, fmt.Sprintf("--env=XDG_CONFIG_HOME=%s/.config", containerConfig.ContainerHome))
	cmd = append(cmd, fmt.Sprintf("--env=XDG_DATA_HOME=%s/.local/share", containerConfig.ContainerHome))
	cmd = append(cmd, fmt.Sprintf("--env=XDG_STATE_HOME=%s/.local/state", containerConfig.ContainerHome))

	if len(additionalFlags) > 0 {
		cmd = append(cmd, strings.Fields(additionalFlags)...)
	}

	cmd = append(cmd, containerName)

	return cmd, containerConfig, nil
}

func (n *Nerdctl) Start(ctx context.Context, containerName string, dryRun bool) error {
	if dryRun {
		ui.DefaultLogger.Info("%s start %s", n.Name(), containerName)
		return nil
	}

	inspectResult, err := n.InspectContainer(ctx, containerName)
	if err != nil {
		return fmt.Errorf("container '%s' not found", containerName)
	}
	if inspectResult.ContainerStatus == containermanager.RunningStatus {
		ui.DefaultLogger.Info("container '%s' is already running", containerName)
		return nil
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

	progress.Next("Starting container...")

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

	progress.Finalize("Container Setup Complete!")
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
			ui.DefaultLogger.Error("Container Setup Failure!")
			return fmt.Errorf("container stopped during setup: %w", err)
		}
		if inspectResult.ContainerStatus != containermanager.RunningStatus {
			ui.DefaultLogger.Error("Container Setup Failure!")
			return fmt.Errorf(
				"container stopped during setup: status=%s",
				inspectResult.ContainerStatus,
			)
		}

		nextSince := containermanager.TimestampNow()
		output, err := n.run(ctx, []string{"logs", "--since", since, containerName}, runOptions{})
		if err != nil {
			time.Sleep(logsRetryInterval)
			continue
		}
		since = nextSince

		lines := strings.Split(output, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)

			switch {
			case strings.HasPrefix(line, "+"):
				continue

			case strings.HasPrefix(line, "Error:"):
				progress.Fail()
				ui.DefaultLogger.Error("%s", line)
				return fmt.Errorf("container setup error: %s", line)

			case strings.HasPrefix(line, "Warning:"):
				ui.DefaultLogger.Warn("%s", line)

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
		args = append(args, "--tail", fmt.Sprintf("%d", opts.Tail))
	}
	args = append(args, containerName)
	_, err := n.run(ctx, args, runOptions{Interactive: true})
	return err
}
