package migrate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var fixedNow = func() time.Time {
	return time.Date(2026, 5, 23, 12, 0, 0, 0, time.UTC)
}

func writeFile(t *testing.T, path string, content []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func geminiSettings(t *testing.T, homeDir string, content string) {
	t.Helper()
	writeFile(t, filepath.Join(homeDir, ".gemini", "settings.json"), []byte(content))
}

func TestPlan_CopiesHTTPEntry(t *testing.T) {
	home := t.TempDir()
	geminiSettings(t, home, `{"mcpServers":{"exa":{"url":"https://mcp.exa.ai/mcp?key=abc"}}}`)

	preview, err := Plan(Options{HomeDir: home, Target: TargetAntigravityCLI})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	if len(preview.Copied) != 1 || preview.Copied[0].ProviderID != "exa" {
		t.Errorf("expected exa in Copied, got %+v", preview.Copied)
	}
	if !preview.Copied[0].URLRewritten {
		t.Error("expected URLRewritten=true for http entry")
	}
	if len(preview.Conflicts) != 0 {
		t.Errorf("expected no conflicts, got %+v", preview.Conflicts)
	}
}

func TestPlan_CopiesStdioEntryWithoutURLRewrite(t *testing.T) {
	home := t.TempDir()
	geminiSettings(t, home, `{"mcpServers":{"mcp-tool":{"command":"npx","args":["@tool/mcp"]}}}`)

	preview, err := Plan(Options{HomeDir: home, Target: TargetAntigravityCLI})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	if len(preview.Copied) != 1 || preview.Copied[0].ProviderID != "mcp-tool" {
		t.Errorf("expected mcp-tool in Copied, got %+v", preview.Copied)
	}
	if preview.Copied[0].URLRewritten {
		t.Error("expected URLRewritten=false for stdio entry")
	}
}

func TestPlan_DetectsConflict(t *testing.T) {
	home := t.TempDir()
	geminiSettings(t, home, `{"mcpServers":{"exa":{"url":"https://mcp.exa.ai/mcp?key=aaa"}}}`)
	writeFile(t, filepath.Join(home, ".gemini", "antigravity-cli", "mcp_config.json"),
		[]byte(`{"mcpServers":{"exa":{"serverUrl":"https://mcp.exa.ai/mcp?key=bbb"}}}`))

	preview, err := Plan(Options{HomeDir: home, Target: TargetAntigravityCLI})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	if len(preview.Conflicts) != 1 || preview.Conflicts[0].ProviderID != "exa" {
		t.Errorf("expected exa conflict, got %+v", preview.Conflicts)
	}
	if len(preview.Copied) != 0 {
		t.Errorf("expected nothing copied on conflict, got %+v", preview.Copied)
	}
}

func TestPlan_SkipsIdenticalEntry(t *testing.T) {
	home := t.TempDir()
	geminiSettings(t, home, `{"mcpServers":{"exa":{"url":"https://mcp.exa.ai/mcp?key=abc"}}}`)
	writeFile(t, filepath.Join(home, ".gemini", "antigravity-cli", "mcp_config.json"),
		[]byte(`{"mcpServers":{"exa":{"serverUrl":"https://mcp.exa.ai/mcp?key=abc"}}}`))

	preview, err := Plan(Options{HomeDir: home, Target: TargetAntigravityCLI})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	if len(preview.Skipped) != 1 || preview.Skipped[0] != "exa" {
		t.Errorf("expected exa skipped, got %+v", preview.Skipped)
	}
	if len(preview.Copied) != 0 || len(preview.Conflicts) != 0 {
		t.Errorf("expected nothing else, got copied=%v conflicts=%v", preview.Copied, preview.Conflicts)
	}
}

func TestPlan_EmptySourceErrors(t *testing.T) {
	home := t.TempDir()
	_, err := Plan(Options{HomeDir: home, Target: TargetAntigravityCLI})
	if err == nil || !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("expected missing source error, got %v", err)
	}
}

func TestPlan_InvalidTargetErrors(t *testing.T) {
	home := t.TempDir()
	geminiSettings(t, home, `{"mcpServers":{}}`)
	_, err := Plan(Options{HomeDir: home, Target: "unknown-target"})
	if err == nil || !strings.Contains(err.Error(), "unknown migration target") {
		t.Errorf("expected unknown target error, got %v", err)
	}
}

