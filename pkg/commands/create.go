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
	"github.com/ferret-linux/otter/pkg/ui"
)

const (
	maxHostnameLength = 64
)

var ErrHostnameTooLong = fmt.Errorf("hostname too long, must be less than %d characters", maxHostnameLength)
var ErrImagePullAbortedByUser = errors.New("image pull operation aborted by user")
var ErrUnknownImage = errors.New("unknown image")

var imageAliases = map[string]string{
	"arch":                "docker.io/library/archlinux:latest",
	"alma":                "docker.io/library/almalinux:latest",
	"kali":                "docker.io/kalilinux/kali-rolling:latest",
	"rhel":                "registry.access.redhat.com/ubi10/ubi-init:latest",
	"rocky":               "docker.io/rockylinux/rockylinux:10-ubi-init",
	"fedora":              "quay.io/fedora/fedora:44",
	"centos":              "quay.io/centos/centos:stream10",
	"gentoo":              "docker.io/gentoo/stage3:latest",
	"ubuntu":              "docker.io/library/ubuntu:latest",
	"debian":              "docker.io/library/debian:stable",
	"alpine":              "docker.io/library/alpine:latest",
	"oracle":              "container-registry.oracle.com/os/oraclelinux:10",
	"void-musl":           "ghcr.io/void-linux/void-musl-full:latest",
	"blackarch":           "docker.io/blackarchlinux/blackarch:latest",
	"kali-edge":           "docker.io/kalilinux/kali-bleeding-edge:latest",
	"ubuntu-lts":          "docker.io/library/ubuntu:26.04",
	"void-glibc":          "ghcr.io/void-linux/void-glibc-full:latest",
	"alpine-edge":         "docker.io/library/alpine:edge",
	"opensuse-leap":       "registry.opensuse.org/opensuse/leap:latest",
	"fedora-rawhide":      "quay.io/fedora/fedora:rawhide",
	"debian-testing":      "docker.io/library/debian:testing",
	"debian-unstable":     "docker.io/library/debian:unstable",
	"opensuse-tumbleweed": "registry.opensuse.org/opensuse/tumbleweed:latest",
}

func resolveImage(image string) (string, error) {
	if strings.ContainsAny(image, "/:") {
		return image, nil
	}
	if resolved, ok := imageAliases[strings.ToLower(image)]; ok {
		return resolved, nil
	}
	return "", fmt.Errorf("%w %q, use a full registry path or a valid alias", ErrUnknownImage, image)
}

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
	prompter         *ui.Prompter
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

	Nvidia     bool
	DryRun     bool
	Verbose    bool
	Memory     string
	CPUThreads int

	GenerateEntry bool
	Rootful       bool

	ContainerAlwaysPull bool
	NonInteractive      bool
}

type CreateResult struct {
	ContainerName     string
	ContainerImage    string
	ContainerHostname string
}

func NewCreateCommand(cfg *config.Values, cm containermanager.ContainerManager, progress *ui.Progress, prompter *ui.Prompter) *CreateCommand {
	return &CreateCommand{
		cfg:              cfg,
		containerManager: cm,
		generateEntryCmd: NewGenerateEntryCommand(cfg, NewListCommand(cfg, cm), cm),
		progress:         progress,
		prompter:         prompter,
	}
}

