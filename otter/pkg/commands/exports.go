package commands

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/ferret-linux/otter/internal/userenv"
	"github.com/ferret-linux/otter/pkg/ui"
)

const (
	exportIndexVersion  = 1
	exportBinaryMarker  = "# otter_binary"
	exportDesktopMarker = "X-Otter-Export=app"

	exportKindBinary = "binary"
	exportKindApp    = "app"

	exportRootfulPrefix = "rootful-"
)

var errExportNotManaged = errors.New("not an otter-managed export")

// ExportOwner identities who owns an export: the container name, the runtime
// that created it, and whether that runtime was running as root. The tuple is
// absolute: a different owner can never overwrite or delete an export.
type ExportOwner struct {
	Container string `json:"container"`
	Manager   string `json:"manager"`
	Rootful   bool   `json:"rootful"`
}

// ExportRecord is one entry in otter's export index. The index is stored next
// to the exports themselves and is authoritative for ownership; the markers
// inside each exported file are the recoverable fallback.
type ExportRecord struct {
	Version      int         `json:"version"`
	Kind         string      `json:"kind"`
	Owner        ExportOwner `json:"owner"`
	Path         string      `json:"path"`
	Icon         string      `json:"icon,omitempty"`
	Source       string      `json:"source,omitempty"`
	Requested    string      `json:"requested,omitempty"`
	OriginalName string      `json:"original_name,omitempty"`
	CreatedAt    string      `json:"created_at,omitempty"`
	UpdatedAt    string      `json:"updated_at,omitempty"`
}

type exportMetadata struct {
	Kind         string
	Owner        ExportOwner
	Icon         string
	Source       string
	Requested    string
	OriginalName string
}

type exportPaths struct {
	home      string
	indexRoot string
	lockPath  string
	appsDir   string
	iconsDir  string
	binDir    string
}

func newExportPaths(ctx context.Context) (*exportPaths, error) {
	env := userenv.LoadTargetUserEnvironment(ctx)
	if env == nil || env.Home == "" {
		return nil, errors.New("cannot determine the target user's home directory")
	}
	dataRoot := filepath.Join(env.Home, ".local", "share")
	otterRoot := filepath.Join(dataRoot, "otter")
	return &exportPaths{
		home:      env.Home,
		indexRoot: filepath.Join(otterRoot, "exports", "index"),
		lockPath:  filepath.Join(otterRoot, "exports", ".lock"),
		appsDir:   filepath.Join(dataRoot, "applications"),
		iconsDir:  filepath.Join(otterRoot, "icons", "apps"),
		binDir:    filepath.Join(env.Home, ".local", "bin"),
	}, nil
}

func (p *exportPaths) ownerDir(owner ExportOwner) string {
	mode := "rootless"
	if owner.Rootful {
		mode = "rootful"
	}
	return filepath.Join(p.indexRoot, owner.Manager, mode, owner.Container)
}

func (p *exportPaths) recordFile(owner ExportOwner, exportPath string) string {
	sum := sha256.Sum256([]byte(exportPath))
	return filepath.Join(p.ownerDir(owner), hex.EncodeToString(sum[:])[:16]+".json")
}

func (o ExportOwner) valid() bool {
	if o.Manager != "podman" && o.Manager != "docker" && o.Manager != "nerdctl" {
		return false
	}
	for _, value := range []string{o.Container, o.Manager} {
		if value == "" || value == "." || value == ".." ||
			strings.ContainsAny(value, "/\\\x00\n\r") {
			return false
		}
	}
	return true
}

func sameExportOwner(a, b ExportOwner) bool {
	return a.Container == b.Container && a.Manager == b.Manager && a.Rootful == b.Rootful
}

func (o ExportOwner) label() string {
	mode := "rootless"
	if o.Rootful {
		mode = "rootful"
	}
	return fmt.Sprintf("%s (%s, %s)", o.Container, o.Manager, mode)
}

func nowExportTimestamp() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// withExportLock serialises every mutating export operation against the same
// host user. A single flock covers the whole operation so a concurrent export
// cannot observe a half-written file or a record that points at one.
func (p *exportPaths) withExportLock(fn func() error) error {
	if err := os.MkdirAll(filepath.Dir(p.lockPath), 0o755); err != nil {
		return fmt.Errorf("failed to create export index directory: %w", err)
	}
	file, err := os.OpenFile(p.lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return fmt.Errorf("failed to open export lock: %w", err)
	}
	defer file.Close()

	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("failed to lock export index: %w", err)
	}
	defer func() {
		_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
	}()

	return fn()
}

