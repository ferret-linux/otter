package commands

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"syscall"

	"github.com/ferret-linux/otter/internal/userenv"
	"github.com/ferret-linux/otter/pkg/containermanager"
	"github.com/ferret-linux/otter/pkg/containermanager/providers"
	"github.com/ferret-linux/otter/pkg/registry"
)

const (
	cleanupPinMarkerKey   = "X-Otter-Pin"
	cleanupPinMarkerValue = "v1:"
	cleanupImageBuildKey  = "otter.image_build"
	cleanupAutodetect     = "autodetect"
	cleanupOfficialRoot   = "ghcr.io/ferret-linux/"
	cleanupActionRemove   = "remove"
	cleanupManagerDocker  = "docker"
	cleanupManagerNerdctl = "nerdctl"
	cleanupManagerPodman  = "podman"
	cleanupStateUnknown   = "unknown"
)

var (
	//nolint:gochecknoglobals // immutable lookup table
	cleanupSupportedManagers = []string{cleanupManagerPodman, cleanupManagerNerdctl, cleanupManagerDocker}
	cleanupCanonicalPinRe    = regexp.MustCompile(`^(?:(rootful)-)?([a-z0-9]+)-(.+)\.desktop$`)
)

// CleanupOptions controls the cleanup sweep. Apply is true for a normal run
// and false only for --no-apply.
type CleanupOptions struct {
	Apply       bool
	JSON        bool
	AllManagers bool
	Images      bool
	Root        bool
	ScanPaths   []string
	ManagerType string
	SudoCommand string
}

type CleanupItem struct {
	Category string `json:"category"`
	Scope    string `json:"scope"`
	Path     string `json:"path,omitempty"`
	Reason   string `json:"reason"`
	Action   string `json:"action"`
}

type CleanupProblem struct {
	Severity string `json:"severity"`
	Category string `json:"category"`
	Scope    string `json:"scope"`
	Path     string `json:"path,omitempty"`
	Message  string `json:"message"`
	Hard     bool   `json:"hard"`
}

type CleanupScopeState struct {
	Manager string `json:"manager"`
	Root    bool   `json:"root"`
	State   string `json:"state"`
}

type CleanupResult struct {
	Mode     string              `json:"mode"`
	Items    []CleanupItem       `json:"items"`
	Problems []CleanupProblem    `json:"problems"`
	Scopes   []CleanupScopeState `json:"scopes"`
	Removed  int                 `json:"removed"`
}

// HasHardProblems reports whether cleanup must exit nonzero.
func (r *CleanupResult) HasHardProblems() bool {
	if r == nil {
		return false
	}
	for _, problem := range r.Problems {
		if problem.Hard {
			return true
		}
	}
	return false
}

type CleanupCommand struct{}

func NewCleanupCommand() *CleanupCommand {
	return &CleanupCommand{}
}

type cleanupScope struct {
	manager  string
	root     bool
	cm       containermanager.ContainerManager
	state    string
	images   []containermanager.ImageSummary
	usedIDs  map[string]struct{}
	allNames map[string]struct{}
	allIDs   map[string]struct{}
}

type cleanupAction struct {
	CleanupItem

	apply func() error
	key   string
}

type cleanupPlanner struct {
	ctx          context.Context
	opts         CleanupOptions
	paths        *exportPaths
	scopes       []*cleanupScope
	primary      string
	actions      map[string]cleanupAction
	problems     []CleanupProblem
	unknown      bool
	manualReview map[string]struct{}
}

type rawExportRecord struct {
	Record     ExportRecord
	RecordFile string
}

