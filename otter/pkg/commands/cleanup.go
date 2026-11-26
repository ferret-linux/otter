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
	// Sentinel shared by apply actions and the counter.
	errCleanupAlreadyAbsent = errors.New("cleanup candidate already absent")
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
	ctx                context.Context
	opts               CleanupOptions
	paths              *exportPaths
	scopes             []*cleanupScope
	primary            string
	actions            map[string]cleanupAction
	problems           []CleanupProblem
	problemSeen        map[string]struct{}
	unknown            bool
	manualReview       map[string]struct{}
	rawRecords         []rawExportRecord
	unverifiableOwners map[string]struct{}
	protectedPaths     map[string]struct{}
	indexUnreadable    bool
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
		ctx:                ctx,
		opts:               opts,
		paths:              paths,
		actions:            make(map[string]cleanupAction),
		problemSeen:        make(map[string]struct{}),
		manualReview:       make(map[string]struct{}),
		unverifiableOwners: make(map[string]struct{}),
		protectedPaths:     make(map[string]struct{}),
	}
	planner.buildScopes()
	planner.rawRecords, _ = planner.loadRawRecords(true)
	planner.protectReferencedIcons()
	planner.planExports()
	planner.planPins()
	planner.planIcons()
	planner.planImages()

	result := &CleanupResult{
		Mode:     "plan",
		Items:    []CleanupItem{},
		Problems: append([]CleanupProblem{}, planner.problems...),
		Scopes:   []CleanupScopeState{},
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
					if errors.Is(err, errCleanupAlreadyAbsent) {
						continue
					}
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
	key := strings.Join([]string{severity, category, scope, path, message, strconv.FormatBool(hard)}, "\x00")
	if _, ok := p.problemSeen[key]; ok {
		return
	}
	p.problemSeen[key] = struct{}{}
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
	if _, exists := p.actions[item.Path]; exists {
		return
	}
	action := cleanupAction{CleanupItem: item, key: item.Path}
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
		action.apply = func() error {
			if guard != nil {
				if err := guard(); err != nil {
					return err
				}
			}
			return safeRemove(item.Path, info)
		}
	}
	p.actions[action.key] = action
}

func safeRemove(path string, expected os.FileInfo) error {
	if expected == nil {
		if err := os.Remove(path); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return errCleanupAlreadyAbsent
			}
			return fmt.Errorf("remove candidate: %w", err)
		}
		return nil
	}
	file, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return errCleanupAlreadyAbsent
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
	if err := os.Remove(path); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return errCleanupAlreadyAbsent
		}
		return fmt.Errorf("remove candidate: %w", err)
	}
	return nil
}