func writeFileAtomic(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".otter.tmp.*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpPath, mode); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func (p *exportPaths) writeRecord(record ExportRecord) error {
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return writeFileAtomic(p.recordFile(record.Owner, record.Path), data, 0o644)
}

func (p *exportPaths) removeRecord(owner ExportOwner, exportPath string) error {
	err := os.Remove(p.recordFile(owner, exportPath))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (p *exportPaths) loadRecordsInDir(dir string) []ExportRecord {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var records []ExportRecord
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
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
		records = append(records, record)
	}
	return records
}

func (p *exportPaths) loadOwnerRecords(owner ExportOwner) []ExportRecord {
	var owned []ExportRecord
	for _, record := range p.loadRecordsInDir(p.ownerDir(owner)) {
		if sameExportOwner(record.Owner, owner) {
			owned = append(owned, record)
		}
	}
	return owned
}

func (p *exportPaths) loadAllRecords() []ExportRecord {
	managers, err := os.ReadDir(p.indexRoot)
	if err != nil {
		return nil
	}
	byPath := make(map[string]ExportRecord)
	for _, manager := range managers {
		if !manager.IsDir() {
			continue
		}
		modes, err := os.ReadDir(filepath.Join(p.indexRoot, manager.Name()))
		if err != nil {
			continue
		}
		for _, mode := range modes {
			if !mode.IsDir() {
				continue
			}
			containers, err := os.ReadDir(filepath.Join(p.indexRoot, manager.Name(), mode.Name()))
			if err != nil {
				continue
			}
			for _, container := range containers {
				if !container.IsDir() {
					continue
				}
				dir := filepath.Join(p.indexRoot, manager.Name(), mode.Name(), container.Name())
				for _, record := range p.loadRecordsInDir(dir) {
					// The directory is authoritative for the owner bucket. A
					// record whose JSON identity disagrees with it is corrupt;
					// ignoring it prevents cross-owner deletes.
					if record.Owner.Manager != manager.Name() ||
						record.Owner.Container != container.Name() ||
						(record.Owner.Rootful && mode.Name() != "rootful") ||
						(!record.Owner.Rootful && mode.Name() != "rootless") {
						continue
					}
					existing, ok := byPath[record.Path]
					if !ok || record.UpdatedAt >= existing.UpdatedAt {
						byPath[record.Path] = record
					}
				}
			}
		}
	}

	records := make([]ExportRecord, 0, len(byPath))
	for _, record := range byPath {
		records = append(records, record)
	}
	slices.SortFunc(records, func(a, b ExportRecord) int {
		return strings.Compare(a.Path, b.Path)
	})
	return records
}

func readExportMetadata(path string) (*exportMetadata, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, errExportNotManaged
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	text := string(data)

	if strings.Contains(text, exportBinaryMarker) {
		values := parseMarkerLines(text, map[string]string{
			"container": "# otter-container:",
			"manager":   "# otter-manager:",
			"rootful":   "# otter-rootful:",
			"source":    "# otter-source:",
		})
		return metadataFromValues(values)
	}

	if strings.Contains(text, exportDesktopMarker) {
		values := parseKeyValueLines(text, map[string]string{
			"container":     "X-Otter-Container=",
			"manager":       "X-Otter-Manager=",
			"rootful":       "X-Otter-Rootful=",
			"source":        "X-Otter-Source=",
			"requested":     "X-Otter-Requested=",
			"original_name": "X-Otter-Original-Name=",
			"icon":          "X-Otter-Icon=",
		})
		metadata, err := metadataFromValues(values)
		if err != nil {
			return nil, err
		}
		metadata.Kind = exportKindApp
		return metadata, nil
	}

	return nil, errExportNotManaged
}

func parseMarkerLines(text string, keys map[string]string) map[string]string {
	values := make(map[string]string, len(keys))
	scanner := bufio.NewScanner(strings.NewReader(text))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		for key, prefix := range keys {
			if strings.HasPrefix(line, prefix) {
				values[key] = strings.TrimSpace(strings.TrimPrefix(line, prefix))
			}
		}
	}
	return values
}

func parseKeyValueLines(text string, keys map[string]string) map[string]string {
	values := make(map[string]string, len(keys))
	scanner := bufio.NewScanner(strings.NewReader(text))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		for key, prefix := range keys {
			if strings.HasPrefix(line, prefix) {
				values[key] = strings.TrimSpace(strings.TrimPrefix(line, prefix))
			}
		}
	}
	return values
}