func (c *CreateCommand) Execute(ctx context.Context, opts CreateOptions) (*CreateResult, error) {
	opts.ContainerShell = c.makeContainerShell(&opts)

	if err := validateMemory(opts.Memory); err != nil {
		return nil, err
	}
	if err := validateCPUThreads(opts.CPUThreads); err != nil {
		return nil, err
	}

	containerImage, err := c.makeContainerImage(&opts)
	if err != nil {
		return nil, err
	}
	containerName := c.makeContainerName(&opts, containerImage)
	containerHostname, err := c.makeContainerHostname(&opts)
	if err != nil {
		return nil, err
	}

	containerUserCustomHome := c.makeContainerUserCustomHome(&opts, containerName)

	if !opts.DryRun && c.containerManager.Exists(ctx, containerName) {
		return nil, &ContainerAlreadyExistsError{ContainerName: containerName}
	}

	if opts.ContainerClone != "" && !opts.DryRun {
		cloneImage, err := c.clone(ctx, opts.ContainerClone)
		if err != nil {
			return nil, fmt.Errorf("failed to clone container %s: %w", opts.ContainerClone, err)
		}
		containerImage = cloneImage
	}

	if err := c.askPullImage(ctx, containerImage, opts); err != nil {
		return nil, err
	}

	if opts.Verbose {
		ui.DefaultLogger.Info("creating '%s' from image '%s' with hostname '%s'", containerName, containerImage, containerHostname)
	}

	displayImage := opts.ContainerImage
	if displayImage == "" {
		displayImage = c.cfg.DefaultContainerImage
	}
	c.progress.Next("Creating '%s' using image '%s'", containerName, displayImage)

	err = c.containerManager.Create(
		ctx,
		containermanager.CreateOptions{
			ContainerName:           containerName,
			ContainerImage:          containerImage,
			ContainerClone:          opts.ContainerClone,
			ContainerUserCustomHome: containerUserCustomHome,
			ContainerHostname:       containerHostname,
			ContainerShell:          opts.ContainerShell,
			ContainerPlatform:       opts.ContainerPlatform,
			Nopasswd:                opts.Nopasswd,
			UnshareDevsys:           opts.UnshareDevsys,
			UnshareGroups:           opts.UnshareGroups,
			UnshareIPC:              opts.UnshareIpc,
			UnshareNetNS:            opts.UnshareNetNs,
			UnshareProcess:          opts.UnshareProcess,
			AdditionalFlags:         splitFields(opts.AdditionalFlags),
			AdditionalVolumes:       opts.AdditionalVolumes,
			AdditionalPackages:      opts.AdditionalPackages,
			ContainerPreInitHook:    opts.ContainerPreInitHook,
			ContainerInitHook:       opts.ContainerInitHook,
			Init:                    opts.Init,
			Nvidia:                  opts.Nvidia,
			DryRun:                  opts.DryRun,
			Memory:                  opts.Memory,
			CPUThreads:              opts.CPUThreads,
		},
	)

	if err != nil {
		c.progress.Fail()
		return nil, fmt.Errorf("failed to create container: %w", err)
	}

	c.progress.Done()

	if opts.GenerateEntry && !opts.DryRun && !opts.Rootful {
		err := c.generateEntryCmd.Execute(
			ctx,
			&GenerateEntryOptions{
				ContainerName: containerName,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("failed to generate entry for container %s: %w", containerName, err)
		}
	}

	return &CreateResult{
		ContainerName:     containerName,
		ContainerImage:    containerImage,
		ContainerHostname: containerHostname,
	}, nil
}

// Determine right containerImage to use
//
// If no clone option and no container image, let's choose a default image to use.
//
// If no name is specified and we're using the default container_image, then let's
// set a default name for the container, that is distinguishable from the default
// toolbx one. This will avoid problems when using both toolbx and otter on
// the same system.
func validateMemory(memory string) error {
	if memory == "" {
		return nil
	}
	matched, _ := regexp.MatchString(`^[0-9]+(m|g)$`, memory)
	if !matched {
		return fmt.Errorf("invalid memory format, use m or g suffix (e.g. 512m, 2g)")
	}
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return fmt.Errorf("failed to read memory info: %w", err)
	}
	var totalKB uint64
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "MemTotal:") {
			fmt.Sscanf(line, "MemTotal: %d kB", &totalKB)
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
		return fmt.Errorf("cpu-threads must be greater than 0")
	}
	data, err := os.ReadFile("/sys/devices/system/cpu/present")
	if err != nil {
		return fmt.Errorf("failed to read cpu info: %w", err)
	}
	parts := strings.SplitN(strings.TrimSpace(string(data)), "-", 2)
	if len(parts) != 2 {
		return fmt.Errorf("failed to parse cpu info")
	}
	start, err1 := strconv.Atoi(parts[0])
	end, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return fmt.Errorf("failed to parse cpu info")
	}
	hostThreads := end - start + 1
	if threads > hostThreads {
		return fmt.Errorf("not enough threads, host has max %d threads available", hostThreads)
	}
	return nil
}