// withCleanupLock serialises the apply phase with other export operations.
// Plan mode intentionally never takes this lock and never creates its parent
// directories. When otter runs through an escalator only directories created
// by this call and the opened lock inode are handed back to the target user.
func (p *exportPaths) withCleanupLock(ctx context.Context, fn func() error) error {
	lockDir := filepath.Dir(p.lockPath)
	if !managedDirInHome(p.home, lockDir) {
		return errors.New("export lock path resolves outside target home")
	}
	missing, err := missingDirsUntil(lockDir, p.home)
	if err != nil {
		return fmt.Errorf("failed to inspect export index directory: %w", err)
	}
	if err := os.MkdirAll(lockDir, 0o750); err != nil {
		return fmt.Errorf("failed to create export index directory: %w", err)
	}

	uid, gid, haveIDs := targetUserIDs(ctx)
	if os.Geteuid() == 0 && haveIDs {
		for i := len(missing) - 1; i >= 0; i-- {
			if err := chownDirNoFollow(missing[i], uid, gid); err != nil {
				return fmt.Errorf("failed to hand export directory to target user: %w", err)
			}
		}
	}

	file, err := os.OpenFile(p.lockPath, os.O_CREATE|os.O_RDWR|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0o600)
	if err != nil {
		return fmt.Errorf("failed to open export lock: %w", err)
	}
	defer file.Close()

	if os.Geteuid() == 0 && haveIDs {
		if err := syscall.Fchown(int(file.Fd()), uid, gid); err != nil {
			return fmt.Errorf("failed to hand export lock to target user: %w", err)
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

func targetUserIDs(ctx context.Context) (int, int, bool) {
	env := userenv.LoadTargetUserEnvironment(ctx)
	if env == nil {
		return 0, 0, false
	}
	uid, uidErr := strconv.Atoi(env.UserID)
	gid, gidErr := strconv.Atoi(env.GroupID)
	return uid, gid, uidErr == nil && gidErr == nil
}

func missingDirsUntil(path, home string) ([]string, error) {
	stop := filepath.Clean(home)
	if _, err := os.Lstat(stop); err != nil {
		return nil, fmt.Errorf("inspect home %s: %w", stop, err)
	}
	var missing []string
	for dir := filepath.Clean(path); dir != stop; dir = filepath.Dir(dir) {
		_, err := os.Lstat(dir)
		if err == nil {
			break
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("inspect directory %s: %w", dir, err)
		}
		missing = append(missing, dir)
		if filepath.Dir(dir) == dir {
			return nil, fmt.Errorf("export path %q is outside home %q", path, home)
		}
	}
	return missing, nil
}

func chownDirNoFollow(path string, uid, gid int) error {
	file, err := os.OpenFile(path, os.O_RDONLY|syscall.O_DIRECTORY|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0)
	if err != nil {
		return fmt.Errorf("open directory %s: %w", path, err)
	}
	defer file.Close()
	if err := syscall.Fchown(int(file.Fd()), uid, gid); err != nil {
		return fmt.Errorf("chown directory %s: %w", path, err)
	}
	return nil
}

func pathWithin(base, target string) bool {
	rel, err := filepath.Rel(filepath.Clean(base), filepath.Clean(target))
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) && !filepath.IsAbs(rel)
}

func canonicalImageID(id string) string {
	id = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(id), "sha256:"))
	return strings.ToLower(id)
}

// imageIDMatches accepts both full IDs and engine-abbreviated IDs. Prefixes
// shorter than 12 hex characters are too ambiguous to be used as guards.
func imageIDMatches(a, b string) bool {
	left, right := canonicalImageID(a), canonicalImageID(b)
	if left == "" || right == "" {
		return false
	}
	if left == right {
		return true
	}
	if len(left) < 12 || len(right) < 12 {
		return false
	}
	return strings.HasPrefix(left, right) || strings.HasPrefix(right, left)
}

func imageIDIn(ids map[string]struct{}, id string) bool {
	for known := range ids {
		if imageIDMatches(id, known) {
			return true
		}
	}
	return false
}

// managedDirInHome accepts a directory only when its nearest existing
// ancestor resolves inside the target home and no path component below home
// is a symlink.
func managedDirInHome(home, dir string) bool {
	realHome, err := filepath.EvalSymlinks(home)
	if err != nil {
		return false
	}
	stop := filepath.Clean(home)
	for cur := filepath.Clean(dir); ; cur = filepath.Dir(cur) {
		if cur == stop || filepath.Dir(cur) == cur {
			resolved, err := filepath.EvalSymlinks(cur)
			return err == nil && pathWithin(realHome, resolved)
		}
		info, err := os.Lstat(cur)
		if err == nil {
			if info.Mode()&os.ModeSymlink != 0 {
				return false
			}
			resolved, err := filepath.EvalSymlinks(cur)
			return err == nil && pathWithin(realHome, resolved)
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return false
		}
	}
}

// resolvedPathInHome follows every existing ancestor before comparing with
// the real target home. queueRemove uses it so symlinked app/index/pin
// directories cannot redirect a deletion outside the home.
func resolvedPathInHome(home, path string) bool {
	if !pathWithin(home, path) {
		return false
	}
	realHome, err := filepath.EvalSymlinks(home)
	if err != nil {
		return false
	}
	clean := filepath.Clean(path)
	realParent, err := filepath.EvalSymlinks(filepath.Dir(clean))
	if err != nil {
		return false
	}
	return pathWithin(realHome, filepath.Join(realParent, filepath.Base(clean)))
}