func (c *CleanupCommand) Execute(ctx context.Context, opts CleanupOptions) (*CleanupResult, error) {
	paths, err := newExportPaths(ctx)
	if err != nil {
		return nil, err
	}

	planner := &cleanupPlanner{
		ctx:          ctx,
		opts:         opts,
		paths:        paths,
		actions:      make(map[string]cleanupAction),
		manualReview: make(map[string]struct{}),
	}
	planner.buildScopes()
	planner.planExports()
	planner.planPins()
	planner.planIcons()
	planner.planImages()

	result := &CleanupResult{
		Mode:     "plan",
		Problems: planner.problems,
	}
	if opts.Apply {
		result.Mode = "apply"
	}
	for _, scope := range planner.scopes {
		result.Scopes = append(result.Scopes, CleanupScopeState{
			Manager: scope.manager,
			Root:    scope.root,
			State:   scope.state,
		})
	}

	keys := make([]string, 0, len(planner.actions))
	for key := range planner.actions {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, key := range keys {
		result.Items = append(result.Items, planner.actions[key].CleanupItem)
	}

	if opts.Apply {
		lockErr := paths.withCleanupLock(ctx, func() error {
			for _, key := range keys {
				action := planner.actions[key]
				if err := action.apply(); err != nil {
					result.Problems = append(result.Problems, CleanupProblem{
						Severity: "error",
						Category: action.Category,
						Scope:    action.Scope,
						Path:     action.Path,
						Message:  err.Error(),
						Hard:     true,
					})
					continue
				}
				result.Removed++
			}
			removeEmptyDirs(paths.indexRoot)
			return nil
		})
		if lockErr != nil {
			result.Problems = append(result.Problems, CleanupProblem{
				Severity: "error",
				Category: "lock",
				Scope:    "host",
				Path:     paths.lockPath,
				Message:  lockErr.Error(),
				Hard:     true,
			})
		}
	}

	return result, nil
}

func (p *cleanupPlanner) problem(severity, category, scope, path, message string, hard bool) {
	p.problems = append(p.problems, CleanupProblem{
		Severity: severity,
		Category: category,
		Scope:    scope,
		Path:     path,
		Message:  message,
		Hard:     hard,
	})
}

func (p *cleanupPlanner) note(category, scope, path, message string) {
	p.problem("note", category, scope, path, message, false)
}

func (p *cleanupPlanner) fail(category, scope, path, message string) {
	p.problem("error", category, scope, path, message, true)
}

func (p *cleanupPlanner) addAction(
	item CleanupItem,
	info os.FileInfo,
	cm containermanager.ContainerManager,
	imageID string,
	guard func() error,
) {
	if _, exists := p.actions[item.Category+"|"+item.Path]; exists {
		return
	}
	action := cleanupAction{CleanupItem: item, key: item.Category + "|" + item.Path}
	switch item.Category {
	case "image":
		action.apply = func() error {
			if guard != nil {
				if err := guard(); err != nil {
					return err
				}
			}
			if info, ok := cm.InspectImage(p.ctx, imageID); !ok || info.ID == "" {
				return errors.New("image vanished before removal")
			}
			return cm.RemoveImage(p.ctx, imageID, false)
		}
	default:
		action.apply = func() error { return safeRemove(item.Path, info) }
	}
	p.actions[action.key] = action
}

func safeRemove(path string, expected os.FileInfo) error {
	if expected == nil {
		if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("remove candidate: %w", err)
		}
		return nil
	}
	file, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("open for removal: %w", err)
	}
	current, err := file.Stat()
	_ = file.Close()
	if err != nil {
		return fmt.Errorf("stat before removal: %w", err)
	}
	if !os.SameFile(expected, current) {
		return errors.New("candidate changed since scan")
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("remove candidate: %w", err)
	}
	return nil
}

// withCleanupLock serialises the apply phase with other export operations.
// Plan mode intentionally never takes this lock and never creates its parent
// directories. When otter runs through an escalator the lock is handed back to
// the target user so a later rootless run can still open it.
func (p *exportPaths) withCleanupLock(ctx context.Context, fn func() error) error {
	if err := os.MkdirAll(filepath.Dir(p.lockPath), 0o750); err != nil {
		return fmt.Errorf("failed to create export index directory: %w", err)
	}
	file, err := os.OpenFile(p.lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return fmt.Errorf("failed to open export lock: %w", err)
	}
	defer file.Close()

	if os.Geteuid() == 0 {
		env := userenv.LoadTargetUserEnvironment(ctx)
		uid, uidErr := strconv.Atoi(env.UserID)
		gid, gidErr := strconv.Atoi(env.GroupID)
		if uidErr == nil && gidErr == nil {
			if err := os.Chown(p.lockPath, uid, gid); err != nil {
				return fmt.Errorf("failed to hand export lock to target user: %w", err)
			}
		}
	}

	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("failed to lock export index: %w", err)
	}
	defer func() {
		_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
	}()

	return fn()
}

