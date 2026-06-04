package commands

import (
	"bufio"
	"context"
	"embed"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"

	"github.com/ferret-linux/otter/internal/userenv"
	"github.com/ferret-linux/otter/pkg/containermanager"
	"github.com/ferret-linux/otter/pkg/ui"
)

//go:embed assets/desktop_entry.toml.tmpl
var desktopEntryTmpl string

//go:embed assets/terminal-otter-icon.svg
var defaultIconData []byte

//go:embed assets/distros
var distroIconsFS embed.FS

var distroIconMap = map[string]string{
	"ol":                  "oracle-box.svg",
	"arch":                "arch-box.svg",
	"ubuntu":              "ubuntu-box.svg",
	"debian":              "debian-box.svg",
	"fedora":              "fedora-box.svg",
	"alpine":              "alpine-box.svg",
	"kali":                "kali-box.svg",
	"centos":              "centos-box.svg",
	"alma":                "alma-box.svg",
	"rocky":               "rocky-box.svg",
	"gentoo":              "gentoo-box.svg",
	"void":                "void-box.svg",
	"blackarch":           "blackarch-box.svg",
	"opensuse-leap":       "leap-box.svg",
	"opensuse-tumbleweed": "tumbleweed-box.svg",
	"rhel":                "rhel-box.svg",
	"slackware":           "slackware-box.svg",
}

type GenerateEntryOptions struct {
	Verbose             bool
	DryRun              bool
	Delete              bool
	Root                bool
	DesktopEntryBaseDir string
	OtterPath           string
	All                 bool
	Icon                string   // ignored when All=true or multiple names
	ContainerNames      []string // ignored when All=true
}

type GenerateEntryCommand struct {
	listCommand      *ListCommand
	containerManager containermanager.ContainerManager
}

func NewGenerateEntryCommand(listCommand *ListCommand, cm containermanager.ContainerManager) *GenerateEntryCommand {
	return &GenerateEntryCommand{
		listCommand:      listCommand,
		containerManager: cm,
	}
}

func (c *GenerateEntryCommand) Execute(
	ctx context.Context,
	opts *GenerateEntryOptions) error {
	// Determine whether is a single or all entries generation
	// If all is set, fetch the list of all containers
	// If not, use the provided container name or the default one
	var containerNames []string
	var icon string
	switch {
	case opts.All:
		// Generate entries for all containers
		listResult, err := c.listCommand.Execute(ctx)
		if err != nil {
			return fmt.Errorf("failed to list containers: %w", err)
		}

		containerNames = make([]string, 0, len(listResult.Containers))
		for _, container := range listResult.Containers {
			containerNames = append(containerNames, container.Name)
		}
		// Set icon to auto for all entries
		icon = "auto"
	case len(opts.ContainerNames) > 0:
		for _, name := range opts.ContainerNames {
			if _, err := c.containerManager.InspectContainer(ctx, name); err != nil {
				return fmt.Errorf("container '%s' not found", name)
			}
		}
		containerNames = opts.ContainerNames
		if len(opts.ContainerNames) == 1 {
			icon = opts.Icon
		} else {
			icon = "auto"
		}
	default:
		return fmt.Errorf("please specify a container name with --name/-n")
	}

	// Determine the desktop entry base dir
	desktopEntryBaseDir := opts.DesktopEntryBaseDir
	if desktopEntryBaseDir == "" {
		userEnv := userenv.LoadUserEnvironment(ctx)
		desktopEntryBaseDir = userEnv.DesktopEntryBaseDir
	}

	if opts.Delete {
		for _, containerName := range containerNames {
			entryPath := c.getEntryFilePath(filepath.Join(desktopEntryBaseDir, "applications"), containerName, opts.Root)
			if _, err := os.Stat(entryPath); os.IsNotExist(err) {
				ui.DefaultLogger.Info("no desktop entry found for '%s'", containerName)
				continue
			}
			if err := c.deleteEntry(containerName, desktopEntryBaseDir, opts.Root); err != nil {
				return fmt.Errorf("failed to delete desktop entry for container %s: %w", containerName, err)
			}
			ui.DefaultLogger.Ok("desktop entry removed for '%s'", containerName)
		}

		return nil
	}

	// Determine OtterPath
	otterPath := opts.OtterPath
	if otterPath == "" {
		p, err := os.Executable()
		if err != nil {
			return fmt.Errorf("cannot read otter path, %w", err)
		}
		otterPath = p
	}

	// Create the desktop entries for all the containers
	for _, containerName := range containerNames {
		entryPath := c.getEntryFilePath(filepath.Join(desktopEntryBaseDir, "applications"), containerName, opts.Root)
		if opts.Verbose {
			ui.DefaultLogger.Info("writing desktop entry for '%s' to %s", containerName, entryPath)
		}
		if opts.DryRun {
			ui.DefaultLogger.Info("would write desktop entry for '%s' to %s", containerName, entryPath)
			continue
		}
		if err := c.createEntry(ctx, containerName, icon, desktopEntryBaseDir, otterPath, opts.Root); err != nil {
			return fmt.Errorf("failed to create desktop entry for container %s: %w", containerName, err)
		}
		ui.DefaultLogger.Ok("desktop entry created for '%s'", containerName)
	}

	return nil
}

func (c *GenerateEntryCommand) deleteEntry(containerName string, desktopEntryBaseDir string, root bool) error {
	desktopEntryAppsDir := filepath.Join(desktopEntryBaseDir, "applications")
	entryFilePath := c.getEntryFilePath(desktopEntryAppsDir, containerName, root)
	if _, err := os.Stat(entryFilePath); os.IsNotExist(err) {
		return nil
	}
	if err := os.Remove(entryFilePath); err != nil {
		return fmt.Errorf("failed to delete desktop entry for container %s: %w", containerName, err)
	}
	return nil
}

