package migrate

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/nawodyaishan/universal-mcp-sync/pkg/audit"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/config"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/redact"
)

const GeminiSunsetDeadline = "2026-06-18"

// GeminiSunsetWarning is shown whenever a Gemini config is involved in a migration.
// Enterprise/Workspace users retain Gemini CLI access; this deadline applies only to
// consumer/free/Pro/Ultra plans.
const GeminiSunsetWarning = "Gemini CLI consumer/free/Pro/Ultra access ends on " +
	GeminiSunsetDeadline + ". Enterprise/Workspace users are not affected."

type TargetID string

const (
	TargetAntigravityCLI TargetID = "antigravity-cli"
	TargetAntigravityIDE TargetID = "antigravity-ide"
)

// TargetPath returns the canonical MCP config path for the given migration target.
func TargetPath(homeDir string, target TargetID) (string, error) {
	switch target {
	case TargetAntigravityCLI:
		return filepath.Join(homeDir, ".gemini", "antigravity-cli", "mcp_config.json"), nil
	case TargetAntigravityIDE:
		return filepath.Join(homeDir, ".gemini", "config", "mcp_config.json"), nil
	default:
		return "", fmt.Errorf("unknown migration target %q; use antigravity-cli or antigravity-ide", target)
	}
}

// ExistingTargets returns the TargetIDs whose config paths already exist on disk.
func ExistingTargets(homeDir string) []TargetID {
	var found []TargetID
	for _, id := range []TargetID{TargetAntigravityCLI, TargetAntigravityIDE} {
		p, err := TargetPath(homeDir, id)
		if err != nil {
			continue
		}
		if _, err := os.Lstat(p); err == nil {
			found = append(found, id)
		}
	}
	return found
}

type CopiedEntry struct {
	ProviderID   string
	URLRewritten bool // true when url was rewritten to serverUrl
}

type ConflictEntry struct {
	ProviderID string
	SourceURL  string // redacted
	TargetURL  string // redacted
}

type Preview struct {
	SourcePath     string
	TargetPath     string
	ResolvedTarget string // resolved symlink path; equals TargetPath if not a symlink
	IsSymlink      bool
	Copied         []CopiedEntry
	Skipped        []string // provider IDs already identical in target
	Conflicts      []ConflictEntry
	Warnings       []string
}

type Result struct {
	Preview
	BackupPath string
	Applied    bool
}

type Options struct {
	HomeDir string
	Target  TargetID
	Now     func() time.Time
}

// Plan computes what a migration would do without writing anything.
func Plan(opts Options) (Preview, error) {
	if opts.HomeDir == "" {
		return Preview{}, fmt.Errorf("missing home directory")
	}
	targetPath, err := TargetPath(opts.HomeDir, opts.Target)
	if err != nil {
		return Preview{}, err
	}

	sourcePath := filepath.Join(opts.HomeDir, ".gemini", "settings.json")

	sourceData, sourceExists, err := config.ReadFileOrEmpty(sourcePath)
	if err != nil {
		return Preview{}, fmt.Errorf("read source %s: %w", sourcePath, err)
	}
	if !sourceExists || len(sourceData) == 0 {
		return Preview{}, fmt.Errorf("source %s does not exist or is empty", sourcePath)
	}

	sourceEntries, err := parseMCPServers(sourceData, "mcpServers")
	if err != nil {
		return Preview{}, fmt.Errorf("parse source %s: %w", sourcePath, err)
	}

	resolvedTarget, isSymlink, err := resolveSymlink(targetPath)
	if err != nil {
		return Preview{}, fmt.Errorf("resolve target %s: %w", targetPath, err)
	}
	if isSymlink && !isUnderHome(resolvedTarget, opts.HomeDir) {
		return Preview{}, fmt.Errorf("symlink target %s is outside home directory %s; refusing to write", resolvedTarget, opts.HomeDir)
	}

	var targetEntries map[string]map[string]any
	targetData, targetExists, err := config.ReadFileOrEmpty(resolvedTarget)
	if err != nil {
		return Preview{}, fmt.Errorf("read target %s: %w", resolvedTarget, err)
	}
	if targetExists && len(targetData) > 0 {
		targetEntries, err = parseMCPServers(targetData, "mcpServers")
		if err != nil {
			return Preview{}, fmt.Errorf("parse target %s: %w", resolvedTarget, err)
		}
	}

	preview := Preview{
		SourcePath:     sourcePath,
		TargetPath:     targetPath,
		ResolvedTarget: resolvedTarget,
		IsSymlink:      isSymlink,
		Warnings:       []string{GeminiSunsetWarning},
	}

	for _, id := range sortedKeys(sourceEntries) {
		srcEntry := sourceEntries[id]
		srcURL := extractURL(srcEntry)

		dstEntry, exists := targetEntries[id]
		if exists {
			dstURL := extractAnyURL(dstEntry)
			if srcURL != "" && srcURL == dstURL {
				preview.Skipped = append(preview.Skipped, id)
			} else {
				preview.Conflicts = append(preview.Conflicts, ConflictEntry{
					ProviderID: id,
					SourceURL:  redact.Text(srcURL),
					TargetURL:  redact.Text(dstURL),
				})
			}
			continue
		}

		preview.Copied = append(preview.Copied, CopiedEntry{
			ProviderID:   id,
			URLRewritten: srcURL != "" && !isStdio(srcEntry),
		})
	}

	return preview, nil
}