func (p *cleanupPlanner) buildScopes() {
	sudoCommand := p.opts.SudoCommand

	managerType := p.opts.ManagerType
	if managerType != "" && managerType != cleanupAutodetect &&
		!slices.Contains(cleanupSupportedManagers, managerType) {
		p.fail("manager", "host", "", fmt.Sprintf("invalid container manager %q", managerType))
		return
	}

	managerTypes := cleanupSupportedManagers
	if !p.opts.AllManagers {
		if managerType == "" || managerType == cleanupAutodetect {
			managerType = firstInstalledManager()
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
		canonical := canonicalImageID(id)
		scope.usedIDs[canonical] = struct{}{}
		scope.allIDs[canonical] = struct{}{}
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
			for known := range scope.allIDs {
				if imageIDMatches(id, known) {
					return true
				}
			}
		}
	}
	return false
}

//nolint:gocognit,funlen // nested index traversal keeps ownership checks explicit
func (p *cleanupPlanner) loadRawRecords(report bool) ([]rawExportRecord, bool) {
	unreadableNow := false
	unreadable := func(path, reason string) {
		unreadableNow = true
		if !report {
			return
		}
		p.indexUnreadable = true
		p.fail("record", "host", path, reason)
	}
	badRecord := func(dir, path, reason string) {
		// A bad record is never proof of absence, including during apply-time
		// rechecks: guarded removals must fail closed if one appears.
		unreadableNow = true
		if !report {
			return
		}
		p.unverifiableOwners[dir] = struct{}{}
		p.fail("record", "host", path, "unverifiable export record; "+reason)
	}

	managerEntries, err := os.ReadDir(p.paths.indexRoot)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			unreadable(p.paths.indexRoot, err.Error())
		}
		return nil, unreadableNow
	}
	var out []rawExportRecord
	for _, managerEntry := range managerEntries {
		if !managerEntry.IsDir() {
			continue
		}
		managerDir := filepath.Join(p.paths.indexRoot, managerEntry.Name())
		modeEntries, err := os.ReadDir(managerDir)
		if err != nil {
			unreadable(managerDir, err.Error())
			continue
		}
		for _, modeEntry := range modeEntries {
			if !modeEntry.IsDir() {
				continue
			}
			modeDir := filepath.Join(managerDir, modeEntry.Name())
			containerEntries, err := os.ReadDir(modeDir)
			if err != nil {
				unreadable(modeDir, err.Error())
				continue
			}
			for _, containerEntry := range containerEntries {
				if !containerEntry.IsDir() {
					continue
				}
				dir := filepath.Join(modeDir, containerEntry.Name())
				files, err := os.ReadDir(dir)
				if err != nil {
					unreadable(dir, err.Error())
					continue
				}
				for _, file := range files {
					if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
						continue
					}
					recordFile := filepath.Join(dir, file.Name())
					data, err := os.ReadFile(recordFile)
					if err != nil {
						badRecord(dir, recordFile, err.Error())
						continue
					}
					var record ExportRecord
					if err := json.Unmarshal(data, &record); err != nil {
						badRecord(dir, recordFile, err.Error())
						continue
					}
					if record.Path == "" || !record.Owner.valid() {
						badRecord(dir, recordFile, "invalid owner or path")
						continue
					}
					if record.Owner.Manager != managerEntry.Name() ||
						record.Owner.Container != containerEntry.Name() ||
						(record.Owner.Rootful && modeEntry.Name() != rootfulValue) ||
						(!record.Owner.Rootful && modeEntry.Name() != rootlessValue) {
						badRecord(dir, recordFile, "owner does not match index location")
						continue
					}
					if p.paths.recordFile(record.Owner, record.Path) != recordFile {
						badRecord(dir, recordFile, "non-canonical record filename")
						continue
					}
					out = append(out, rawExportRecord{Record: record, RecordFile: recordFile})
				}
			}
		}
	}
	return out, unreadableNow
}

