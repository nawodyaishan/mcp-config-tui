package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDetectAppConfigs(t *testing.T) {
	home := t.TempDir()

	// Test missing home
	if _, err := DetectAppConfigs(""); err == nil {
		t.Error("expected error for empty home")
	}

	// Create a dummy config file for the current platform.
	claudeConfig := appPathsForOS(home, runtime.GOOS).claudeDesktop
	_ = os.MkdirAll(filepath.Dir(claudeConfig), 0700)
	_ = os.WriteFile(claudeConfig, []byte("{}"), 0600)

	apps, err := DetectAppConfigs(home)
	if err != nil {
		t.Fatalf("DetectAppConfigs failed: %v", err)
	}

	if len(apps) == 0 {
		t.Fatal("expected apps to be detected")
	}

	foundClaude := false
	for _, app := range apps {
		if app.ID == AppClaudeDesktop {
			foundClaude = true
			if !app.Files[0].Exists {
				t.Error("expected Claude Desktop config to exist")
			}
		}
	}
	if !foundClaude {
		t.Error("Claude Desktop app not found in detected list")
	}
}

func TestDetectAppConfigsForOS_DarwinPreservesCurrentPaths(t *testing.T) {
	home := t.TempDir()
	apps, err := DetectAppConfigsForOS(home, "darwin")
	if err != nil {
		t.Fatalf("DetectAppConfigsForOS returned error: %v", err)
	}

	assertAppPath(t, apps, AppClaudeDesktop, filepath.Join(home, "Library", "Application Support", "Claude", "claude_desktop_config.json"))
	assertAppPath(t, apps, AppVSCode, filepath.Join(home, ".vscode", "mcp.json"))
	assertAppPath(t, apps, AppWindsurf, filepath.Join(home, ".codeium", "windsurf", "mcp_config.json"))
	assertAppPath(t, apps, AppRooCode, filepath.Join(home, "Library", "Application Support", "Code", "User", "globalStorage", "saoudrizwan.claude-dev", "settings", "mcp_settings.json"))
	assertAppPath(t, apps, AppOpenCode, filepath.Join(home, ".opencode.json"))
	if files := findApp(t, apps, AppGeminiCLI).Files; len(files) != 2 {
		t.Fatalf("expected darwin Gemini CLI to keep 2 files, got %d", len(files))
	}
	if files := findApp(t, apps, AppAntigravityCLI).Files; len(files) != 2 {
		t.Fatalf("expected darwin Antigravity CLI to keep 2 files, got %d", len(files))
	}
	assertAppPath(t, apps, AppAntigravityCLI, filepath.Join(home, ".gemini", "antigravity-cli", "settings.json"))
}

func TestDetectAppConfigsForOS_LinuxUsesNativePaths(t *testing.T) {
	home := t.TempDir()
	apps, err := DetectAppConfigsForOS(home, "linux")
	if err != nil {
		t.Fatalf("DetectAppConfigsForOS returned error: %v", err)
	}

	assertAppPath(t, apps, AppClaudeDesktop, filepath.Join(home, ".config", "Claude", "claude_desktop_config.json"))
	assertAppPath(t, apps, AppVSCode, filepath.Join(home, ".config", "Code", "User", "mcp.json"))
	assertAppPath(t, apps, AppWindsurf, filepath.Join(home, ".codeium", "mcp_config.json"))
	assertAppPath(t, apps, AppRooCode, filepath.Join(home, ".config", "Code", "User", "globalStorage", "saoudrizwan.claude-dev", "settings", "mcp_settings.json"))
	assertAppPath(t, apps, AppOpenCode, filepath.Join(home, ".config", "opencode", "opencode.json"))
	assertAppPath(t, apps, AppZed, filepath.Join(home, ".config", "zed", "settings.json"))
	assertAppPath(t, apps, AppKiro, filepath.Join(home, ".kiro", "settings", "mcp.json"))
	assertAppPath(t, apps, AppCodexCLI, filepath.Join(home, ".codex", "config.toml"))
	assertAppPath(t, apps, AppCursor, filepath.Join(home, ".cursor", "mcp.json"))
	if files := findApp(t, apps, AppGeminiCLI).Files; len(files) != 1 {
		t.Fatalf("expected linux Gemini CLI to use settings.json only, got %d files", len(files))
	}
	if files := findApp(t, apps, AppAntigravityCLI).Files; len(files) != 1 {
		t.Fatalf("expected linux Antigravity CLI to use settings.json only, got %d files", len(files))
	}
	assertAppPath(t, apps, AppAntigravityCLI, filepath.Join(home, ".gemini", "antigravity-cli", "settings.json"))
}