func (p *cleanupPlanner) buildScopes() {
	sudoCommand := p.opts.SudoCommand

	managerTypes := cleanupSupportedManagers
	if !p.opts.AllManagers {
		managerType := p.opts.ManagerType
		if managerType == "" || managerType == cleanupAutodetect {
			managerType = firstInstalledManager()
		} else if !slices.Contains(cleanupSupportedManagers, managerType) {
			p.fail("manager", "host", "", fmt.Sprintf("invalid container manager %q", managerType))
			return
		}
		if managerType == "" {
			p.scopes = append(p.scopes, &cleanupScope{manager: cleanupStateUnknown, root: p.opts.Root, state: "empty"})
			p.note("manager", cleanupStateUnknown, "", "no container manager installed; manager-backed checks are empty")
			return
		}
		managerTypes = []string{managerType}
		p.primary = managerType
	}

	for _, managerType := range managerTypes {
		if p.opts.Root {
			p.scopes = append(p.scopes, p.buildScope(managerType, true, sudoCommand))
		} else {
			p.scopes = append(p.scopes, &cleanupScope{manager: managerType, root: true, state: "out-of-scope"})
		}
		p.scopes = append(p.scopes, p.buildScope(managerType, false, ""))
	}

	for _, scope := range p.scopes {
		if scope.state == cleanupStateUnknown {
			p.unknown = true
		}
	}
}

func firstInstalledManager() string {
	for _, manager := range cleanupSupportedManagers {
		if _, err := exec.LookPath(manager); err == nil {
			return manager
		}
	}
	return ""
}

func (p *cleanupPlanner) buildScope(managerType string, root bool, sudoCommand string) *cleanupScope {
	scope := &cleanupScope{
		manager:  managerType,
		root:     root,
		state:    "ok",
		usedIDs:  make(map[string]struct{}),
		allNames: make(map[string]struct{}),
		allIDs:   make(map[string]struct{}),
	}
	if _, err := exec.LookPath(managerType); err != nil {
		scope.state = "empty"
		return scope
	}
	cm, err := providers.NewByType(managerType, root, sudoCommand)
	if err != nil {
		scope.state = cleanupStateUnknown
		p.fail("manager", scopeLabel(managerType, root), "", err.Error())
		return scope
	}
	scope.cm = cm

	containers, err := cm.ListContainers(p.ctx)
	if err != nil {
		scope.state = cleanupStateUnknown
		p.fail("manager", scopeLabel(managerType, root), "", fmt.Sprintf("failed to list containers: %v", err))
		return scope
	}
	for _, container := range containers {
		scope.allNames[container.Name] = struct{}{}
		id, ok := cm.ContainerImageID(p.ctx, container.Name)
		if !ok {
			scope.state = cleanupStateUnknown
			p.fail("manager", scopeLabel(managerType, root), container.Name, "failed to resolve frozen container image ID")
			continue
		}
		scope.usedIDs[id] = struct{}{}
		scope.allIDs[id] = struct{}{}
	}

	if p.opts.Images && scope.state == "ok" {
		scope.images, err = cm.ListImages(p.ctx)
		if err != nil {
			scope.state = cleanupStateUnknown
			p.fail("image", scopeLabel(managerType, root), "", fmt.Sprintf("failed to list images: %v", err))
		}
	}
	return scope
}

func scopeLabel(manager string, root bool) string {
	if root {
		return "rootful-" + manager
	}
	return "rootless-" + manager
}

func (p *cleanupPlanner) scopeAllows(manager string, root bool) bool {
	if root != p.opts.Root {
		return false
	}
	if p.opts.AllManagers {
		return slices.Contains(cleanupSupportedManagers, manager)
	}
	return p.primary != "" && manager == p.primary
}

// hasQueryableScope requires a successfully queried engine scope before an
// owner can be acted on. A missing or failed manager never authorises deletion.
func (p *cleanupPlanner) hasQueryableScope(manager string, root bool) bool {
	for _, scope := range p.scopes {
		if scope.manager == manager && scope.root == root && scope.state == "ok" {
			return true
		}
	}
	return false
}

