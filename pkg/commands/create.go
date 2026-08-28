package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ferret-linux/otter/pkg/config"
	"github.com/ferret-linux/otter/pkg/containermanager"
	"github.com/ferret-linux/otter/pkg/registry"
	"github.com/ferret-linux/otter/pkg/ui"
)

const (
	maxHostnameLength = 64
)

var ErrHostnameTooLong = fmt.Errorf("hostname too long, must be less than %d characters", maxHostnameLength)

type ContainerAlreadyExistsError struct {
	ContainerName string
}

func (e *ContainerAlreadyExistsError) Error() string {
	return fmt.Sprintf("container named '%s' already exists", e.ContainerName)
}

type CreateCommand struct {
	cfg              *config.Values
	containerManager containermanager.ContainerManager
	generateEntryCmd *GenerateEntryCommand
	progress         *ui.Progress
}

type CreateOptions struct {
	// ContainerClone name of the otter container to use as base for a new container
	ContainerClone string
	// ContainerImage image to use for the container
	ContainerImage string
	// ContainerName name of the otter container
	ContainerName string
	// ContainerHostname hostname to set inside the container
	ContainerHostname string
	// ContainerPlatform platform to use for the container (e.g., linux/amd64, linux/arm64)
	ContainerPlatform string
	// ContainerShell custom shell to use inside the container instead of the host shell
	ContainerShell string
	Nopasswd       bool

	// UnshareNetNs if true, do not share host network namespace
	UnshareNetNs bool
	// UnshareDevsys if true, do not share host devices and sysfs dirs from host
	UnshareDevsys bool
	// UnshareGroups if true, do not forward user's additional groups into the container
	UnshareGroups bool
	// UnshareIpc if true, do not share host IPC namespace
	UnshareIpc bool
	// UnshareProcess if true, do not share host process namespace
	UnshareProcess bool

	AdditionalFlags      []string
	AdditionalVolumes    []string
	AdditionalPackages   []string
	ContainerPreInitHook string
	ContainerInitHook    string

	ContainerUserCustomHome string
	ContainerHomePrefix     string
	Init                    bool

	GPU           string
	NoUsernsLimit bool
	Memory        string
	CPUThreads    int

	GenerateEntry bool
	Rootful       bool

	ContainerAlwaysPull bool
}

type CreateResult struct {
	ContainerName     string
	ContainerImage    string
	ContainerHostname string
	AlreadyExisted    bool
}

func NewCreateCommand(cfg *config.Values, cm containermanager.ContainerManager, progress *ui.Progress) *CreateCommand {
	if progress == nil {
		progress = ui.NewProgress(os.Stderr)
	}
	return &CreateCommand{
		cfg:              cfg,
		containerManager: cm,
		generateEntryCmd: NewGenerateEntryCommand(NewListCommand(cm), cm),
		progress:         progress,
	}
}

