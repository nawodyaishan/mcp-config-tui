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
	AppCursor        AppID = "cursor"
	AppVSCode        AppID = "vscode"
	AppWindsurf      AppID = "windsurf"
	AppZed           AppID = "zed"
	AppRooCode       AppID = "roocode"
	AppOpenCode      AppID = "opencode"
	AppKiro          AppID = "kiro"
)

type FileKind string

const (
	FileKindMCPServers     FileKind = "mcpServers"
	FileKindBareMCPServers FileKind = "bareMCPServers"
	FileKindNamedServer    FileKind = "namedServer"
	FileKindCodexTOML      FileKind = "codexTOML"
	FileKindClaudeCodeCLI  FileKind = "claudeCodeCLI"
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
	AppCursor,
	AppVSCode,
	AppWindsurf,
	AppZed,
	AppRooCode,
	AppOpenCode,
	AppKiro,
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
	cursor := filepath.Join(home, ".cursor", "mcp.json")
	vscode := filepath.Join(home, ".vscode", "mcp.json")
	windsurf := filepath.Join(home, ".codeium", "windsurf", "mcp_config.json")
	zed := filepath.Join(home, ".config", "zed", "settings.json")
	roocode := filepath.Join(home, "Library", "Application Support", "Code", "User", "globalStorage", "saoudrizwan.claude-dev", "settings", "mcp_settings.json")
	opencode := filepath.Join(home, ".opencode.json")
	kiro := filepath.Join(home, ".kiro", "settings", "mcp.json")
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
			ID:   AppCursor,
			Name: "Cursor",
			Files: []TargetFile{
				targetFile("Cursor MCP config", cursor, FileKindMCPServers, true),
			},
		},
		{
			ID:   AppVSCode,
			Name: "VS Code",
			Files: []TargetFile{
				targetFile("VS Code MCP config", vscode, FileKindNamedServer, true), // Uses "servers" root
			},
		},
		{
			ID:   AppWindsurf,
			Name: "Windsurf",
			Files: []TargetFile{
				targetFile("Windsurf MCP config", windsurf, FileKindMCPServers, true),
			},
		},
		{
			ID:   AppZed,
			Name: "Zed",
			Files: []TargetFile{
				targetFile("Zed settings", zed, FileKindNamedServer, true), // Uses "context_servers" root
			},
		},
		{
			ID:   AppRooCode,
			Name: "Roo Code",
			Files: []TargetFile{
				targetFile("Roo Code settings", roocode, FileKindMCPServers, true),
			},
		},
		{
			ID:   AppOpenCode,
			Name: "OpenCode",
			Files: []TargetFile{
				targetFile("OpenCode config", opencode, FileKindNamedServer, true), // Uses "mcp" root
			},
		},
		{
			ID:   AppKiro,
			Name: "Kiro",
			Files: []TargetFile{
				targetFile("Kiro settings", kiro, FileKindMCPServers, true),
			},
		},
		{
			ID:   AppGeminiCLI,
			Name: "Gemini CLI",
			Files: []TargetFile{
				targetFile("Gemini settings", geminiSettings, FileKindMCPServers, true),
				targetFile("Gemini MCP config", geminiMCP, FileKindBareMCPServers, true),
			},
		},
		{
			ID:   AppAntigravity,
			Name: "Antigravity",
			Files: []TargetFile{
				targetFile("Antigravity MCP config", antigravity, FileKindMCPServers, true),
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