func (c *CreateCommand) makeContainerShell(opts *CreateOptions) string {
	if opts.ContainerShell != "" {
		return opts.ContainerShell
	}
	switch filepath.Base(os.Getenv("SHELL")) {
	case "bash", "zsh", "fish":
		return filepath.Base(os.Getenv("SHELL"))
	default:
		return "bash"
	}
}

func (c *CreateCommand) makeContainerImage(opts *CreateOptions) (string, error) {
	containerImage := opts.ContainerImage
	if opts.ContainerClone == "" && containerImage == "" {
		containerImage = c.cfg.DefaultContainerImage
	}
	if opts.DryRun && opts.ContainerClone != "" {
		containerImage = opts.ContainerClone
	}
	if containerImage != "" && opts.ContainerClone == "" {
		resolved, err := resolveImage(containerImage)
		if err != nil {
			return "", err
		}
		containerImage = resolved
	}

	return containerImage, nil
}

// Determine right containerName to use
//
// If no name is specified and no image is specified, then let's
// set a default name for the container, that is distinguishable from the default
// toolbx one. This will avoid problems when using both toolbx and otter on
// the same system.
//
// If no container_name is declared, we build our container name starting from the
// container image specified.
//
// Examples:
//
//	alpine -> alpine
//	ubuntu:20.04 -> ubuntu-20.04
//	registry.fedoraproject.org/fedora-toolbox:39 -> fedora-toolbox-39
//	ghcr.io/void-linux/void-linux:latest-full-x86_64 -> void-linux-latest-full-x86_64
func (c *CreateCommand) makeContainerName(opts *CreateOptions, containerImage string) string {
	containerName := opts.ContainerName
	if containerName == "" && opts.ContainerImage == "" {
		containerName = c.cfg.DefaultContainerName
	}
	if containerName == "" {
		if _, ok := imageAliases[strings.ToLower(opts.ContainerImage)]; ok {
			containerName = "my-" + strings.ToLower(opts.ContainerImage)
		} else {
			base := path.Base(containerImage)
			base = strings.ReplaceAll(base, ":", "-")
			base = strings.ReplaceAll(base, ".", "-")
			containerName = base
		}
	}

	return containerName
}

func (c *CreateCommand) makeContainerHostname(opts *CreateOptions) (string, error) {
	containerHostname := opts.ContainerHostname
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
	containerUserCustomHome := opts.ContainerUserCustomHome
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

	if i.ContainerStatus == "running" {
		return "", errors.New("cannot clone running container, name: " + containerName)
	}

	commitTag := fmt.Sprintf("%s:%s", strings.ToLower(containerName), time.Now().Format("2006-01-02"))

	err = c.containerManager.Commit(ctx, i.ContainerID, commitTag)
	if err != nil {
		return "", fmt.Errorf("failed to commit container '%s:%s': %w", i.ContainerID, commitTag, err)
	}

	return commitTag, nil
}

func (c *CreateCommand) askPullImage(ctx context.Context, containerImage string, opts CreateOptions) error {
	if opts.ContainerAlwaysPull || !c.containerManager.ImageExists(ctx, containerImage) {
		skipConfirm := opts.NonInteractive || opts.ContainerAlwaysPull || opts.DryRun
		if !skipConfirm {
			msg := fmt.Sprintf("Image '%s' not found on your system.\nWould you like to pull it?", containerImage)
			answer := c.prompter.Prompt(msg, true)
			if !answer {
				return ErrImagePullAbortedByUser
			}
		}

		err := c.containerManager.PullImage(ctx, containerImage, opts.ContainerPlatform, opts.DryRun)
		if err != nil {
			return fmt.Errorf("failed to pull image '%s': %w", containerImage, err)
		}
	}

	return nil
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
