package containermanager

import "strings"

type Container struct {
	ID     string
	Image  string
	Name   string
	Status string
	Labels map[string]string
}

type InspectResult struct {
	ContainerID               string
	ContainerStatus           string
	ContainerHome             string
	ContainerCustomHomeSource string
	ContainerPath             string
	ContainerImage            string
	ContainerPlatform         string
	ContainerCreated          string
	ContainerHostname         string
	ContainerShell            string
	Memory                    string
	CPUThreads                int
	UnshareGroups             bool
	UnshareIPC                bool
	UnshareNetNS              bool
	UnshareProcess            bool
	UnshareDevsys             bool
	Init                      bool
	GPU                       string
	Rootful                   bool
	UsernsNoLimit             bool
}

type CreateOptions struct {
	ContainerName           string
	ContainerImage          string
	ContainerClone          string
	ContainerUserCustomHome string
	ContainerHostname       string
	// ContainerHostnameExplicit is true when the user explicitly passed
	// --hostname, as opposed to ContainerHostname being a computed default.
	// This is the real signal for "is this hostname custom", instead of
	// guessing by comparing the resolved hostname string against the host's
	// current hostname (which breaks if a user's explicit --hostname
	// coincidentally matches the host's hostname at creation time).
	ContainerHostnameExplicit bool
	ContainerPlatform         string
	ContainerShell            string
	ContainerUserHome         string
	Nopasswd                  bool
	UnshareDevsys             bool
	UnshareGroups             bool
	UnshareIPC                bool
	UnshareNetNS              bool
	UnshareProcess            bool
	AdditionalFlags           []string
	AdditionalVolumes         []string
	AdditionalPackages        []string
	ContainerPreInitHook      string
	ContainerInitHook         string
	Init                      bool
	GPU                       string
	NoUsernsLimit             bool
	Memory                    string
	CPUThreads                int
	ScriptsDir                string
}

type EnterOptions struct {
	ContainerName   string
	AdditionalFlags string
	CustomCommand   []string
	AddEnv          []string
	NoTTY           bool
	NoWorkDir       bool
	CleanPath       bool
	EmptyEnv        bool
}

type JournalOptions struct {
	Follow     bool
	Since      string
	Until      string
	Timestamps bool
	Tail       int
}

type RmOptions struct {
	Force         bool
	RemoveHome    bool
	ContainerHome string
}

func (c Container) IsOtterContainer() bool {
	if c.Labels["manager"] == "otter" {
		return true
	}
	// Fallback: manager label was overridden via --additional-flags,
	// check for otter.managed_container which is always set at creation
	return c.Labels["otter.managed_container"] == "1"
}

func (c Container) IsRunning() bool {
	s := strings.ToLower(c.Status)
	if strings.Contains(s, "paused") {
		return false
	}
	return strings.Contains(s, "up") || strings.Contains(s, "running")
}