func (p *cleanupPlanner) ownerIndexDir(owner ExportOwner) string {
	mode := rootlessValue
	if owner.Rootful {
		mode = rootfulValue
	}
	return filepath.Join(p.paths.indexRoot, owner.Manager, mode, owner.Container)
}

func (p *cleanupPlanner) ownerVetoed(owner ExportOwner) bool {
	if p.indexUnreadable {
		return true
	}
	_, ok := p.unverifiableOwners[p.ownerIndexDir(owner)]
	return ok
}

func (p *cleanupPlanner) realPathInHome(path string) bool {
	return resolvedPathInHome(p.paths.home, path)
}

func (p *cleanupPlanner) scanRoots() []string {
	roots := make([]string, 0, len(p.opts.ScanPaths))
	for _, root := range p.opts.ScanPaths {
		if abs, err := filepath.Abs(root); err == nil {
			roots = append(roots, abs)
		} else {
			roots = append(roots, root)
		}
	}
	return roots
}

// protectReferencedIcons keeps live icon references out of the artifact sweep
// even when their filenames contain temp/backup substrings.
func (p *cleanupPlanner) protectReferencedIcons() {
	refs, bareNames := p.collectIconReferences(p.rawRecords)
	for path := range refs {
		p.protectedPaths[filepath.Clean(path)] = struct{}{}
	}
	if len(bareNames) == 0 {
		return
	}
	env := userenv.LoadTargetUserEnvironment(p.ctx)
	if env == nil {
		return
	}
	for _, dir := range []string{
		filepath.Join(env.DesktopEntryBaseDir, "otter", "icons", "distros"),
		filepath.Join(env.DesktopEntryBaseDir, "otter", "icons", "apps"),
	} {
		_ = filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
			if err != nil || entry.IsDir() {
				return nil //nolint:nilerr // best-effort protection scan
			}
			if _, ok := bareNames[entry.Name()]; ok {
				p.protectedPaths[filepath.Clean(path)] = struct{}{}
			}
			return nil
		})
	}
}