//nolint:gocognit,funlen // ignore cognitive complexity here, the function orchestrates multi-step container creation
func (c *CreateCommand) Execute(ctx context.Context, opts CreateOptions) (*CreateResult, error) {
	opts.ContainerShell = c.makeContainerShell(&opts)

	if err := validateShell(opts.ContainerShell); err != nil {
		return nil, fmt.Errorf("shell validation failed: %w", err)
	}

	opts.GPU = c.makeGpu(&opts)

	if err := validateGpu(opts.GPU); err != nil {
		return nil, fmt.Errorf("gpu validation failed: %w", err)
	}
	if opts.GPU == "nvidia-toolkit" && !containermanager.NvidiaToolkitAvailable() {
		return nil, errors.New(
			"--gpu=nvidia-toolkit requires the NVIDIA Container Toolkit to be installed " +
				"and a CDI spec generated on the host (nvidia-ctk cdi generate); " +
				"neither nvidia-ctk nor a CDI spec file could be found",
		)
	}
	if err := validateMemory(opts.Memory); err != nil {
		return nil, fmt.Errorf("memory validation failed: %w", err)
	}
	if err := validateCPUThreads(opts.CPUThreads); err != nil {
		return nil, fmt.Errorf("cpu-threads validation failed: %w", err)
	}

	containerImage, imageProps, err := c.makeContainerImage(ctx, &opts)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve container image: %w", err)
	}
	containerName := c.makeContainerName(&opts, containerImage)
	opts.ContainerName = containerName
	containerHostname, err := c.makeContainerHostname(&opts)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve container hostname: %w", err)
	}
	// Capture whether --hostname (or its otter.conf default) was explicitly
	// set before any further defaulting happens, so providers can rely on
	// this instead of comparing the resolved hostname string against the
	// host's current hostname.
	containerHostnameExplicit := opts.ContainerHostname != "" || c.cfg.DefaultHostname != ""

	containerUserCustomHome := c.makeContainerUserCustomHome(&opts, containerName)

	if c.containerManager.Exists(ctx, containerName) {
		printContainerAlreadyExists(c.progress, containerName, opts.Rootful)
		return &CreateResult{ContainerName: containerName, AlreadyExisted: true}, nil
	}

	if opts.ContainerClone != "" {
		cloneImage, err := c.clone(ctx, opts.ContainerClone)
		if err != nil {
			return nil, fmt.Errorf("failed to clone container %s: %w", opts.ContainerClone, err)
		}
		containerImage = cloneImage
	}

	switch {
	case !c.containerManager.ImageExists(ctx, containerImage):
		// Missing image always auto-pulls unconditionally.
		if err := registry.Pull(ctx, c.containerManager, containerImage, opts.ContainerPlatform, c.progress); err != nil {
			return nil, fmt.Errorf("failed to pull image '%s': %w", containerImage, err)
		}
	case opts.ContainerAlwaysPull:
		if err := registry.Pull(ctx, c.containerManager, containerImage, opts.ContainerPlatform, c.progress); err != nil {
			return nil, fmt.Errorf("failed to pull image '%s': %w", containerImage, err)
		}
	case imageProps != nil:
		if err := c.checkImageStaleness(ctx, imageProps, containerImage, opts); err != nil {
			return nil, err
		}
	}

	displayImage := opts.ContainerImage
	if displayImage == "" {
		displayImage = c.cfg.DefaultContainerImage
	}
	c.progress.Next("creating '%s' using image '%s'", containerName, displayImage)

	err = c.containerManager.Create(
		ctx,
		containermanager.CreateOptions{
			ContainerName:             containerName,
			ContainerImage:            containerImage,
			ContainerClone:            opts.ContainerClone,
			ContainerUserCustomHome:   containerUserCustomHome,
			ContainerHostname:         containerHostname,
			ContainerHostnameExplicit: containerHostnameExplicit,
			ContainerShell:            opts.ContainerShell,
			ContainerPlatform:         opts.ContainerPlatform,
			Nopasswd:                  opts.Nopasswd,
			UnshareDevsys:             opts.UnshareDevsys,
			UnshareGroups:             opts.UnshareGroups,
			UnshareIPC:                opts.UnshareIpc,
			UnshareNetNS:              opts.UnshareNetNs,
			UnshareProcess:            opts.UnshareProcess,
			AdditionalFlags:           splitFields(opts.AdditionalFlags),
			AdditionalVolumes:         opts.AdditionalVolumes,
			AdditionalPackages:        opts.AdditionalPackages,
			ContainerPreInitHook:      opts.ContainerPreInitHook,
			ContainerInitHook:         opts.ContainerInitHook,
			Init:                      opts.Init,
			GPU:                       opts.GPU,
			NoUsernsLimit:             opts.NoUsernsLimit,
			Memory:                    opts.Memory,
			CPUThreads:                opts.CPUThreads,
			ScriptsDir:                c.cfg.ScriptsDir,
		},
	)

	if err != nil {
		c.progress.Fail()
		return nil, fmt.Errorf("failed to create container: %w", err)
	}

	c.progress.Done()

	if opts.GenerateEntry {
		err := c.generateEntryCmd.Execute(
			ctx,
			&GenerateEntryOptions{
				ContainerNames: []string{containerName},
				Root:           opts.Rootful,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("failed to generate entry for container %s: %w", containerName, err)
		}
	}

	result := &CreateResult{
		ContainerName:     containerName,
		ContainerImage:    containerImage,
		ContainerHostname: containerHostname,
	}

	printCreateCompleted(c.progress, result.ContainerName, opts.Rootful)

	return result, nil
}

func printCreateCompleted(progress *ui.Progress, containerName string, rootful bool) {
	rootFlag := ""
	if rootful {
		rootFlag = "--root "
	}

	progress.Finalize("'%s' successfully created", containerName)
	ui.DefaultLogger.Info(fmt.Sprintf("to enter, run: otter enter %s%s", rootFlag, containerName))
}

func printContainerAlreadyExists(progress *ui.Progress, containerName string, rootful bool) {
	rootFlag := ""
	if rootful {
		rootFlag = "--root "
	}

	progress.Finalize("container named '%s' already exists", containerName)
	ui.DefaultLogger.Info(fmt.Sprintf("to enter, run: otter enter %s%s", rootFlag, containerName))
}

func validateShell(shell string) error {
	if shell == "" {
		return nil
	}
	switch shell {
	case "bash", "zsh", "fish": //nolint:goconst // shell name literals are self-documenting
		return nil
	default:
		return fmt.Errorf("invalid shell %q, must be one of: bash, zsh, fish", shell)
	}
}

