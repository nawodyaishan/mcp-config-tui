package config

import (
	"fmt"
	"os"
	"path/filepath"
)

type AppID string

const (
	AppClaudeDesktop AppID = "claude-desktop"
	AppClaudeCode    AppID = "claude-code"
	AppGeminiCLI     AppID = "gemini-cli"
	AppAntigravity   AppID = "antigravity"
	AppCodexCLI      AppID = "codex-cli"
)

type FileKind string

const (
	FileKindMCPServers    FileKind = "mcpServers"
	FileKindNamedServer   FileKind = "namedServer"
	FileKindCodexTOML     FileKind = "codexTOML"
	FileKindClaudeCodeCLI FileKind = "claudeCodeCLI"
)

type TargetFile struct {
	Label     string
	Path      string
	Kind      FileKind
	Exists    bool
	Creatable bool
}

type AppConfig struct {
	ID    AppID
	Name  string
	Files []TargetFile
}

var AppOrder = []AppID{
	AppClaudeDesktop,
	AppClaudeCode,
	AppGeminiCLI,
	AppAntigravity,
	AppCodexCLI,
}

func DetectAppConfigs(home string) ([]AppConfig, error) {
	if home == "" {
		return nil, fmt.Errorf("missing home directory")
	}

	claudeDesktop := filepath.Join(home, "Library", "Application Support", "Claude", "claude_desktop_config.json")
	claudeCode := filepath.Join(home, ".claude.json")
	geminiSettings := filepath.Join(home, ".gemini", "settings.json")
	geminiMCP := filepath.Join(home, ".gemini", "mcp_config.json")
	antigravity := filepath.Join(home, ".gemini", "antigravity", "mcp_config.json")
	codex := filepath.Join(home, ".codex", "config.toml")

	apps := []AppConfig{
		{
			ID:   AppClaudeDesktop,
			Name: "Claude Desktop",
			Files: []TargetFile{
				targetFile("Claude Desktop config", claudeDesktop, FileKindMCPServers, true),
			},
		},
		{
			ID:   AppClaudeCode,
			Name: "Claude Code",
			Files: []TargetFile{
				targetFile("Claude Code user config", claudeCode, FileKindClaudeCodeCLI, false),
			},
		},
		{
			ID:   AppGeminiCLI,
			Name: "Gemini CLI",
			Files: []TargetFile{
				targetFile("Gemini settings", geminiSettings, FileKindMCPServers, true),
				targetFile("Gemini MCP config", geminiMCP, FileKindMCPServers, true),
			},
		},
		{
			ID:   AppAntigravity,
			Name: "Antigravity",
			Files: []TargetFile{
				targetFile("Antigravity MCP config", antigravity, FileKindNamedServer, true),
			},
		},
		{
			ID:   AppCodexCLI,
			Name: "Codex CLI",
			Files: []TargetFile{
				targetFile("Codex config", codex, FileKindCodexTOML, true),
			},
		},
	}

	return apps, nil
}

func targetFile(label, path string, kind FileKind, creatable bool) TargetFile {
	_, err := os.Stat(path)
	return TargetFile{
		Label:     label,
		Path:      path,
		Kind:      kind,
		Exists:    err == nil,
		Creatable: creatable,
	}
}

func AppName(id AppID) string {
	switch id {
	case AppClaudeDesktop:
		return "Claude Desktop"
	case AppClaudeCode:
		return "Claude Code"
	case AppGeminiCLI:
		return "Gemini CLI"
	case AppAntigravity:
		return "Antigravity"
	case AppCodexCLI:
		return "Codex CLI"
	default:
		return string(id)
	}
}