//nolint:gocognit,gocyclo,cyclop,funlen // planning keeps each ownership decision visible instead of hidden in helpers
func (p *cleanupPlanner) planExports() {
	byPath := make(map[string][]rawExportRecord)
	for _, raw := range p.rawRecords {
		clean := filepath.Clean(raw.Record.Path)
		byPath[clean] = append(byPath[clean], raw)
	}

	referenced := make(map[string]struct{})
	for path, records := range byPath {
		// Any path with an index record must survive the temp/artifact sweep,
		// even if that record is stale, vetoed or under manual review.
		p.protectedPaths[path] = struct{}{}
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

		metadata, metaErr := readExportMetadata(path)
		for _, raw := range records {
			owner := raw.Record.Owner
			if p.ownerVetoed(owner) {
				p.note("export", owner.label(), path, "unverifiable index record; deletion vetoed")
				continue
			}
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
			if metaErr != nil {
				p.manualReview[path] = struct{}{}
				if errors.Is(metaErr, fs.ErrNotExist) || errors.Is(metaErr, errExportNotManaged) {
					p.note("export", owner.label(), path, "marker/record mismatch; manual review required")
					p.queueRemove("record", owner.label(), raw.RecordFile, "stale export record", raw.RecordFile)
				} else {
					p.fail("export", owner.label(), path, metaErr.Error())
				}
				continue
			}
			if !metadataMatchesRecord(metadata, raw.Record) {
				p.manualReview[path] = struct{}{}
				p.note("export", owner.label(), path, "marker/record mismatch; manual review required")
				p.queueRemove("record", owner.label(), raw.RecordFile, "stale export record", raw.RecordFile)
				continue
			}
			if !p.paths.underHome(path) || !p.realPathInHome(path) {
				p.note("export", owner.label(), path, "export outside target home")
				continue
			}
			referenced[path] = struct{}{}
			p.protectedPaths[filepath.Clean(path)] = struct{}{}
		}
	}

	roots := append([]string{p.paths.binDir, p.paths.appsDir}, p.scanRoots()...)
	candidates, err := collectExportCandidates(roots)
	if err != nil {
		p.fail("export", "host", "", err.Error())
		return
	}
	for _, candidate := range candidates {
		clean := filepath.Clean(candidate)
		// Protect only report-only dispositions. Unmarked temp/backup files
		// must stay eligible for the artifact sweep below.
		if _, ok := p.manualReview[clean]; ok {
			p.protectedPaths[clean] = struct{}{}
			continue
		}
		if _, ok := referenced[clean]; ok {
			p.protectedPaths[clean] = struct{}{}
			continue
		}
		if !p.paths.underHome(candidate) || !p.realPathInHome(candidate) {
			p.note("export", "host", candidate, "candidate outside target home")
			p.protectedPaths[clean] = struct{}{}
			continue
		}
		metadata, err := readExportMetadata(candidate)
		if err != nil {
			continue
		}
		owner := metadata.Owner
		source := metadata.Source
		if p.ownerVetoed(owner) {
			p.note("export", owner.label(), candidate, "unverifiable index record; deletion vetoed")
			p.protectedPaths[clean] = struct{}{}
			continue
		}
		if owner.Rootful && !p.opts.Root {
			p.note("export", owner.label(), candidate, "rootful export outside scope")
			p.protectedPaths[clean] = struct{}{}
			continue
		}
		if !p.scopeAllows(owner.Manager, owner.Rootful) {
			p.note("export", owner.label(), candidate, "owner manager outside scope")
			p.protectedPaths[clean] = struct{}{}
			continue
		}
		if !p.hasQueryableScope(owner.Manager, owner.Rootful) {
			p.note("export", owner.label(), candidate, "owner manager unavailable; deletion vetoed")
			p.protectedPaths[clean] = struct{}{}
			continue
		}
		info, err := os.Lstat(candidate)
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		p.queueRemove("export", owner.label(), candidate, "orphaned otter export", candidate,
			func() error { return p.recheckExportRemoval(candidate, owner, source) })
	}

	defaultRoots := []string{p.paths.binDir, p.paths.appsDir, p.paths.iconsDir, p.paths.indexRoot}
	for _, root := range defaultRoots {
		p.queueTemps(root, referenced)
	}
	for _, root := range p.scanRoots() {
		p.queueTemps(root, referenced)
	}
}

func (p *cleanupPlanner) recheckExportRemoval(path string, owner ExportOwner, source string) error {
	if !p.realPathInHome(path) {
		return errors.New("export resolves outside target home before removal")
	}
	metadata, err := readExportMetadata(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("export metadata changed before removal: %w", err)
	}
	if !sameExportOwner(metadata.Owner, owner) || metadata.Source != source {
		return errors.New("export ownership changed before removal")
	}
	records, unreadable := p.loadRawRecords(false)
	if unreadable {
		return errors.New("export index became unreadable before removal")
	}
	for _, raw := range records {
		if filepath.Clean(raw.Record.Path) != filepath.Clean(path) {
			continue
		}
		if sameExportOwner(raw.Record.Owner, owner) {
			return errors.New("export became referenced by a valid record")
		}
	}
	return nil
}

func (p *cleanupPlanner) ownedByTarget(info os.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return true
	}
	env := userenv.LoadTargetUserEnvironment(p.ctx)
	if env == nil || env.UserID == "" {
		return int(stat.Uid) == os.Geteuid()
	}
	uid, err := strconv.Atoi(env.UserID)
	if err != nil {
		return int(stat.Uid) == os.Geteuid()
	}
	return int(stat.Uid) == uid
}