func TestPlan_RefusesSymlinkOutsideHome(t *testing.T) {
	home := t.TempDir()
	outside := t.TempDir()
	geminiSettings(t, home, `{"mcpServers":{"exa":{"url":"https://x.example/mcp"}}}`)

	// Create a real file outside home, symlink inside home pointing to it.
	outsideTarget := filepath.Join(outside, "mcp_config.json")
	writeFile(t, outsideTarget, []byte("{}"))
	linkPath := filepath.Join(home, ".gemini", "antigravity-cli", "mcp_config.json")
	if err := os.MkdirAll(filepath.Dir(linkPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outsideTarget, linkPath); err != nil {
		t.Fatal(err)
	}

	_, err := Plan(Options{HomeDir: home, Target: TargetAntigravityCLI})
	if err == nil || !strings.Contains(err.Error(), "outside home directory") {
		t.Errorf("expected symlink-outside-home error, got %v", err)
	}
}

func TestPlan_ContainsSunsetWarning(t *testing.T) {
	home := t.TempDir()
	geminiSettings(t, home, `{"mcpServers":{"exa":{"url":"https://mcp.exa.ai/mcp"}}}`)

	preview, err := Plan(Options{HomeDir: home, Target: TargetAntigravityCLI})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	found := false
	for _, w := range preview.Warnings {
		if strings.Contains(w, GeminiSunsetDeadline) {
			found = true
		}
	}
	if !found {
		t.Errorf("expected sunset warning in preview, got %v", preview.Warnings)
	}
}

func TestApply_WritesURLAsServerUrl(t *testing.T) {
	home := t.TempDir()
	geminiSettings(t, home, `{"mcpServers":{"exa":{"url":"https://mcp.exa.ai/mcp?key=abc"}}}`)

	opts := Options{HomeDir: home, Target: TargetAntigravityCLI, Now: fixedNow}
	preview, err := Plan(opts)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	result, err := Apply(opts, preview)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if !result.Applied {
		t.Error("expected Applied=true")
	}

	targetPath := filepath.Join(home, ".gemini", "antigravity-cli", "mcp_config.json")
	data, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	if !strings.Contains(string(data), `"serverUrl"`) {
		t.Errorf("expected serverUrl in target, got:\n%s", data)
	}
	if strings.Contains(string(data), `"url"`) && !strings.Contains(string(data), `"serverUrl"`) {
		t.Errorf("expected url field to be replaced by serverUrl in target:\n%s", data)
	}
}

func TestApply_PreservesExistingTargetEntries(t *testing.T) {
	home := t.TempDir()
	geminiSettings(t, home, `{"mcpServers":{"exa":{"url":"https://mcp.exa.ai/mcp"}}}`)
	writeFile(t, filepath.Join(home, ".gemini", "antigravity-cli", "mcp_config.json"),
		[]byte(`{"mcpServers":{"existing-tool":{"serverUrl":"https://other.example/mcp"}}}`))

	opts := Options{HomeDir: home, Target: TargetAntigravityCLI, Now: fixedNow}
	preview, err := Plan(opts)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	result, err := Apply(opts, preview)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}

	data, err := os.ReadFile(result.ResolvedTarget)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	body := string(data)
	if !strings.Contains(body, "exa") {
		t.Error("expected exa in merged output")
	}
	if !strings.Contains(body, "existing-tool") {
		t.Error("expected existing-tool to be preserved in merged output")
	}
}

func TestApply_DoesNotModifySource(t *testing.T) {
	home := t.TempDir()
	srcContent := `{"mcpServers":{"exa":{"url":"https://mcp.exa.ai/mcp?key=abc"}}}`
	geminiSettings(t, home, srcContent)

	opts := Options{HomeDir: home, Target: TargetAntigravityCLI, Now: fixedNow}
	preview, err := Plan(opts)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if _, err := Apply(opts, preview); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(home, ".gemini", "settings.json"))
	if err != nil {
		t.Fatalf("read source: %v", err)
	}
	if string(got) != srcContent {
		t.Errorf("source was modified: want %q got %q", srcContent, string(got))
	}
}