func metadataFromValues(values map[string]string) (*exportMetadata, error) {
	for _, key := range []string{"container", "manager", "rootful", "source"} {
		if values[key] == "" {
			return nil, errExportNotManaged
		}
	}
	rootful := strings.EqualFold(values["rootful"], "true")
	if !rootful && !strings.EqualFold(values["rootful"], "false") {
		return nil, errExportNotManaged
	}
	owner := ExportOwner{
		Container: values["container"],
		Manager:   values["manager"],
		Rootful:   rootful,
	}
	if !owner.valid() {
		return nil, errExportNotManaged
	}
	kind := exportKindBinary
	if values["requested"] != "" || values["icon"] != "" {
		kind = exportKindApp
	}
	return &exportMetadata{
		Kind:         kind,
		Owner:        owner,
		Icon:         values["icon"],
		Source:       values["source"],
		Requested:    values["requested"],
		OriginalName: values["original_name"],
	}, nil
}

func recordFromMetadata(path string, metadata *exportMetadata) ExportRecord {
	timestamp := nowExportTimestamp()
	return ExportRecord{
		Version:      exportIndexVersion,
		Kind:         metadata.Kind,
		Owner:        metadata.Owner,
		Path:         path,
		Icon:         metadata.Icon,
		Source:       metadata.Source,
		Requested:    metadata.Requested,
		OriginalName: metadata.OriginalName,
		CreatedAt:    timestamp,
		UpdatedAt:    timestamp,
	}
}

func metadataMatchesRecord(metadata *exportMetadata, record ExportRecord) bool {
	return metadata != nil &&
		metadata.Kind == record.Kind &&
		sameExportOwner(metadata.Owner, record.Owner) &&
		metadata.Source == record.Source
}

func (p *exportPaths) underHome(path string) bool {
	path = filepath.Clean(path)
	rel, err := filepath.Rel(p.home, path)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) && !filepath.IsAbs(rel)
}

func removeFileIfExists(path string) bool {
	err := os.Remove(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	ui.DefaultLogger.Warn("failed to remove file", "path", path, "err", err)
	return false
}

func (p *exportPaths) deleteRecordArtifacts(record ExportRecord) {
	removeFileIfExists(record.Path)
	if record.Kind == exportKindApp && p.underHome(record.Icon) &&
		strings.HasPrefix(filepath.Clean(record.Icon), filepath.Clean(p.iconsDir)+string(os.PathSeparator)) {
		removeFileIfExists(record.Icon)
	}
	cleanupOtterArtifacts(record.Path)
	if record.Kind == exportKindApp && record.Icon != "" {
		cleanupOtterArtifacts(record.Icon)
	}
}

// cleanupOtterArtifacts removes the temp and backup siblings a staged export
// may have left behind after a crash.
func cleanupOtterArtifacts(path string) int {
	if path == "" {
		return 0
	}
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	removed := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, base+".otter.tmp.") &&
			!strings.HasPrefix(name, base+".otter.bak.") &&
			!strings.HasPrefix(name, ".otter.tmp.") {
			continue
		}
		if removeFileIfExists(filepath.Join(dir, name)) {
			removed++
		}
	}
	return removed
}

func cleanupTempArtifacts(root string, includeRootful bool) int {
	removed := 0
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return nil //nolint:nilerr // best-effort scan: skip unreadable entries
		}
		if entry.IsDir() {
			return nil
		}
		name := entry.Name()
		if !includeRootful && strings.HasPrefix(name, exportRootfulPrefix) {
			return nil
		}
		if strings.Contains(name, ".otter.tmp.") || strings.Contains(name, ".otter.bak.") {
			if removeFileIfExists(path) {
				removed++
			}
		}
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		ui.DefaultLogger.Warn("failed to scan for export artifacts", "path", root, "err", err)
	}
	return removed
}

// ExportsListCommand lists every export recorded in this user's export index,
// across every container, manager, and mode.
type ExportsListCommand struct{}

type ExportsListOptions struct{}

type ExportsListResult struct {
	Exports []ExportRecord
}

func NewExportsListCommand() *ExportsListCommand {
	return &ExportsListCommand{}
}