//nolint:gocognit,gocyclo,cyclop // ownership and containment checks stay explicit in the walk callback
func (p *cleanupPlanner) queueTemps(root string, referenced map[string]struct{}) {
	if !managedDirInHome(p.paths.home, root) {
		p.note("artifact", "host", root, "artifact root outside target home; report only")
		return
	}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
			return nil
		}
		clean := filepath.Clean(path)
		if _, ok := referenced[clean]; ok {
			return nil
		}
		if _, ok := p.protectedPaths[clean]; ok {
			return nil
		}
		if _, ok := p.manualReview[clean]; ok {
			return nil
		}
		if !p.opts.Root && strings.HasPrefix(entry.Name(), exportRootfulPrefix) {
			return nil
		}
		if !strings.Contains(entry.Name(), ".otter.tmp.") && !strings.Contains(entry.Name(), ".otter.bak.") {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("inspect artifact %s: %w", path, err)
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		if !p.paths.underHome(path) || !p.realPathInHome(path) {
			p.note("artifact", "host", path, "candidate outside target home")
			return nil
		}
		if !p.opts.Root && !p.ownedByTarget(info) {
			p.note("artifact", "host", path, "artifact owned by another user; manual review required")
			return nil
		}
		// Snapshot plan-time markers so an already-marked leftover in an
		// index/icon root stays sweepable, while a file that becomes or
		// changes into an export between plan and apply fails closed.
		var planned *exportMetadata
		if metadata, err := readExportMetadata(path); err == nil && metadata.Kind != "" {
			snapshot := *metadata
			planned = &snapshot
		}
		p.queueRemove("artifact", "host", path, "leftover otter temp/backup artifact", path, func() error {
			current, err := readExportMetadata(path)
			if errors.Is(err, fs.ErrNotExist) || errors.Is(err, errExportNotManaged) {
				return nil
			}
			if err != nil {
				return fmt.Errorf("artifact metadata changed before removal: %w", err)
			}
			if current.Kind == "" {
				return nil
			}
			if planned == nil {
				return errors.New("artifact became an otter export before removal")
			}
			if current.Kind != planned.Kind ||
				!sameExportOwner(current.Owner, planned.Owner) ||
				current.Source != planned.Source ||
				current.Icon != planned.Icon {
				return errors.New("artifact metadata changed before removal")
			}
			return nil
		})
		return nil
	})
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		p.fail("artifact", "host", root, err.Error())
	}
}

func (p *cleanupPlanner) queueRemove(category, scope, path, reason, target string, guards ...func() error) {
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
	if !p.realPathInHome(target) {
		p.note(category, scope, path, "candidate outside target home after symlink resolution")
		return
	}
	var guard func() error
	if len(guards) > 0 {
		guard = guards[0]
	}
	p.addAction(CleanupItem{
		Category: category,
		Scope:    scope,
		Path:     path,
		Reason:   reason,
		Action:   cleanupActionRemove,
	}, info, nil, "", guard)
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
		marker, hasMarker := values[cleanupPinMarkerKey]
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
		expected := *payload
		p.queueRemove("pin", scopeLabel(manager, root), path, "orphaned generated desktop launcher", path,
			func() error { return p.recheckPinRemoval(path, manager, container, root, expected) })
	}
}

func (p *cleanupPlanner) recheckPinRemoval(path, manager, container string, root bool, expected cleanupPinPayload) error {
	if !p.realPathInHome(path) {
		return errors.New("pin resolves outside target home before removal")
	}
	values, err := readDesktopEntrySection(path, "Desktop Entry")
	if err != nil {
		return fmt.Errorf("pin metadata changed before removal: %w", err)
	}
	marker, ok := values[cleanupPinMarkerKey]
	if !ok {
		return errors.New("pin marker vanished before removal")
	}
	current, err := parsePinMarker(marker)
	if err != nil {
		return fmt.Errorf("pin marker changed before removal: %w", err)
	}
	if current.Container != expected.Container || current.Manager != expected.Manager ||
		current.Rootful != expected.Rootful || current.ID != expected.ID {
		return errors.New("pin identity changed before removal")
	}
	if p.liveIdentityNow(manager, root, container, current.ID) {
		return errors.New("pin identity became live before removal")
	}
	return nil
}