func TestDetectAppConfigsForOS_LinuxMarksExistingFiles(t *testing.T) {
	home := t.TempDir()
	claudeConfig := filepath.Join(home, ".config", "Claude", "claude_desktop_config.json")
	mustWritePathTestFile(t, claudeConfig)

	apps, err := DetectAppConfigsForOS(home, "linux")
	if err != nil {
		t.Fatalf("DetectAppConfigsForOS returned error: %v", err)
	}

	app := findApp(t, apps, AppClaudeDesktop)
	if !app.Files[0].Exists {
		t.Fatal("expected existing linux Claude Desktop config to be marked exists")
	}
}

func TestDetectAppConfigsForOS_LinuxWindsurfPrefersExistingDefaultPath(t *testing.T) {
	home := t.TempDir()
	defaultPath := filepath.Join(home, ".codeium", "mcp_config.json")
	legacyPath := filepath.Join(home, ".codeium", "windsurf", "mcp_config.json")
	mustWritePathTestFile(t, defaultPath)
	mustWritePathTestFile(t, legacyPath)

	apps, err := DetectAppConfigsForOS(home, "linux")
	if err != nil {
		t.Fatalf("DetectAppConfigsForOS returned error: %v", err)
	}

	assertAppPath(t, apps, AppWindsurf, defaultPath)
}

func TestDetectAppConfigsForOS_LinuxWindsurfPrefersExistingLegacyPath(t *testing.T) {
	home := t.TempDir()
	legacyPath := filepath.Join(home, ".codeium", "windsurf", "mcp_config.json")
	mustWritePathTestFile(t, legacyPath)

	apps, err := DetectAppConfigsForOS(home, "linux")
	if err != nil {
		t.Fatalf("DetectAppConfigsForOS returned error: %v", err)
	}

	assertAppPath(t, apps, AppWindsurf, legacyPath)
}

func TestAppName(t *testing.T) {
	tests := []struct {
		id   AppID
		want string
	}{
		{AppClaudeDesktop, "Claude Desktop"},
		{AppClaudeCode, "Claude Code"},
		{AppGeminiCLI, "Gemini CLI (deprecated)"},
		{AppAntigravity, "Antigravity IDE"},
		{AppAntigravityCLI, "Antigravity CLI"},
		{"custom", "custom"},
	}
	for _, tc := range tests {
		if got := AppName(tc.id); got != tc.want {
			t.Errorf("AppName(%q) = %q, want %q", tc.id, got, tc.want)
		}
	}
}

func findApp(t *testing.T, apps []AppConfig, id AppID) AppConfig {
	t.Helper()
	for _, app := range apps {
		if app.ID == id {
			return app
		}
	}
	t.Fatalf("app %s not found", id)
	return AppConfig{}
}

func assertAppPath(t *testing.T, apps []AppConfig, id AppID, want string) {
	t.Helper()
	app := findApp(t, apps, id)
	if len(app.Files) == 0 {
		t.Fatalf("app %s has no files", id)
	}
	if got := app.Files[0].Path; got != want {
		t.Fatalf("%s path = %q, want %q", id, got, want)
	}
}

func mustWritePathTestFile(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte("{}"), 0600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