func (c *ExportsListCommand) Execute(ctx context.Context, _ ExportsListOptions) (*ExportsListResult, error) {
	paths, err := newExportPaths(ctx)
	if err != nil {
		return nil, err
	}
	records := paths.loadAllRecords()
	if len(records) == 0 {
		ui.DefaultLogger.Info("no exports found")
		return &ExportsListResult{}, nil
	}

	table := ui.NewTable(os.Stdout, "NAME", "TYPE", "CONTAINER", "MANAGER", "MODE", "PATH")
	for _, record := range records {
		name := record.OriginalName
		if name == "" {
			name = record.Requested
		}
		if name == "" {
			name = strings.TrimSuffix(filepath.Base(record.Path), filepath.Ext(record.Path))
		}
		mode := "rootless"
		if record.Owner.Rootful {
			mode = "rootful"
		}
		table.AddRow(
			[]string{name, record.Kind, record.Owner.Container, record.Owner.Manager, mode, record.Path},
			[]func(string) string{ui.Teal, ui.Dim, nil, nil, ui.Dim, ui.Dim},
		)
	}
	table.Render()
	return &ExportsListResult{Exports: records}, nil
}

// ExportsRebuildCommand re-indexes otter-managed export files found on disk.
// It is additive and metadata-refreshing only: it never deletes an export file
// and it can move a record when the ownership metadata inside a file changed.
type ExportsRebuildCommand struct{}

type ExportsRebuildOptions struct {
	ScanPaths []string
}

type ExportsRebuildResult struct {
	Added   int
	Updated int
	Skipped int
}

func NewExportsRebuildCommand() *ExportsRebuildCommand {
	return &ExportsRebuildCommand{}
}