func (p *cleanupPlanner) liveIdentityNow(manager string, root bool, name, id string) bool {
	for _, scope := range p.scopes {
		if scope.manager != manager || scope.root != root || scope.state != "ok" || scope.cm == nil {
			continue
		}
		containers, err := scope.cm.ListContainers(p.ctx)
		if err != nil {
			return true
		}
		for _, container := range containers {
			if container.Name == name {
				return true
			}
			currentID, ok := scope.cm.ContainerImageID(p.ctx, container.Name)
			if !ok || imageIDMatches(currentID, id) {
				return true
			}
		}
		return false
	}
	return true
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
	if p.indexUnreadable || len(p.unverifiableOwners) > 0 {
		p.note("icon", "host", "", "unverifiable index records; icon cleanup vetoed")
		return
	}
	referenced, bareNames := p.collectIconReferences(p.rawRecords)

	managedIcons := []string{
		filepath.Join(env.DesktopEntryBaseDir, "otter", "icons", "distros"),
		filepath.Join(env.DesktopEntryBaseDir, "otter", "icons", "apps"),
	}
	for _, dir := range managedIcons {
		if !managedDirInHome(p.paths.home, dir) {
			p.note("icon", "host", dir, "managed icon root resolves outside target home")
			continue
		}
		err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
				return nil
			}
			if !p.opts.Root && strings.HasPrefix(entry.Name(), exportRootfulPrefix) {
				return nil
			}
			if _, ok := referenced[filepath.Clean(path)]; ok {
				return nil
			}
			base := filepath.Base(path)
			if _, ok := bareNames[base]; ok {
				return nil
			}
			info, err := entry.Info()
			if err != nil {
				return fmt.Errorf("inspect icon %s: %w", path, err)
			}
			if !info.Mode().IsRegular() {
				return nil
			}
			if !p.opts.Root && !p.ownedByTarget(info) {
				p.note("icon", "host", path, "icon owned by another user; manual review required")
				return nil
			}
			p.queueRemove("icon", "host", path, "unreferenced managed icon", path, func() error {
				return p.recheckIconRemoval(path, base)
			})
			return nil
		})
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			p.fail("icon", "host", dir, err.Error())
		}
	}
}

//nolint:gocognit // icon reference gathering stays explicit across desktop roots
func (p *cleanupPlanner) collectIconReferences(records []rawExportRecord) (map[string]struct{}, map[string]struct{}) {
	referenced := make(map[string]struct{})
	bareNames := make(map[string]struct{})
	for _, raw := range records {
		if raw.Record.Icon != "" {
			referenced[filepath.Clean(raw.Record.Icon)] = struct{}{}
		}
	}
	roots := append([]string{p.paths.binDir, p.paths.appsDir}, p.scanRoots()...)
	candidates, err := collectExportCandidates(roots)
	if err == nil {
		for _, candidate := range candidates {
			metadata, err := readExportMetadata(candidate)
			if err != nil || metadata.Icon == "" {
				continue
			}
			referenced[filepath.Clean(metadata.Icon)] = struct{}{}
		}
	}

	env := userenv.LoadTargetUserEnvironment(p.ctx)
	if env == nil {
		return referenced, bareNames
	}
	iconDirs := []string{
		filepath.Join(env.DesktopEntryBaseDir, "applications"),
		filepath.Join(env.Home, ".local", "share", "applications"),
		"/usr/local/share/applications",
		"/usr/share/applications",
	}
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
	return referenced, bareNames
}

