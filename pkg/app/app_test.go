package app

import (
	"bytes"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nawodyaishan/mcp-config-tui/pkg/config"
	"github.com/nawodyaishan/mcp-config-tui/pkg/exa"
	"github.com/nawodyaishan/mcp-config-tui/pkg/provider"
	"github.com/nawodyaishan/mcp-config-tui/pkg/verify"
)

type fakeRunner struct {
	available map[string]bool
	outputs   map[string]string
}

func (f fakeRunner) LookPath(name string) (string, error) {
	if f.available[name] {
		return "/usr/bin/" + name, nil
	}
	return "", os.ErrNotExist
}

func (f fakeRunner) Run(name string, args ...string) (string, error) {
	key := name + " " + strings.Join(args, " ")
	if output, ok := f.outputs[key]; ok {
		return output, nil
	}
	if f.available[name] {
		return "ok", nil
	}
	return "", os.ErrNotExist
}

func TestDefaultAssignmentsForTwoKeys(t *testing.T) {
	selected := map[config.AppID]bool{
		config.AppClaudeDesktop: true,
		config.AppClaudeCode:    true,
		config.AppGeminiCLI:     true,
		config.AppAntigravity:   true,
		config.AppCodexCLI:      true,
	}

	assignments := DefaultAssignments(selected, 2)
	if assignments[config.AppClaudeDesktop] != 0 || assignments[config.AppGeminiCLI] != 0 || assignments[config.AppCodexCLI] != 0 {
		t.Fatal("expected first key for Claude Desktop, Gemini CLI, and Codex")
	}
	if assignments[config.AppClaudeCode] != 1 || assignments[config.AppAntigravity] != 1 {
		t.Fatal("expected second key for Claude Code and Antigravity")
	}
}

func TestManagerApplyUsesFixturesAndMarksOptionalCLIsSkipped(t *testing.T) {
	homeDir := t.TempDir()
	writeFixture(t, homeDir, filepath.Join("Library", "Application Support", "Claude", "claude_desktop_config.json"), filepath.Join("..", "config", "testdata", "claude_desktop.json"))
	writeFixture(t, homeDir, filepath.Join(".gemini", "settings.json"), filepath.Join("..", "config", "testdata", "gemini_settings.json"))
	writeFixture(t, homeDir, filepath.Join(".gemini", "antigravity", "mcp_config.json"), filepath.Join("..", "config", "testdata", "antigravity.json"))
	writeFixture(t, homeDir, filepath.Join(".codex", "config.toml"), filepath.Join("..", "config", "testdata", "codex.toml"))

	runner := fakeRunner{
		available: map[string]bool{
			"codex": false,
		},
	}

	now := func() time.Time {
		return time.Date(2026, time.May, 8, 21, 30, 45, 0, time.UTC)
	}

	manager, err := NewManager(homeDir, now, runner)
	if err != nil {
		t.Fatalf("NewManager returned error: %v", err)
	}

	keys := []string{
		"11111111-1111-1111-1111-111111111111",
		"22222222-2222-2222-2222-222222222222",
	}
	selected := map[config.AppID]bool{
		config.AppClaudeDesktop: true,
		config.AppClaudeCode:    true,
		config.AppGeminiCLI:     true,
		config.AppAntigravity:   true,
		config.AppCodexCLI:      true,
	}
	assignments := DefaultAssignments(selected, len(keys))

	plan, err := manager.Prepare(keys, selected, assignments)
	if err != nil {
		t.Fatalf("Prepare returned error: %v", err)
	}
	if strings.Contains(FormatPlan(plan), keys[0]) || strings.Contains(FormatPlan(plan), keys[1]) {
		t.Fatal("plan output should not contain full API keys")
	}

	result, err := manager.Apply(plan)
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	missingGeminiPath := filepath.Join(homeDir, ".gemini", "mcp_config.json")
	if _, err := os.Stat(missingGeminiPath); err != nil {
		t.Fatalf("expected missing Gemini MCP file to be created: %v", err)
	}

	claudeDesktopBackup := filepath.Join(homeDir, "Library", "Application Support", "Claude", "claude_desktop_config.json.bak-exa-20260508-213045")
	if _, err := os.Stat(claudeDesktopBackup); err != nil {
		t.Fatalf("expected backup to be created: %v", err)
	}

	if len(result.Verification) == 0 {
		t.Fatal("expected verification results")
	}

	for _, item := range result.Verification {
		// Optional CLIs might be skipped or warning, but files must be OK
		if strings.HasSuffix(item.Target, ".json") || strings.HasSuffix(item.Target, ".toml") {
			if item.Status != verify.StatusOK {
				t.Fatalf("expected status OK for file %s, got %s: %v", item.Target, item.Status, item.Details)
			}
		}
	}

	foundSkippedCodex := false
	for _, item := range result.Verification {
		if item.Target == "codex mcp get exa" {
			foundSkippedCodex = true
			if item.Status != verify.StatusSkipped {
				t.Fatalf("expected skipped status for optional codex CLI, got %s", item.Status)
			}
		}
	}
	if !foundSkippedCodex {
		t.Fatal("expected codex optional verification result")
	}
}