func validateGpu(gpu string) error {
	switch gpu {
	case "", "mesa", "nvidia", "nvidia-toolkit":
		return nil
	default:
		return fmt.Errorf("invalid gpu %q, must be one of: mesa, nvidia, nvidia-toolkit", gpu)
	}
}

func validateMemory(memory string) error {
	if memory == "" {
		return nil
	}
	matched, _ := regexp.MatchString(`^[0-9]+(m|g)$`, memory)
	if !matched {
		return errors.New("invalid memory format, use m or g suffix (e.g. 512m, 2g)")
	}
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return fmt.Errorf("failed to read memory info: %w", err)
	}
	var totalKB uint64
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "MemTotal:") {
			if _, err := fmt.Sscanf(line, "MemTotal: %d kB", &totalKB); err != nil {
				return fmt.Errorf("failed to parse MemTotal: %w", err)
			}
			break
		}
	}
	value, _ := strconv.ParseUint(memory[:len(memory)-1], 10, 64)
	if memory[len(memory)-1] == 'g' {
		value *= 1024 * 1024
		if value > totalKB {
			return fmt.Errorf("not enough memory, host has %dg", totalKB/1024/1024)
		}
	} else {
		value *= 1024
		if value > totalKB {
			return fmt.Errorf("not enough memory, host has %dm", totalKB/1024)
		}
	}
	return nil
}

func validateCPUThreads(threads int) error {
	if threads == 0 {
		return nil
	}
	if threads < 0 {
		return errors.New("cpu-threads must be greater than 0")
	}
	data, err := os.ReadFile("/sys/devices/system/cpu/present")
	if err != nil {
		return fmt.Errorf("failed to read cpu info: %w", err)
	}
	parts := strings.SplitN(strings.TrimSpace(string(data)), "-", 2)
	if len(parts) != 2 {
		return errors.New("failed to parse cpu info")
	}
	start, err1 := strconv.Atoi(parts[0])
	end, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return errors.New("failed to parse cpu info")
	}
	hostThreads := end - start + 1
	if threads > hostThreads {
		return fmt.Errorf("not enough threads, host has max %d threads available", hostThreads)
	}
	return nil
}

func (c *CreateCommand) makeGpu(opts *CreateOptions) string {
	if opts.GPU != "" {
		return opts.GPU
	}
	return "mesa"
}

func (c *CreateCommand) makeContainerShell(opts *CreateOptions) string {
	if opts.ContainerShell != "" {
		return opts.ContainerShell
	}
	if c.cfg.DefaultShell != "" {
		return c.cfg.DefaultShell
	}
	switch filepath.Base(os.Getenv("SHELL")) {
	case "bash", "zsh", "fish":
		return filepath.Base(os.Getenv("SHELL"))
	default:
		return "bash"
	}
}

// Determine right containerImage to use
//
// If no clone option and no container image, let's choose a default image to use.
func (c *CreateCommand) makeContainerImage(ctx context.Context, opts *CreateOptions) (string, *registry.ImagesProperties, error) {
	containerImage := opts.ContainerImage
	if opts.ContainerClone == "" && containerImage == "" {
		containerImage = c.cfg.DefaultContainerImage
	}
	if containerImage != "" && opts.ContainerClone == "" {
		if strings.ContainsAny(containerImage, "/:") {
			return containerImage, nil, nil
		}
		props, err := registry.Fetch(ctx)
		if err != nil {
			return "", nil, fmt.Errorf("failed to fetch registry properties: %w", err)
		}
		resolved, err := registry.Resolve(props, containerImage)
		if err != nil {
			return "", nil, fmt.Errorf("failed to resolve image '%s': %w", containerImage, err)
		}
		containerImage = resolved
		return containerImage, props, nil
	}

	return containerImage, nil, nil
}

// checkImageStaleness compares the locally present image against the latest
// known remote build and warns or auto-pulls per the configured thresholds.
// Only called when the image already exists locally and ContainerAlwaysPull
// is not set.
func (c *CreateCommand) checkImageStaleness(
	ctx context.Context,
	props *registry.ImagesProperties,
	containerImage string,
	opts CreateOptions,
) error {
	st := registry.CheckStaleness(ctx, c.containerManager, props, containerImage)

	switch st.State {
	case registry.StalenessBehind:
		switch {
		case c.cfg.StalenessAutopullThreshold > 0 && st.Diff >= c.cfg.StalenessAutopullThreshold:
			if err := registry.Pull(ctx, c.containerManager, containerImage, opts.ContainerPlatform, c.progress); err != nil {
				return fmt.Errorf("failed to pull image '%s': %w", containerImage, err)
			}
		case c.cfg.StalenessWarnThreshold > 0 && st.Diff >= c.cfg.StalenessWarnThreshold:
			ui.DefaultLogger.Warn("image is behind upstream", "image", ui.TrimImageRef(containerImage), "local", st.LocalBuild, "latest", st.RemoteBuild)
		}
	case registry.StalenessAhead:
		ui.DefaultLogger.Warn("image is ahead of upstream", "image", ui.TrimImageRef(containerImage), "local", st.LocalBuild, "latest", st.RemoteBuild)
	case registry.StalenessUnknown:
		// Local build label missing or unreadable — treat like a missing
		// image and pull to heal, no special-casing for pre-feature images.
		if err := registry.Pull(ctx, c.containerManager, containerImage, opts.ContainerPlatform, c.progress); err != nil {
			return fmt.Errorf("failed to pull image '%s': %w", containerImage, err)
		}
	case registry.StalenessCurrent, registry.StalenessNotOtterImage:
		// nothing to do
	}

	return nil
}

