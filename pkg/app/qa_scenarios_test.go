package app

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/nawodyaishan/mcp-config-tui/pkg/config"
)

func TestQAExaReadmeScenarios(t *testing.T) {
	homeDir := t.TempDir()

	// Initialize dummy files
	claudeDesktopPath := filepath.Join(homeDir, "Library", "Application Support", "Claude", "claude_desktop_config.json")
	geminiSettingsPath := filepath.Join(homeDir, ".gemini", "settings.json")
	antigravityPath := filepath.Join(homeDir, ".gemini", "antigravity", "mcp_config.json")
	codexPath := filepath.Join(homeDir, ".codex", "config.toml")
	cursorPath := filepath.Join(homeDir, ".cursor", "mcp.json")
	vscodePath := filepath.Join(homeDir, ".vscode", "mcp.json")
	windsurfPath := filepath.Join(homeDir, ".codeium", "windsurf", "mcp_config.json")
	zedPath := filepath.Join(homeDir, ".config", "zed", "settings.json")
	roocodePath := filepath.Join(homeDir, "Library", "Application Support", "Code", "User", "globalStorage", "saoudrizwan.claude-dev", "settings", "mcp_settings.json")
	opencodePath := filepath.Join(homeDir, ".opencode.json")
	kiroPath := filepath.Join(homeDir, ".kiro", "settings", "mcp.json")

	mustWriteFile(t, claudeDesktopPath, []byte("{}"))
	mustWriteFile(t, geminiSettingsPath, []byte("{}"))
	mustWriteFile(t, antigravityPath, []byte("{}"))
	mustWriteFile(t, codexPath, []byte(""))
	mustWriteFile(t, cursorPath, []byte("{}"))
	mustWriteFile(t, vscodePath, []byte("{}"))
	mustWriteFile(t, windsurfPath, []byte("{}"))
	mustWriteFile(t, zedPath, []byte("{}"))
	mustWriteFile(t, roocodePath, []byte("{}"))
	mustWriteFile(t, opencodePath, []byte("{}"))
	mustWriteFile(t, kiroPath, []byte("{}"))

	manager, err := NewManager(homeDir, fixedNow(), fakeRunner{})
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	key := "11111111-1111-1111-1111-111111111111"
	selected := make(map[config.AppID]bool)
	for _, id := range config.AppOrder {
		selected[id] = true
	}
	assignments := DefaultAssignments(selected, 1)

	plan, err := manager.Prepare([]string{key}, selected, assignments)
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}

	_, err = manager.Apply(plan)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}

	// 1. Verify Claude Desktop (Manual config / Bridge path)
	data, _ := os.ReadFile(claudeDesktopPath)
	if !bytes.Contains(data, []byte(`"command": "npx"`)) || !bytes.Contains(data, []byte(`"mcp-remote"`)) {
		t.Errorf("Claude Desktop mismatch")
	}

	// 2. Verify Gemini CLI (httpUrl)
	data, _ = os.ReadFile(geminiSettingsPath)
	if !bytes.Contains(data, []byte(`"httpUrl":`)) {
		t.Errorf("Gemini CLI mismatch")
	}

	// 3. Verify Windsurf (serverUrl)
	data, _ = os.ReadFile(windsurfPath)
	if !bytes.Contains(data, []byte(`"serverUrl":`)) {
		t.Errorf("Windsurf mismatch")
	}

	// 4. Verify VS Code (servers root + type: http)
	data, _ = os.ReadFile(vscodePath)
	if !bytes.Contains(data, []byte(`"servers":`)) || !bytes.Contains(data, []byte(`"type": "http"`)) {
		t.Errorf("VS Code mismatch:\n%s", string(data))
	}

	// 5. Verify Zed (context_servers root)
	data, _ = os.ReadFile(zedPath)
	if !bytes.Contains(data, []byte(`"context_servers":`)) {
		t.Errorf("Zed mismatch")
	}

	// 6. Verify Roo Code (type: streamable-http)
	data, _ = os.ReadFile(roocodePath)
	if !bytes.Contains(data, []byte(`"type": "streamable-http"`)) {
		t.Errorf("Roo Code mismatch")
	}

	// 7. Verify OpenCode (mcp root + enabled: true)
	data, _ = os.ReadFile(opencodePath)
	if !bytes.Contains(data, []byte(`"mcp":`)) || !bytes.Contains(data, []byte(`"enabled": true`)) {
		t.Errorf("OpenCode mismatch")
	}

	// 8. Verify Cursor (Standard url)
	data, _ = os.ReadFile(cursorPath)
	if !bytes.Contains(data, []byte(`"url":`)) {
		t.Errorf("Cursor mismatch")
	}
}

func TestQAIdempotency(t *testing.T) {
	homeDir := t.TempDir()
	claudeDesktopPath := filepath.Join(homeDir, "Library", "Application Support", "Claude", "claude_desktop_config.json")
	mustWriteFile(t, claudeDesktopPath, []byte("{}"))

	manager, err := NewManager(homeDir, fixedNow(), fakeRunner{})
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	key := "11111111-1111-1111-1111-111111111111"
	selected := map[config.AppID]bool{config.AppClaudeDesktop: true}
	assignments := DefaultAssignments(selected, 1)

	// First run
	plan1, _ := manager.Prepare([]string{key}, selected, assignments)
	_, _ = manager.Apply(plan1)
	data1, _ := os.ReadFile(claudeDesktopPath)

	// Second run with same key
	plan2, _ := manager.Prepare([]string{key}, selected, assignments)
	_, _ = manager.Apply(plan2)
	data2, _ := os.ReadFile(claudeDesktopPath)

	if !bytes.Equal(data1, data2) {
		t.Errorf("Idempotency failure: second run changed file content")
	}
}