func (p *cleanupPlanner) liveIdentity(name, id string) bool {
	for _, scope := range p.scopes {
		if scope.state != "ok" {
			continue
		}
		if name != "" {
			if _, ok := scope.allNames[name]; ok {
				return true
			}
		}
		if id != "" {
			if _, ok := scope.allIDs[id]; ok {
				return true
			}
		}
	}
	return false
}

//nolint:gocognit // nested index traversal keeps ownership checks explicit
func (p *cleanupPlanner) loadRawRecords() []rawExportRecord {
	managerEntries, err := os.ReadDir(p.paths.indexRoot)
	if err != nil {
		return nil
	}
	var out []rawExportRecord
	for _, managerEntry := range managerEntries {
		if !managerEntry.IsDir() {
			continue
		}
		modeEntries, err := os.ReadDir(filepath.Join(p.paths.indexRoot, managerEntry.Name()))
		if err != nil {
			continue
		}
		for _, modeEntry := range modeEntries {
			if !modeEntry.IsDir() {
				continue
			}
			containerEntries, err := os.ReadDir(filepath.Join(p.paths.indexRoot, managerEntry.Name(), modeEntry.Name()))
			if err != nil {
				continue
			}
			for _, containerEntry := range containerEntries {
				if !containerEntry.IsDir() {
					continue
				}
				dir := filepath.Join(p.paths.indexRoot, managerEntry.Name(), modeEntry.Name(), containerEntry.Name())
				files, err := os.ReadDir(dir)
				if err != nil {
					continue
				}
				for _, file := range files {
					if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
						continue
					}
					recordFile := filepath.Join(dir, file.Name())
					data, err := os.ReadFile(recordFile)
					if err != nil {
						continue
					}
					var record ExportRecord
					if err := json.Unmarshal(data, &record); err != nil {
						continue
					}
					if record.Path == "" || !record.Owner.valid() {
						continue
					}
					if record.Owner.Manager != managerEntry.Name() ||
						record.Owner.Container != containerEntry.Name() ||
						(record.Owner.Rootful && modeEntry.Name() != rootfulValue) ||
						(!record.Owner.Rootful && modeEntry.Name() != rootlessValue) {
						continue
					}
					if p.paths.recordFile(record.Owner, record.Path) != recordFile {
						continue
					}
					out = append(out, rawExportRecord{Record: record, RecordFile: recordFile})
				}
			}
		}
	}
	return out
}