func (p *cleanupPlanner) recheckIconRemoval(path, base string) error {
	if !p.realPathInHome(path) {
		return errors.New("icon resolves outside target home before removal")
	}
	records, unreadable := p.loadRawRecords(false)
	if unreadable {
		return errors.New("export index became unreadable before icon removal")
	}
	referenced, bareNames := p.collectIconReferences(records)
	if _, ok := referenced[filepath.Clean(path)]; ok {
		return errors.New("icon became referenced before removal")
	}
	if _, ok := bareNames[base]; ok {
		return errors.New("icon became referenced by name before removal")
	}
	return nil
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
	if len(catalogRefs) == 0 {
		p.fail("image", "registry", "", "catalog contains no usable image references")
		return
	}
	for _, scope := range p.scopes {
		if scope.state != "ok" || scope.cm == nil {
			continue
		}
		protectedIDs := make(map[string]struct{})
		for ref := range catalogRefs {
			if id, ok := scope.cm.ImageID(p.ctx, ref); ok && id != "" {
				protectedIDs[canonicalImageID(id)] = struct{}{}
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
	if props == nil {
		return refs
	}
	add := func(ref string) {
		if ref != "" {
			refs[cleanupNormalizeImageRef(ref)] = struct{}{}
		}
	}
	for _, entry := range props.Images {
		add(entry.OfficialImage)
		if entry.OfficialImage == "" && entry.Enabled {
			add(cleanupOfficialRoot + entry.Name + ":latest")
		}
		add(entry.FallbackVendorImage)
	}
	return refs
}

// cleanupNormalizeImageRef makes an untagged catalog ref equivalent to its
// implicit :latest form so exact-tag protection cannot be bypassed.
func cleanupNormalizeImageRef(ref string) string {
	if ref == "" || strings.Contains(ref, "@") || strings.HasPrefix(ref, "<none>") {
		return ref
	}
	name := ref
	if slash := strings.LastIndex(name, "/"); slash >= 0 {
		name = name[slash+1:]
	}
	if !strings.Contains(name, ":") {
		return ref + ":latest"
	}
	return ref
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
	if imageIDIn(protectedIDs, image.ID) {
		return
	}
	if imageIDIn(scope.usedIDs, image.ID) {
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
	if multiTag {
		p.note("image", label, image.ID, "image has multiple tags; manual review required")
	}
	if unverifiedTag {
		p.note("image", label, image.ID, "otter-derived image has unverified tags")
	}
	// An exact catalog tag protects an image independently of ID resolution.
	if catalogTagged || multiTag || unverifiedTag {
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
		normalized := cleanupNormalizeImageRef(tag)
		if _, ok := catalogRefs[normalized]; ok {
			catalog = true
		}
		if !strings.HasPrefix(normalized, cleanupOfficialRoot) {
			unverified = true
		}
	}
	return catalog, unverified, meaningful > 1
}

// recheckImageRemoval revalidates provenance, live usage and exact catalog
// protection inside the apply lock. Planning can be slow enough for container
// state to change between the plan and the deletion.
//
//nolint:gocognit // all apply-time image vetoes remain visible in one guard
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
		if imageIDMatches(id, imageID) {
			return errors.New("image became used by a container")
		}
	}

	for ref := range catalogRefs {
		if id, ok := scope.cm.ImageID(p.ctx, ref); ok && imageIDMatches(id, imageID) {
			return errors.New("image became an exact catalog ref")
		}
	}
	images, err := scope.cm.ListImages(p.ctx)
	if err != nil {
		return fmt.Errorf("recheck image list: %w", err)
	}
	for _, image := range images {
		if !imageIDMatches(image.ID, imageID) {
			continue
		}
		catalog, unverified, multiTag := cleanupTagDisposition(image.RepoTags, catalogRefs)
		if catalog {
			return errors.New("image became tagged as an exact catalog ref")
		}
		if multiTag {
			return errors.New("image gained multiple tags before removal")
		}
		if unverified {
			return errors.New("image gained unverified tags before removal")
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