// Determine right containerName to use
//
// If no name is specified and no image is specified, then let's
// set a default name for the container, that is distinguishable from the default
// toolbx one. This will avoid problems when using both toolbx and otter on
// the same system.
//
// If no container_name is declared, we build our container name starting from the
// container image specified. Short image names (no registry path or tag) get a
// "my-" prefix so multiple containers built from bare distro names remain
// distinguishable from the default toolbx naming; images with an explicit
// registry path or tag are derived directly from their last path segment.
//
// Examples:
//
//	alpine -> my-alpine
//	ubuntu:20.04 -> ubuntu-20.04
//	registry.fedoraproject.org/fedora-toolbox:39 -> fedora-toolbox-39
//	ghcr.io/void-linux/void-linux:latest-full-x86_64 -> void-linux-latest-full-x86_64
func (c *CreateCommand) makeContainerName(opts *CreateOptions, containerImage string) string {
	containerName := opts.ContainerName
	if containerName == "" && opts.ContainerImage == "" {
		containerName = c.cfg.DefaultContainerName
	}
	if containerName == "" {
		if !strings.ContainsAny(opts.ContainerImage, "/:") {
			containerName = "my-" + strings.ToLower(opts.ContainerImage)
		} else {
			base := path.Base(containerImage)
			base = strings.ReplaceAll(base, ":", "-")
			containerName = base
		}
	}

	return containerName
}

func (c *CreateCommand) makeContainerHostname(opts *CreateOptions) (string, error) {
	containerHostname := opts.ContainerHostname
	if containerHostname == "" {
		containerHostname = c.cfg.DefaultHostname
	}
	if containerHostname == "" {
		hostname, err := os.Hostname()
		if err != nil {
			return "", fmt.Errorf("unable to get hostname: %w", err)
		}
		containerHostname = hostname
		if opts.UnshareNetNs {
			containerHostname = fmt.Sprintf("%s.%s", opts.ContainerName, hostname)
		}
	}

	if len(containerHostname) > maxHostnameLength {
		return "", ErrHostnameTooLong
	}

	return containerHostname, nil
}

// Determine right containerUserCustomHome to use
//
// We check if the user has a custom home prefix to use for the container home.
// If we have a home prefix to use, and no custom home set, then we set
// the custom home to be PREFIX/CONTAINER_NAME
func (c *CreateCommand) makeContainerUserCustomHome(
	opts *CreateOptions,
	containerName string,
) string {
	containerUserCustomHome := strings.TrimRight(opts.ContainerUserCustomHome, "/")
	if opts.ContainerHomePrefix != "" && containerUserCustomHome == "" {
		containerUserCustomHome = filepath.Join(opts.ContainerHomePrefix, containerName)
	}
	return containerUserCustomHome
}

func (c *CreateCommand) clone(ctx context.Context, containerName string) (string, error) {
	i, err := c.containerManager.InspectContainer(ctx, containerName)
	if err != nil {
		return "", fmt.Errorf("failed to inspect container status: %w", err)
	}

	if i.ContainerStatus == containermanager.RunningStatus {
		return "", errors.New("cannot clone running container, name: " + containerName)
	}

	commitTag := fmt.Sprintf("%s:%s", strings.ToLower(containerName), time.Now().Format("2006-01-02"))

	err = c.containerManager.Commit(ctx, i.ContainerID, commitTag)
	if err != nil {
		return "", fmt.Errorf("failed to commit container '%s:%s': %w", i.ContainerID, commitTag, err)
	}

	return commitTag, nil
}

// splitFields word-splits each entry of in on whitespace. A single CLI
// invocation like --additional-flags "--label=a=1 --label=b=2" arrives here as
// one slice element with embedded spaces; podman would parse that as a single
// flag value. Splitting matches the manifest parser's behavior.
func splitFields(in []string) []string {
	out := make([]string, 0, len(in))
	for _, v := range in {
		out = append(out, strings.Fields(v)...)
	}
	return out
}