func TestApply_WritesSymlinkTarget(t *testing.T) {
	home := t.TempDir()
	geminiSettings(t, home, `{"mcpServers":{"exa":{"url":"https://mcp.exa.ai/mcp"}}}`)

	// Set up a symlink inside home.
	realTarget := filepath.Join(home, ".gemini", "antigravity-data", "mcp_config.json")
	writeFile(t, realTarget, []byte(`{"mcpServers":{}}`))
	linkPath := filepath.Join(home, ".gemini", "antigravity-cli", "mcp_config.json")
	if err := os.MkdirAll(filepath.Dir(linkPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(realTarget, linkPath); err != nil {
		t.Fatal(err)
	}

	opts := Options{HomeDir: home, Target: TargetAntigravityCLI, Now: fixedNow}
	preview, err := Plan(opts)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if !preview.IsSymlink {
		t.Error("expected IsSymlink=true")
	}
	resolvedRealTarget, _ := filepath.EvalSymlinks(realTarget)
	if preview.ResolvedTarget != resolvedRealTarget {
		t.Errorf("expected ResolvedTarget=%s, got %s", resolvedRealTarget, preview.ResolvedTarget)
	}

	result, err := Apply(opts, preview)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}

	// Symlink itself must still be a symlink.
	info, err := os.Lstat(linkPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("expected symlink to remain a symlink after apply")
	}

	// Real target must have the migrated content.
	data, err := os.ReadFile(result.ResolvedTarget)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "serverUrl") {
		t.Errorf("expected serverUrl in resolved target, got:\n%s", data)
	}
}

func TestApply_CreatesBackup(t *testing.T) {
	home := t.TempDir()
	geminiSettings(t, home, `{"mcpServers":{"exa":{"url":"https://mcp.exa.ai/mcp"}}}`)
	writeFile(t, filepath.Join(home, ".gemini", "antigravity-cli", "mcp_config.json"),
		[]byte(`{"mcpServers":{}}`))

	opts := Options{HomeDir: home, Target: TargetAntigravityCLI, Now: fixedNow}
	preview, err := Plan(opts)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	result, err := Apply(opts, preview)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}

	if result.BackupPath == "" {
		t.Error("expected non-empty BackupPath")
	}
	if _, err := os.Stat(result.BackupPath); err != nil {
		t.Errorf("backup file does not exist: %v", err)
	}
}

func TestApply_IDETarget(t *testing.T) {
	home := t.TempDir()
	geminiSettings(t, home, `{"mcpServers":{"exa":{"url":"https://mcp.exa.ai/mcp"}}}`)

	opts := Options{HomeDir: home, Target: TargetAntigravityIDE, Now: fixedNow}
	preview, err := Plan(opts)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if _, err := Apply(opts, preview); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	targetPath := filepath.Join(home, ".gemini", "config", "mcp_config.json")
	data, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("read IDE target: %v", err)
	}
	if !strings.Contains(string(data), "serverUrl") {
		t.Errorf("expected serverUrl in IDE target, got:\n%s", data)
	}
}

func TestExistingTargets_ReturnsInstalled(t *testing.T) {
	home := t.TempDir()
	writeFile(t, filepath.Join(home, ".gemini", "antigravity-cli", "mcp_config.json"), []byte("{}"))

	targets := ExistingTargets(home)
	if len(targets) != 1 || targets[0] != TargetAntigravityCLI {
		t.Errorf("expected [antigravity-cli], got %v", targets)
	}
}

func TestFormat_ContainsExpectedSections(t *testing.T) {
	preview := Preview{
		SourcePath:     "/home/.gemini/settings.json",
		TargetPath:     "/home/.gemini/antigravity-cli/mcp_config.json",
		ResolvedTarget: "/home/.gemini/antigravity-cli/mcp_config.json",
		Copied:         []CopiedEntry{{ProviderID: "exa", URLRewritten: true}},
		Skipped:        []string{"ctx7"},
		Conflicts:      []ConflictEntry{{ProviderID: "tavily", SourceURL: "https://a.example", TargetURL: "https://b.example"}},
		Warnings:       []string{GeminiSunsetWarning},
	}

	out := Format(preview)
	for _, want := range []string{"exa", "url → serverUrl", "ctx7", "tavily", GeminiSunsetDeadline} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in Format output:\n%s", want, out)
		}
	}
}

func TestRedactionInConflictOutput(t *testing.T) {
	home := t.TempDir()
	geminiSettings(t, home, `{"mcpServers":{"exa":{"url":"https://mcp.exa.ai/mcp?exaApiKey=11111111-1111-1111-1111-111111111111"}}}`)
	writeFile(t, filepath.Join(home, ".gemini", "antigravity-cli", "mcp_config.json"),
		[]byte(`{"mcpServers":{"exa":{"serverUrl":"https://mcp.exa.ai/mcp?exaApiKey=22222222-2222-2222-2222-222222222222"}}}`))

	preview, err := Plan(Options{HomeDir: home, Target: TargetAntigravityCLI})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	out := Format(preview)
	if strings.Contains(out, "11111111-1111-1111-1111-111111111111") || strings.Contains(out, "22222222-2222-2222-2222-222222222222") {
		t.Errorf("expected full UUIDs to be redacted in output:\n%s", out)
	}
}