func TestManagerApplyRollsBackPriorWritesOnLaterFailure(t *testing.T) {
	homeDir := t.TempDir()
	firstPath := filepath.Join(homeDir, ".gemini", "settings.json")
	secondPath := filepath.Join(homeDir, ".gemini", "mcp_config.json")
	firstOriginal := []byte("{\n  \"name\": \"first\"\n}\n")
	secondOriginal := []byte("{\n  \"name\": \"second\"\n}\n")

	mustWriteFile(t, firstPath, firstOriginal)
	mustWriteFile(t, secondPath, secondOriginal)

	manager, err := NewManager(homeDir, fixedNow(), fakeRunner{})
	if err != nil {
		t.Fatalf("NewManager returned error: %v", err)
	}

	callCount := 0
	manager.WriteConfig = func(path string, data []byte, now time.Time) (config.WriteOutcome, error) {
		callCount++
		if callCount == 2 {
			return config.WriteOutcome{}, errors.New("synthetic write failure")
		}
		return config.WriteWithBackup(path, data, now)
	}

	key := "11111111-1111-1111-1111-111111111111"
	urlValue, err := exa.BuildURL(key, exa.DefaultTools)
	if err != nil {
		t.Fatalf("BuildURL returned error: %v", err)
	}
	cfg := provider.MCPConfig{Type: provider.TransportHTTP, URL: urlValue}

	plan := ExecutionPlan{
		Operations: []Operation{
			{AppName: "Gemini CLI", FileLabel: "Gemini settings", Path: firstPath, Kind: config.FileKindMCPServers, CredentialLabel: "1111...1111", ProviderID: "exa", Config: cfg},
			{AppName: "Gemini CLI", FileLabel: "Gemini MCP config", Path: secondPath, Kind: config.FileKindMCPServers, CredentialLabel: "1111...1111", ProviderID: "exa", Config: cfg},
		},
	}

	result, err := manager.Apply(plan)
	if err == nil {
		t.Fatal("expected Apply to fail")
	}

	firstData, readErr := os.ReadFile(firstPath)
	if readErr != nil {
		t.Fatalf("read first file after rollback: %v", readErr)
	}
	if string(firstData) != string(firstOriginal) {
		t.Fatalf("expected first file to be restored, got:\n%s", string(firstData))
	}

	if len(result.RolledBack) == 0 || result.RolledBack[0] != firstPath {
		t.Fatalf("expected rollback to include %s, got %#v", firstPath, result.RolledBack)
	}
}

func TestManagerLoggerRedactsKeysAndURLs(t *testing.T) {
	homeDir := t.TempDir()
	targetPath := filepath.Join(homeDir, ".gemini", "settings.json")
	mustWriteFile(t, targetPath, []byte("{\n  \"name\": \"settings\"\n}\n"))

	manager, err := NewManager(homeDir, fixedNow(), fakeRunner{})
	if err != nil {
		t.Fatalf("NewManager returned error: %v", err)
	}

	var logs bytes.Buffer
	manager.Logger = slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
	manager.WriteConfig = func(path string, data []byte, now time.Time) (config.WriteOutcome, error) {
		return config.WriteOutcome{}, errors.New("failed with key 11111111-1111-1111-1111-111111111111 and url https://mcp.exa.ai/mcp?exaApiKey=11111111-1111-1111-1111-111111111111&tools=web_search_exa")
	}

	key := "11111111-1111-1111-1111-111111111111"
	urlValue, err := exa.BuildURL(key, exa.DefaultTools)
	if err != nil {
		t.Fatalf("BuildURL returned error: %v", err)
	}
	cfg := provider.MCPConfig{Type: provider.TransportHTTP, URL: urlValue}

	_, err = manager.Apply(ExecutionPlan{
		Operations: []Operation{
			{AppName: "Gemini CLI", FileLabel: "Gemini settings", Path: targetPath, Kind: config.FileKindMCPServers, CredentialLabel: "1111...1111", ProviderID: "exa", Config: cfg},
		},
	})
	if err == nil {
		t.Fatal("expected Apply to fail")
	}

	logText := logs.String()
	if strings.Contains(logText, key) {
		t.Fatalf("expected logs to redact key, got:\n%s", logText)
	}
	if strings.Contains(logText, "https://mcp.exa.ai/mcp?exaApiKey=") {
		t.Fatalf("expected logs to redact Exa URL, got:\n%s", logText)
	}
}

func writeFixture(t *testing.T, homeDir, relativePath, fixturePath string) {
	t.Helper()
	data, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read fixture %s: %v", fixturePath, err)
	}

	path := filepath.Join(homeDir, relativePath)
	mustWriteFile(t, path, data)
}

func mustWriteFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write file %s: %v", path, err)
	}
}

func fixedNow() func() time.Time {
	return func() time.Time {
		return time.Date(2026, time.May, 8, 21, 30, 45, 0, time.UTC)
	}
}