func (c *GenerateEntryCommand) createEntry(
	ctx context.Context,
	containerName string,
	icon string,
	desktopEntryBaseDir string,
	otterPath string,
	root bool,
) error {
	desktopEntryAppsDir, desktopEntryIconsDir, err := c.ensureDesktopEntryDirExists(desktopEntryBaseDir)
	if err != nil {
		return fmt.Errorf("failed to ensure desktop entry directories exist: %w", err)
	}

	entryFilePath := c.getEntryFilePath(desktopEntryAppsDir, containerName, root)
	data := c.composeDesktopEntryData(containerName, c.getIconPath(ctx, containerName, icon, desktopEntryIconsDir), otterPath, root)
	if err := c.writeDesktopEntryFile(entryFilePath, data); err != nil {
		return fmt.Errorf("failed to write desktop entry file for container %s: %w", containerName, err)
	}

	return nil
}

func (c *GenerateEntryCommand) ensureDesktopEntryDirExists(desktopEntryBaseDir string) (string, string, error) {
	desktopEntryAppsDir := filepath.Join(desktopEntryBaseDir, "applications")
	if err := os.MkdirAll(desktopEntryAppsDir, 0750); err != nil {
		return "", "", fmt.Errorf("failed to create desktop entry applications directory: %w", err)
	}
	desktopEntryIconsDir := filepath.Join(desktopEntryBaseDir, "otter", "icons", "distros")
	if err := os.MkdirAll(desktopEntryIconsDir, 0750); err != nil {
		return "", "", fmt.Errorf("failed to create desktop entry icons directory: %w", err)
	}
	return desktopEntryAppsDir, desktopEntryIconsDir, nil
}

// composeDesktopEntry generates the desktop entry for a single container
func (c *GenerateEntryCommand) composeDesktopEntryData(
	containerName string,
	icon string,
	otterPath string,
	root bool,
) map[string]string {
	extraFlags := ""
	if root {
		extraFlags += "--root"
	}

	return map[string]string{
		"entry_name":     getEntryName(containerName, root),
		"container_name": containerName,
		"otter_path":     otterPath,
		"icon":           icon,
		"extra_flags":    extraFlags,
	}
}

// getEntryName returns the formatted entry name for the desktop entry
// based on the container name, capitalizing the first letter.
func getEntryName(containerName string, root bool) string {
	if containerName == "" {
		return ""
	}
	first := strings.ToUpper(containerName[:1])
	var name string
	if len(containerName) > 1 {
		name = first + containerName[1:]
	} else {
		name = first
	}
	if root {
		return name + " (rootful)"
	}
	return name
}

func (c *GenerateEntryCommand) writeDesktopEntryFile(
	entryFilePath string,
	data map[string]string,
) error {
	//nolint:gosec // 644 is common permission for desktop entry files
	destFileWriter, err := os.OpenFile(entryFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to create desktop entry file: %w", err)
	}
	defer destFileWriter.Close()

	t, err := template.New("desktopEntry").Parse(desktopEntryTmpl)
	if err != nil {
		return fmt.Errorf("failed to parse desktop entry template: %w", err)
	}
	err = t.Execute(destFileWriter, data)
	if err != nil {
		return fmt.Errorf("failed to execute desktop entry template: %w", err)
	}

	return nil
}

func (c *GenerateEntryCommand) getEntryFilePath(desktopEntryDir, containerName string, root bool) string {
	if root {
		return filepath.Join(desktopEntryDir, "rootful-"+containerName+".desktop")
	}
	return filepath.Join(desktopEntryDir, containerName+".desktop")
}

// getIconPath returns the local icon path for the desktop entry.
// If icon is "auto", reads /etc/os-release from the container to detect the distro,
// writes the matching embedded SVG to iconsDir, and returns the path.
// Falls back to the generic otter icon silently on any failure.
func (c *GenerateEntryCommand) getIconPath(ctx context.Context, containerName string, icon string, iconsDir string) string {
	if icon != "auto" && icon != "" {
		return icon
	}

	distroID := c.readDistroID(ctx, containerName)
	iconFileName, ok := distroIconMap[distroID]

	var iconData []byte
	var destFileName string

	if ok {
		if data, err := distroIconsFS.ReadFile("assets/distros/" + iconFileName); err == nil {
			iconData = data
			destFileName = iconFileName
		}
	}

	if iconData == nil {
		iconData = defaultIconData
		destFileName = "terminal-otter-icon.svg"
	}

	destPath := filepath.Join(iconsDir, destFileName)
	if _, err := os.Stat(destPath); os.IsNotExist(err) {
		//nolint:gosec // 644 is standard for icon files
		_ = os.WriteFile(destPath, iconData, 0644)
	}

	return destPath
}

// readDistroID copies /etc/os-release from the container and returns the ID= value.
// Returns empty string if detection fails.
func (c *GenerateEntryCommand) readDistroID(ctx context.Context, containerName string) string {
	tmpFile := filepath.Join(os.TempDir(), containerName+".os-release")
	defer os.Remove(tmpFile)

	if err := c.containerManager.CopyFromContainer(ctx, containerName, "/etc/os-release", tmpFile); err != nil {
		return ""
	}

	f, err := os.Open(tmpFile)
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "ID=") {
			id := strings.TrimPrefix(line, "ID=")
			id = strings.Trim(id, "\"'")
			return strings.ToLower(strings.TrimSpace(id))
		}
	}

	_ = scanner.Err()

	return ""
}