// Apply executes the migration, writing to the resolved target path.
// Source is never modified. Conflicts and identical entries are skipped.
func Apply(opts Options, preview Preview) (Result, error) {
	if opts.Now == nil {
		opts.Now = time.Now
	}

	writePath := preview.ResolvedTarget

	sourceData, _, err := config.ReadFileOrEmpty(preview.SourcePath)
	if err != nil {
		return Result{}, fmt.Errorf("read source: %w", err)
	}
	sourceEntries, err := parseMCPServers(sourceData, "mcpServers")
	if err != nil {
		return Result{}, fmt.Errorf("parse source: %w", err)
	}

	existingData, existsTarget, err := config.ReadFileOrEmpty(writePath)
	if err != nil {
		return Result{}, fmt.Errorf("read target: %w", err)
	}
	var existingEntries map[string]map[string]any
	if existsTarget && len(existingData) > 0 {
		existingEntries, err = parseMCPServers(existingData, "mcpServers")
		if err != nil {
			return Result{}, fmt.Errorf("parse target: %w", err)
		}
	}

	merged := make(map[string]any, len(existingEntries)+len(preview.Copied))
	for id, entry := range existingEntries {
		merged[id] = entry
	}

	copiedIDs := make(map[string]bool, len(preview.Copied))
	for _, c := range preview.Copied {
		copiedIDs[c.ProviderID] = true
	}
	for id, srcEntry := range sourceEntries {
		if copiedIDs[id] {
			merged[id] = transformEntry(srcEntry)
		}
	}

	serialized, err := json.MarshalIndent(map[string]any{"mcpServers": merged}, "", "  ")
	if err != nil {
		return Result{}, fmt.Errorf("marshal merged config: %w", err)
	}
	serialized = append(serialized, '\n')

	if err := config.EnsureParentDir(writePath); err != nil {
		return Result{}, err
	}
	outcome, err := config.WriteWithBackup(writePath, serialized, opts.Now())
	if err != nil {
		return Result{}, fmt.Errorf("write target: %w", err)
	}

	// Verify parse health of written file.
	written, _, err := config.ReadFileOrEmpty(writePath)
	if err != nil {
		return Result{}, fmt.Errorf("read written target for health check: %w", err)
	}
	if _, err := parseMCPServers(written, "mcpServers"); err != nil {
		return Result{}, fmt.Errorf("target parse health check failed after write: %w", err)
	}

	auditWriter, auditErr := audit.NewWriter(opts.HomeDir)
	if auditErr == nil {
		targets := make([]string, 0, len(preview.Copied))
		for _, c := range preview.Copied {
			targets = append(targets, c.ProviderID)
		}
		_ = auditWriter.Append(audit.Entry{
			Timestamp:    opts.Now(),
			Command:      "migrate gemini-to-antigravity",
			Targets:      targets,
			FilesTouched: []string{writePath},
			ExitCode:     0,
		})
	}

	return Result{Preview: preview, BackupPath: outcome.BackupPath, Applied: true}, nil
}