func (c *ExportsRebuildCommand) Execute(ctx context.Context, opts ExportsRebuildOptions) (*ExportsRebuildResult, error) {
	paths, err := newExportPaths(ctx)
	if err != nil {
		return nil, err
	}

	candidates, err := collectExportCandidates(append([]string{paths.binDir, paths.appsDir}, opts.ScanPaths...))
	if err != nil {
		return nil, err
	}

	result := &ExportsRebuildResult{}
	err = paths.withExportLock(func() error {
		existing := make(map[string]ExportRecord)
		for _, record := range paths.loadAllRecords() {
			existing[record.Path] = record
		}

		for _, candidate := range candidates {
			metadata, err := readExportMetadata(candidate)
			if err != nil {
				result.Skipped++
				continue
			}

			record, found := existing[candidate]
			if found && sameExportOwner(record.Owner, metadata.Owner) {
				record.Kind = metadata.Kind
				record.Source = metadata.Source
				record.Icon = metadata.Icon
				record.Requested = metadata.Requested
				record.OriginalName = metadata.OriginalName
				record.UpdatedAt = nowExportTimestamp()
				record.Version = exportIndexVersion
				if err := paths.writeRecord(record); err != nil {
					return fmt.Errorf("failed to update export record: %w", err)
				}
				result.Updated++
				continue
			}

			if found {
				if err := paths.removeRecord(record.Owner, record.Path); err != nil {
					return fmt.Errorf("failed to move export record: %w", err)
				}
			}

			record = recordFromMetadata(candidate, metadata)
			if err := paths.writeRecord(record); err != nil {
				return fmt.Errorf("failed to write export record: %w", err)
			}
			if found {
				result.Updated++
			} else {
				result.Added++
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	ui.DefaultLogger.Info(
		"export index rebuilt",
		"added", result.Added,
		"updated", result.Updated,
		"skipped", result.Skipped,
	)
	return result, nil
}

func collectExportCandidates(roots []string) ([]string, error) {
	seen := make(map[string]struct{})
	var candidates []string
	for _, root := range roots {
		if root == "" {
			continue
		}
		root = filepath.Clean(root)
		info, err := os.Lstat(root)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("failed to scan export path %s: %w", root, err)
		}
		if !info.IsDir() {
			if info.Mode().IsRegular() {
				if _, ok := seen[root]; !ok {
					seen[root] = struct{}{}
					candidates = append(candidates, root)
				}
			}
			continue
		}
		err = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return nil //nolint:nilerr // best-effort scan: skip unreadable entries
			}
			if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
				return nil
			}
			if !entry.Type().IsRegular() {
				return nil
			}
			if _, ok := seen[path]; !ok {
				seen[path] = struct{}{}
				candidates = append(candidates, path)
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("failed to scan export path %s: %w", root, err)
		}
	}
	slices.Sort(candidates)
	return candidates, nil
}

// ExportsPruneCommand removes stale records, orphaned otter-managed files,
// unreferenced icons, and leftover temp/backup artifacts. It is global and
// destructive, but only within the target user's own export locations.
type ExportsPruneCommand struct{}

type ExportsPruneOptions struct {
	ScanPaths []string
	Root      bool
}

type ExportsPruneResult struct {
	RemovedRecords int
	RemovedFiles   int
	RemovedIcons   int
	RemovedTemps   int
}

func NewExportsPruneCommand() *ExportsPruneCommand {
	return &ExportsPruneCommand{}
}

func (c *ExportsPruneCommand) Execute(ctx context.Context, opts ExportsPruneOptions) (*ExportsPruneResult, error) {
	paths, err := newExportPaths(ctx)
	if err != nil {
		return nil, err
	}

	result := &ExportsPruneResult{}
	err = paths.withExportLock(func() error {
		records := paths.loadAllRecords()
		validRecords := make(map[string]ExportRecord)
		for _, record := range records {
			if record.Owner.Rootful && !opts.Root {
				validRecords[record.Path] = record
				continue
			}
			metadata, err := readExportMetadata(record.Path)
			if err != nil || !metadataMatchesRecord(metadata, record) {
				if err := paths.removeRecord(record.Owner, record.Path); err != nil {
					ui.DefaultLogger.Warn("failed to remove stale export record", "path", record.Path, "err", err)
				} else {
					result.RemovedRecords++
				}
				continue
			}
			validRecords[record.Path] = record
		}

		candidates, err := collectExportCandidates(append([]string{paths.binDir, paths.appsDir}, opts.ScanPaths...))
		if err != nil {
			return err
		}
		for _, candidate := range candidates {
			if _, ok := validRecords[candidate]; ok {
				continue
			}
			metadata, err := readExportMetadata(candidate)
			if err != nil {
				continue
			}
			if metadata.Owner.Rootful && !opts.Root {
				continue
			}
			if removeFileIfExists(candidate) {
				result.RemovedFiles++
			}
			if metadata.Kind == exportKindApp && paths.underHome(metadata.Icon) &&
				strings.HasPrefix(filepath.Clean(metadata.Icon), filepath.Clean(paths.iconsDir)+string(os.PathSeparator)) {
				if removeFileIfExists(metadata.Icon) {
					result.RemovedIcons++
				}
			}
			cleanupOtterArtifacts(candidate)
			if metadata.Icon != "" {
				cleanupOtterArtifacts(metadata.Icon)
			}
		}

		referencedIcons := make(map[string]struct{})
		for _, record := range paths.loadAllRecords() {
			if record.Icon != "" {
				referencedIcons[filepath.Clean(record.Icon)] = struct{}{}
			}
		}
		result.RemovedIcons += removeOrphanIcons(paths.iconsDir, referencedIcons, opts.Root)

		for _, root := range []string{paths.binDir, paths.appsDir, paths.iconsDir, paths.indexRoot} {
			result.RemovedTemps += cleanupTempArtifacts(root, opts.Root)
		}

		removeEmptyDirs(paths.indexRoot)
		return nil
	})
	if err != nil {
		return nil, err
	}

	ui.DefaultLogger.Info(
		"exports pruned",
		"records", result.RemovedRecords,
		"files", result.RemovedFiles,
		"icons", result.RemovedIcons,
		"artifacts", result.RemovedTemps,
	)
	return result, nil
}

func removeOrphanIcons(iconsDir string, referenced map[string]struct{}, includeRootful bool) int {
	removed := 0
	err := filepath.WalkDir(iconsDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return nil //nolint:nilerr // best-effort scan: skip unreadable entries
		}
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
			return nil
		}
		if _, ok := referenced[filepath.Clean(path)]; ok {
			return nil
		}
		if !includeRootful && strings.HasPrefix(entry.Name(), exportRootfulPrefix) {
			return nil
		}
		if strings.Contains(entry.Name(), ".otter.tmp.") || strings.Contains(entry.Name(), ".otter.bak.") {
			return nil
		}
		if removeFileIfExists(path) {
			removed++
		}
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		ui.DefaultLogger.Warn("failed to scan export icons", "path", iconsDir, "err", err)
	}
	return removed
}

func removeEmptyDirs(root string) {
	var dirs []string
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return nil //nolint:nilerr // best-effort cleanup: skip unreadable entries
		}
		if entry.IsDir() && path != root {
			dirs = append(dirs, path)
		}
		return nil
	})
	slices.SortFunc(dirs, func(a, b string) int {
		return len(b) - len(a)
	})
	for _, dir := range dirs {
		_ = os.Remove(dir)
	}
}