//nolint:gocognit,gocyclo,cyclop,funlen // planning keeps each ownership decision visible instead of hidden in helpers
func (p *cleanupPlanner) planExports() {
	rawRecords := p.loadRawRecords()
	byPath := make(map[string][]rawExportRecord)
	for _, raw := range rawRecords {
		byPath[filepath.Clean(raw.Record.Path)] = append(byPath[filepath.Clean(raw.Record.Path)], raw)
	}

	referenced := make(map[string]struct{})
	for path, records := range byPath {
		if len(records) > 1 {
			owners := make(map[string]struct{})
			for _, raw := range records {
				owner := raw.Record.Owner
				owners[fmt.Sprintf("%s|%s|%t", owner.Manager, owner.Container, owner.Rootful)] = struct{}{}
			}
			if len(owners) > 1 {
				p.manualReview[path] = struct{}{}
				p.fail("export", "host", path, "conflicting export owners; manual review required")
				continue
			}
		}

		metadata, err := readExportMetadata(path)
		for _, raw := range records {
			owner := raw.Record.Owner
			if owner.Rootful && !p.opts.Root {
				p.note("export", owner.label(), path, "rootful record outside scope")
				continue
			}
			if !p.scopeAllows(owner.Manager, owner.Rootful) {
				p.note("export", owner.label(), path, "owner manager outside scope")
				continue
			}
			if !p.hasQueryableScope(owner.Manager, owner.Rootful) {
				p.note("export", owner.label(), path, "owner manager unavailable; deletion vetoed")
				continue
			}
			if err != nil || !metadataMatchesRecord(metadata, raw.Record) {
				p.manualReview[path] = struct{}{}
				if err == nil {
					p.note("export", owner.label(), path, "marker/record mismatch; manual review required")
				}
				if err != nil && !errors.Is(err, fs.ErrNotExist) && !errors.Is(err, errExportNotManaged) {
					p.fail("export", owner.label(), path, err.Error())
				}
				p.queueRemove("record", owner.label(), raw.RecordFile, "stale export record", raw.RecordFile)
				continue
			}
			if !p.paths.underHome(path) {
				p.note("export", owner.label(), path, "export outside target home")
				continue
			}
			referenced[path] = struct{}{}
		}
	}

	roots := append([]string{p.paths.binDir, p.paths.appsDir}, p.opts.ScanPaths...)
	candidates, err := collectExportCandidates(roots)
	if err != nil {
		p.fail("export", "host", "", err.Error())
		return
	}
	for _, candidate := range candidates {
		if _, ok := p.manualReview[filepath.Clean(candidate)]; ok {
			continue
		}
		if !p.paths.underHome(candidate) {
			p.note("export", "host", candidate, "candidate outside target home")
			continue
		}
		if _, ok := referenced[filepath.Clean(candidate)]; ok {
			continue
		}
		metadata, err := readExportMetadata(candidate)
		if err != nil {
			continue
		}
		if metadata.Owner.Rootful && !p.opts.Root {
			p.note("export", metadata.Owner.label(), candidate, "rootful export outside scope")
			continue
		}
		if !p.scopeAllows(metadata.Owner.Manager, metadata.Owner.Rootful) {
			p.note("export", metadata.Owner.label(), candidate, "owner manager outside scope")
			continue
		}
		if !p.hasQueryableScope(metadata.Owner.Manager, metadata.Owner.Rootful) {
			p.note("export", metadata.Owner.label(), candidate, "owner manager unavailable; deletion vetoed")
			continue
		}
		info, err := os.Lstat(candidate)
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		p.queueRemove("export", metadata.Owner.label(), candidate, "orphaned otter export", candidate)
	}

	for _, root := range []string{p.paths.binDir, p.paths.appsDir, p.paths.iconsDir, p.paths.indexRoot} {
		p.queueTemps(root)
	}
}

func (p *cleanupPlanner) queueTemps(root string) {
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
			return nil //nolint:nilerr // best-effort scan: skip unreadable entries
		}
		if !p.opts.Root && strings.HasPrefix(entry.Name(), exportRootfulPrefix) {
			return nil
		}
		if !strings.Contains(entry.Name(), ".otter.tmp.") && !strings.Contains(entry.Name(), ".otter.bak.") {
			return nil
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() {
			return nil //nolint:nilerr // best-effort scan: skip unreadable entries
		}
		p.queueRemove("artifact", "host", path, "leftover otter temp/backup artifact", path)
		return nil
	})
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		p.fail("artifact", "host", root, err.Error())
	}
}

func (p *cleanupPlanner) queueRemove(category, scope, path, reason, target string) {
	info, err := os.Lstat(target)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			p.fail(category, scope, path, err.Error())
		}
		return
	}
	if !info.Mode().IsRegular() {
		p.note(category, scope, path, "skipped non-regular path")
		return
	}
	p.addAction(CleanupItem{
		Category: category,
		Scope:    scope,
		Path:     path,
		Reason:   reason,
		Action:   cleanupActionRemove,
	}, info, nil, "", nil)
}