// Format renders a human-readable migration preview.
func Format(p Preview) string {
	var sb strings.Builder
	sb.WriteString("Migration preview: Gemini CLI → Antigravity\n")
	sb.WriteString(strings.Repeat("=", 44) + "\n\n")
	sb.WriteString("Source:  " + p.SourcePath + "\n")
	sb.WriteString("Target:  " + p.TargetPath + "\n")
	if p.IsSymlink && p.ResolvedTarget != p.TargetPath {
		sb.WriteString("Resolved: " + p.ResolvedTarget + "\n")
	}
	sb.WriteString("\n")

	if len(p.Copied) > 0 {
		sb.WriteString("To copy:\n")
		for _, c := range p.Copied {
			note := ""
			if c.URLRewritten {
				note = " (url → serverUrl)"
			}
			sb.WriteString("  + " + c.ProviderID + note + "\n")
		}
		sb.WriteString("\n")
	}
	if len(p.Skipped) > 0 {
		sb.WriteString("Already present (identical, skipping):\n")
		for _, id := range p.Skipped {
			sb.WriteString("  = " + id + "\n")
		}
		sb.WriteString("\n")
	}
	if len(p.Conflicts) > 0 {
		sb.WriteString("Conflicts (target differs, skipping):\n")
		for _, c := range p.Conflicts {
			sb.WriteString("  ! " + c.ProviderID + "\n")
			sb.WriteString("    source: " + c.SourceURL + "\n")
			sb.WriteString("    target: " + c.TargetURL + "\n")
		}
		sb.WriteString("\n")
	}
	if len(p.Warnings) > 0 {
		sb.WriteString("Warnings:\n")
		for _, w := range p.Warnings {
			sb.WriteString("  * " + w + "\n")
		}
	}
	return sb.String()
}

// FormatResult renders a human-readable apply result.
func FormatResult(r Result) string {
	var sb strings.Builder
	sb.WriteString(Format(r.Preview))
	if r.Applied {
		sb.WriteString("\nApplied successfully.\n")
		if r.BackupPath != "" {
			sb.WriteString("Backup: " + r.BackupPath + "\n")
		}
	}
	return sb.String()
}

// --- internal helpers ---

func parseMCPServers(data []byte, rootKey string) (map[string]map[string]any, error) {
	if len(data) == 0 {
		return map[string]map[string]any{}, nil
	}
	var root map[string]any
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, err
	}
	if rootKey == "" {
		return extractServerMap(root), nil
	}
	v, ok := root[rootKey]
	if !ok {
		return map[string]map[string]any{}, nil
	}
	servers, ok := v.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("expected object at %q, got %T", rootKey, v)
	}
	return extractServerMap(servers), nil
}

func extractServerMap(m map[string]any) map[string]map[string]any {
	result := make(map[string]map[string]any, len(m))
	for id, v := range m {
		if entry, ok := v.(map[string]any); ok {
			result[id] = entry
		}
	}
	return result
}

// extractURL returns the remote URL from a Gemini source entry (field: "url").
func extractURL(entry map[string]any) string {
	if v, ok := entry["url"]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// extractAnyURL returns the URL from an Antigravity entry regardless of field name.
func extractAnyURL(entry map[string]any) string {
	for _, key := range []string{"serverUrl", "url", "httpUrl"} {
		if v, ok := entry[key]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
	}
	return ""
}

func isStdio(entry map[string]any) bool {
	_, ok := entry["command"]
	return ok
}

// transformEntry converts a Gemini source entry to Antigravity target format,
// rewriting the "url" field to "serverUrl" for remote entries.
func transformEntry(src map[string]any) map[string]any {
	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = v
	}
	if !isStdio(src) {
		if url, ok := dst["url"]; ok {
			delete(dst, "url")
			dst["serverUrl"] = url
		}
	}
	return dst
}

func resolveSymlink(path string) (resolved string, isSymlink bool, err error) {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return path, false, nil
		}
		return "", false, err
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return path, false, nil
	}
	resolved, err = filepath.EvalSymlinks(path)
	if err != nil {
		return "", true, fmt.Errorf("eval symlinks for %s: %w", path, err)
	}
	return resolved, true, nil
}

func isUnderHome(path, homeDir string) bool {
	// Resolve both sides so that macOS /var → /private/var symlinks don't
	// cause false-negative comparisons.
	if resolved, err := filepath.EvalSymlinks(homeDir); err == nil {
		homeDir = resolved
	}
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		path = resolved
	}
	rel, err := filepath.Rel(homeDir, path)
	if err != nil {
		return false
	}
	return !strings.HasPrefix(rel, "..")
}

func sortedKeys(m map[string]map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