// ExportsDeleteCommand deletes exactly one export by its host path. The path
// is validated against the target user's home and the file must carry a valid
// owner marker, so a desktop action cannot delete an arbitrary host file.
type ExportsDeleteCommand struct{}

type ExportsDeleteOptions struct {
	Path string
}

func NewExportsDeleteCommand() *ExportsDeleteCommand {
	return &ExportsDeleteCommand{}
}

func (c *ExportsDeleteCommand) Execute(ctx context.Context, opts ExportsDeleteOptions) error {
	if strings.TrimSpace(opts.Path) == "" {
		return errors.New("export path is required")
	}
	paths, err := newExportPaths(ctx)
	if err != nil {
		return err
	}

	exportPath, err := filepath.Abs(opts.Path)
	if err != nil {
		return fmt.Errorf("failed to resolve export path: %w", err)
	}
	exportPath = filepath.Clean(exportPath)
	if !paths.underHome(exportPath) {
		return fmt.Errorf("refusing to delete %s: it is outside the target user's home", exportPath)
	}

	return paths.withExportLock(func() error {
		var record *ExportRecord
		for _, candidate := range paths.loadAllRecords() {
			if candidate.Path == exportPath {
				copy := candidate
				record = &copy
				break
			}
		}

		info, err := os.Lstat(exportPath)
		if err != nil {
			if os.IsNotExist(err) {
				if record != nil {
					paths.deleteRecordArtifacts(*record)
					return paths.removeRecord(record.Owner, record.Path)
				}
				return nil
			}
			return fmt.Errorf("failed to inspect export: %w", err)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return fmt.Errorf("refusing to delete %s: not a regular file", exportPath)
		}

		metadata, err := readExportMetadata(exportPath)
		if err != nil {
			if errors.Is(err, errExportNotManaged) {
				return fmt.Errorf("refusing to delete %s: not an otter-managed export", exportPath)
			}
			return fmt.Errorf("failed to read export metadata: %w", err)
		}
		if record != nil && !sameExportOwner(record.Owner, metadata.Owner) {
			return fmt.Errorf("refusing to delete %s: ownership metadata does not match the export index", exportPath)
		}

		deleted := *metadata
		recordToDelete := recordFromMetadata(exportPath, &deleted)
		if record != nil {
			recordToDelete = *record
		}
		paths.deleteRecordArtifacts(recordToDelete)
		return paths.removeRecord(metadata.Owner, exportPath)
	})
}

// CleanupContainerExports removes every export belonging to one owner. It is
// used by `otter remove` and by scoped pruning; callers outside the host
// export commands get the same lock and the same absolute ownership rule.
func CleanupContainerExports(ctx context.Context, manager, container string, rootful, prune bool) error {
	owner := ExportOwner{Container: container, Manager: manager, Rootful: rootful}
	if !owner.valid() {
		return fmt.Errorf("cannot clean exports for %s: invalid owner", owner.label())
	}
	paths, err := newExportPaths(ctx)
	if err != nil {
		return err
	}

	return paths.withExportLock(func() error {
		records := paths.loadOwnerRecords(owner)
		known := make(map[string]struct{}, len(records))
		for _, record := range records {
			known[record.Path] = struct{}{}
			paths.deleteRecordArtifacts(record)
			if err := paths.removeRecord(owner, record.Path); err != nil {
				ui.DefaultLogger.Warn("failed to remove export record", "path", record.Path, "err", err)
			}
		}

		// The index is authoritative, but a container may have been removed
		// before its record was written, so sweep the managed directories for
		// files carrying the same owner marker.
		candidates, err := collectExportCandidates([]string{paths.binDir, paths.appsDir})
		if err != nil {
			return err
		}
		for _, candidate := range candidates {
			if _, ok := known[candidate]; ok {
				continue
			}
			metadata, err := readExportMetadata(candidate)
			if err != nil || !sameExportOwner(metadata.Owner, owner) {
				continue
			}
			if removeFileIfExists(candidate) {
				ui.DefaultLogger.Info("removed export", "path", candidate)
			}
			if metadata.Kind == exportKindApp && paths.underHome(metadata.Icon) &&
				strings.HasPrefix(filepath.Clean(metadata.Icon), filepath.Clean(paths.iconsDir)+string(os.PathSeparator)) {
				removeFileIfExists(metadata.Icon)
				cleanupOtterArtifacts(metadata.Icon)
			}
			cleanupOtterArtifacts(candidate)
		}

		if prune {
			_ = os.RemoveAll(paths.ownerDir(owner))
			removeEmptyDirs(paths.indexRoot)
		}
		return nil
	})
}