//nolint:gocognit // launcher checks are deliberately explicit and fail closed
func (p *cleanupPlanner) planPins() {
	env := userenv.LoadTargetUserEnvironment(p.ctx)
	if env == nil || env.Home == "" {
		return
	}
	appsDir := filepath.Join(env.DesktopEntryBaseDir, "applications")
	entries, err := os.ReadDir(appsDir)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			p.fail("pin", "host", appsDir, err.Error())
		}
		return
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".desktop") {
			continue
		}
		path := filepath.Join(appsDir, name)
		values, err := readDesktopEntrySection(path, "Desktop Entry")
		if err != nil {
			continue
		}
		if _, ok := values["X-Otter-Export"]; ok {
			continue
		}
		if !strings.Contains(values["Categories"], "Otter") ||
			values["TryExec"] == "" || !strings.Contains(values["Exec"], " enter ") {
			continue
		}

		manager, container, root, ok := parseCanonicalPinName(name)
		if !ok {
			p.note("pin", "host", path, "legacy unmarked launcher reported only")
			continue
		}
		marker, hasMarker := values["X-Otter-Pin"]
		if !hasMarker {
			p.note("pin", "host", path, "unmarked launcher reported only")
			continue
		}
		payload, err := parsePinMarker(marker)
		if err != nil || payload.Container != container || payload.Manager != manager || payload.Rootful != root {
			p.note("pin", scopeLabel(manager, root), path, "invalid pin marker; manual review required")
			continue
		}
		if !p.scopeAllows(manager, root) {
			p.note("pin", scopeLabel(manager, root), path, "pin owner outside scope")
			continue
		}
		if !p.hasQueryableScope(manager, root) {
			p.note("pin", scopeLabel(manager, root), path, "owner manager unavailable; deletion vetoed")
			continue
		}
		if p.unknown {
			p.note("pin", scopeLabel(manager, root), path, "manager query failed; deletion vetoed")
			continue
		}
		if p.liveIdentity(container, payload.ID) {
			p.fail("pin", scopeLabel(manager, root), path, "identity still exists under another scope")
			continue
		}
		info, err := os.Lstat(path)
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		p.queueRemove("pin", scopeLabel(manager, root), path, "orphaned generated desktop launcher", path)
	}
}

func parseCanonicalPinName(name string) (string, string, bool, bool) {
	matches := cleanupCanonicalPinRe.FindStringSubmatch(name)
	if len(matches) == 0 {
		return "", "", false, false
	}
	manager := matches[2]
	if !slices.Contains(cleanupSupportedManagers, manager) {
		return "", "", false, false
	}
	return manager, matches[3], matches[1] == "rootful", true
}

func readDesktopEntrySection(path, section string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open desktop entry: %w", err)
	}
	defer file.Close()

	values := make(map[string]string)
	current := ""
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			current = strings.TrimSuffix(strings.TrimPrefix(line, "["), "]")
			continue
		}
		if current != section || line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if _, exists := values[key]; !exists {
			values[key] = strings.TrimSpace(value)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan desktop entry: %w", err)
	}
	return values, nil
}

type cleanupPinPayload struct {
	Container string `json:"c"`
	Manager   string `json:"m"`
	Rootful   bool   `json:"r"`
	ID        string `json:"id"`
	Checksum  string `json:"ck"`
}

func pinPayloadChecksum(payload cleanupPinPayload) string {
	sum := sha256.Sum256([]byte(payload.Container + "\x00" + payload.Manager + "\x00" +
		strconv.FormatBool(payload.Rootful) + "\x00" + payload.ID))
	return hex.EncodeToString(sum[:])[:12]
}

func encodePinMarker(payload cleanupPinPayload) string {
	payload.Checksum = pinPayloadChecksum(payload)
	data, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	return cleanupPinMarkerValue + base64.RawURLEncoding.EncodeToString(data)
}

func parsePinMarker(value string) (*cleanupPinPayload, error) {
	if !strings.HasPrefix(value, cleanupPinMarkerValue) {
		return nil, errors.New("unsupported pin marker version")
	}
	data, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(value, cleanupPinMarkerValue))
	if err != nil {
		return nil, fmt.Errorf("decode pin marker: %w", err)
	}
	var payload cleanupPinPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("parse pin marker: %w", err)
	}
	if payload.Checksum != pinPayloadChecksum(payload) {
		return nil, errors.New("pin marker checksum mismatch")
	}
	return &payload, nil
}

