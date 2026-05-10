package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectAppConfigs(t *testing.T) {
	home := t.TempDir()
	
	// Test missing home
	if _, err := DetectAppConfigs(""); err == nil {
		t.Error("expected error for empty home")
	}

	// Create some dummy config files
	claudePath := filepath.Join(home, "Library", "Application Support", "Claude")
	_ = os.MkdirAll(claudePath, 0700)
	_ = os.WriteFile(filepath.Join(claudePath, "claude_desktop_config.json"), []byte("{}"), 0600)

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

func TestAppName(t *testing.T) {
	tests := []struct {
		id   AppID
		want string
	}{
		{AppClaudeDesktop, "Claude Desktop"},
		{AppClaudeCode, "Claude Code"},
		{AppGeminiCLI, "Gemini CLI"},
		{"custom", "custom"},
	}
	for _, tc := range tests {
		if got := AppName(tc.id); got != tc.want {
			t.Errorf("AppName(%q) = %q, want %q", tc.id, got, tc.want)
		}
	}
}