//nolint:gocognit // reference gathering stays explicit across all desktop directories
func (p *cleanupPlanner) planIcons() {
	env := userenv.LoadTargetUserEnvironment(p.ctx)
	if env == nil || env.Home == "" {
		return
	}
	referenced := make(map[string]struct{})
	for _, raw := range p.loadRawRecords() {
		if raw.Record.Icon != "" {
			referenced[filepath.Clean(raw.Record.Icon)] = struct{}{}
		}
	}

	iconDirs := []string{
		filepath.Join(env.DesktopEntryBaseDir, "applications"),
		filepath.Join(env.Home, ".local", "share", "applications"),
		"/usr/local/share/applications",
		"/usr/share/applications",
	}
	bareNames := make(map[string]struct{})
	for _, dir := range iconDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".desktop") {
				continue
			}
			values, err := readDesktopEntrySection(filepath.Join(dir, entry.Name()), "Desktop Entry")
			if err != nil || values["Icon"] == "" {
				continue
			}
			icon := values["Icon"]
			referenced[filepath.Clean(icon)] = struct{}{}
			if !strings.ContainsAny(icon, `/\`) {
				bareNames[icon] = struct{}{}
			}
		}
	}

	managedIcons := []string{
		filepath.Join(env.DesktopEntryBaseDir, "otter", "icons", "distros"),
		filepath.Join(env.DesktopEntryBaseDir, "otter", "icons", "apps"),
	}
	for _, dir := range managedIcons {
		err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
			if err != nil || entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
				return nil //nolint:nilerr // best-effort scan: skip unreadable entries
			}
			if !p.opts.Root && strings.HasPrefix(entry.Name(), exportRootfulPrefix) {
				return nil
			}
			if _, ok := referenced[filepath.Clean(path)]; ok {
				return nil
			}
			if _, ok := bareNames[filepath.Base(path)]; ok {
				return nil
			}
			info, err := entry.Info()
			if err != nil || !info.Mode().IsRegular() {
				return nil //nolint:nilerr // best-effort scan: skip unreadable entries
			}
			p.queueRemove("icon", "host", path, "unreferenced managed icon", path)
			return nil
		})
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			p.fail("icon", "host", dir, err.Error())
		}
	}
}

func (p *cleanupPlanner) planImages() {
	if !p.opts.Images || !p.hasScopedImages() {
		return
	}
	props, err := registry.Fetch(p.ctx)
	if err != nil {
		p.fail("image", "registry", "", fmt.Sprintf("failed to fetch catalog: %v", err))
		return
	}

	catalogRefs := cleanupCatalogRefs(props)
	for _, scope := range p.scopes {
		if scope.state != "ok" || scope.cm == nil {
			continue
		}
		protectedIDs := make(map[string]struct{})
		for ref := range catalogRefs {
			if id, ok := scope.cm.ImageID(p.ctx, ref); ok && id != "" {
				protectedIDs[id] = struct{}{}
			}
		}
		for _, image := range scope.images {
			p.planImageCandidate(scope, image, protectedIDs, catalogRefs)
		}
	}
}

// hasScopedImages avoids a network catalog fetch when no queryable engine has
// images to consider. With no candidates there can be no deletion to justify
// a hard catalog failure.
func (p *cleanupPlanner) hasScopedImages() bool {
	for _, scope := range p.scopes {
		if scope.state == "ok" && len(scope.images) > 0 {
			return true
		}
	}
	return false
}

func cleanupCatalogRefs(props *registry.ImagesProperties) map[string]struct{} {
	refs := make(map[string]struct{})
	for _, entry := range props.Images {
		if entry.OfficialImage != "" {
			refs[entry.OfficialImage] = struct{}{}
		} else if entry.Enabled {
			refs[cleanupOfficialRoot+entry.Name+":latest"] = struct{}{}
		}
		if entry.FallbackVendorImage != "" {
			refs[entry.FallbackVendorImage] = struct{}{}
		}
	}
	return refs
}

func (p *cleanupPlanner) planImageCandidate(
	scope *cleanupScope,
	image containermanager.ImageSummary,
	protectedIDs, catalogRefs map[string]struct{},
) {
	if image.ID == "" {
		return
	}
	label := scopeLabel(scope.manager, scope.root)
	if _, ok := protectedIDs[image.ID]; ok {
		return
	}
	if _, ok := scope.usedIDs[image.ID]; ok {
		return
	}
	info, ok := scope.cm.InspectImage(p.ctx, image.ID)
	if !ok || info.ID == "" {
		p.fail("image", label, image.ID, "failed to re-inspect image")
		return
	}
	if _, ok := info.Labels[cleanupImageBuildKey]; !ok {
		return
	}

	catalogTagged, unverifiedTag, multiTag := cleanupTagDisposition(image.RepoTags, catalogRefs)
	// An exact catalog tag protects an image independently of ID resolution.
	if catalogTagged {
		return
	}
	if multiTag {
		p.note("image", label, image.ID, "image has multiple tags; manual review required")
		return
	}
	if unverifiedTag {
		p.note("image", label, image.ID, "otter-derived image has unverified tags")
		return
	}
	p.addAction(CleanupItem{
		Category: "image",
		Scope:    label,
		Path:     image.ID,
		Reason:   "unreferenced otter-derived image",
		Action:   cleanupActionRemove,
	}, nil, scope.cm, image.ID, func() error {
		return p.recheckImageRemoval(scope, catalogRefs, image.ID)
	})
}

func cleanupTagDisposition(tags []string, catalogRefs map[string]struct{}) (bool, bool, bool) {
	catalog := false
	unverified := false
	meaningful := 0
	for _, tag := range tags {
		if tag == "" || strings.HasPrefix(tag, "<none>") {
			continue
		}
		meaningful++
		if _, ok := catalogRefs[tag]; ok {
			catalog = true
		}
		if !strings.HasPrefix(tag, cleanupOfficialRoot) {
			unverified = true
		}
	}
	return catalog, unverified, meaningful > 1
}

// recheckImageRemoval revalidates provenance, live usage and exact catalog
// protection inside the apply lock. Planning can be slow enough for container
// state to change between the plan and the deletion.
func (p *cleanupPlanner) recheckImageRemoval(scope *cleanupScope, catalogRefs map[string]struct{}, imageID string) error {
	if scope.cm == nil {
		return errors.New("image owner manager unavailable")
	}
	info, ok := scope.cm.InspectImage(p.ctx, imageID)
	if !ok || info.ID == "" {
		return errors.New("image vanished before removal")
	}
	if _, ok := info.Labels[cleanupImageBuildKey]; !ok {
		return errors.New("image provenance changed since planning")
	}

	containers, err := scope.cm.ListContainers(p.ctx)
	if err != nil {
		return fmt.Errorf("recheck container usage: %w", err)
	}
	for _, container := range containers {
		id, ok := scope.cm.ContainerImageID(p.ctx, container.Name)
		if !ok {
			return fmt.Errorf("failed to recheck container image for %s", container.Name)
		}
		if id == imageID {
			return errors.New("image became used by a container")
		}
	}

	for ref := range catalogRefs {
		if id, ok := scope.cm.ImageID(p.ctx, ref); ok && id == imageID {
			return errors.New("image became an exact catalog ref")
		}
	}
	images, err := scope.cm.ListImages(p.ctx)
	if err != nil {
		return fmt.Errorf("recheck image list: %w", err)
	}
	for _, image := range images {
		if image.ID != imageID {
			continue
		}
		catalog, _, _ := cleanupTagDisposition(image.RepoTags, catalogRefs)
		if catalog {
			return errors.New("image became tagged as an exact catalog ref")
		}
	}
	return nil
}

// PrintCleanupResult prints the plan or final report as a table or JSON.
//
//nolint:forbidigo // direct terminal output is the command's primary interface
func PrintCleanupResult(result *CleanupResult, asJSON bool) error {
	if asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(result); err != nil {
			return fmt.Errorf("encode cleanup report: %w", err)
		}
		return nil
	}
	fmt.Printf("otter cleanup: %s\n\n", result.Mode)
	if len(result.Scopes) > 0 {
		fmt.Println("Scopes")
		for _, scope := range result.Scopes {
			fmt.Printf("  %-20s %s\n", scopeLabel(scope.Manager, scope.Root), scope.State)
		}
		fmt.Println()
	}
	if len(result.Items) == 0 {
		fmt.Println("No cleanup candidates found.")
	} else {
		fmt.Println("Candidates")
		for _, item := range result.Items {
			fmt.Printf("  [%s] %s: %s — %s\n", item.Category, item.Scope, item.Path, item.Reason)
		}
	}
	if len(result.Problems) > 0 {
		fmt.Println("\nProblems")
		for _, problem := range result.Problems {
			prefix := "NOTE"
			if problem.Hard {
				prefix = "ERROR"
			}
			fmt.Printf("  %s [%s] %s: %s\n", prefix, problem.Category, problem.Path, problem.Message)
		}
	}
	if result.Mode == "apply" {
		fmt.Printf("\nRemoved: %d\n", result.Removed)
	}
	return nil
}
