package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

type AppID string

const (
	AppClaudeDesktop  AppID = "claude-desktop"
	AppClaudeCode     AppID = "claude-code"
	AppGeminiCLI      AppID = "gemini-cli"
	AppAntigravity    AppID = "antigravity"
	AppAntigravityCLI AppID = "antigravity-cli"
	AppCodexCLI       AppID = "codex-cli"
	AppCursor         AppID = "cursor"
	AppVSCode         AppID = "vscode"
	AppWindsurf       AppID = "windsurf"
	AppZed            AppID = "zed"
	AppRooCode        AppID = "roocode"
	AppOpenCode       AppID = "opencode"
	AppKiro           AppID = "kiro"
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
	AppAntigravityCLI,
	AppAntigravity,
	AppCodexCLI,
}

func DetectAppConfigs(home string) ([]AppConfig, error) {
	return DetectAppConfigsForOS(home, runtime.GOOS)
}

func DetectAppConfigsForOS(home, goos string) ([]AppConfig, error) {
	if home == "" {
		return nil, fmt.Errorf("missing home directory")
	}

	paths := appPathsForOS(home, goos)
	claudeCode := filepath.Join(home, ".claude.json")
	cursor := filepath.Join(home, ".cursor", "mcp.json")
	zed := filepath.Join(home, ".config", "zed", "settings.json")
	kiro := filepath.Join(home, ".kiro", "settings", "mcp.json")
	geminiSettings := filepath.Join(home, ".gemini", "settings.json")
	antigravity := filepath.Join(home, ".gemini", "antigravity", "mcp_config.json")
	antigravityCLISettings := filepath.Join(home, ".gemini", "antigravity-cli", "settings.json")
	antigravityCLILegacy := filepath.Join(home, ".gemini", "antigravity-cli", "mcp_config.json")
	codex := filepath.Join(home, ".codex", "config.toml")

	apps := []AppConfig{
		{
			ID:   AppClaudeDesktop,
			Name: "Claude Desktop",
			Files: []TargetFile{
				targetFile("Claude Desktop config", paths.claudeDesktop, FileKindMCPServers, true),
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
				targetFile("VS Code MCP config", paths.vscode, FileKindNamedServer, true), // Uses "servers" root
			},
		},
		{
			ID:   AppWindsurf,
			Name: "Windsurf",
			Files: []TargetFile{
				targetFile("Windsurf MCP config", paths.windsurf, FileKindMCPServers, true),
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
				targetFile("Roo Code settings", paths.roocode, FileKindMCPServers, true),
			},
		},
		{
			ID:   AppOpenCode,
			Name: "OpenCode",
			Files: []TargetFile{
				targetFile("OpenCode config", paths.opencode, FileKindNamedServer, true), // Uses "mcp" root
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
			ID:    AppGeminiCLI,
			Name:  "Gemini CLI (deprecated)",
			Files: geminiFilesForOS(goos, geminiSettings, filepath.Join(home, ".gemini", "mcp_config.json")),
		},
		{
			ID:    AppAntigravityCLI,
			Name:  "Antigravity CLI",
			Files: antigravityCLIFilesForOS(goos, antigravityCLISettings, antigravityCLILegacy),
		},
		{
			ID:   AppAntigravity,
			Name: "Antigravity IDE",
			Files: []TargetFile{
				targetFile("Antigravity IDE MCP config", antigravity, FileKindMCPServers, true),
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

type platformAppPaths struct {
	claudeDesktop string
	vscode        string
	windsurf      string
	roocode       string
	opencode      string
}

func appPathsForOS(home, goos string) platformAppPaths {
	if goos == "linux" {
		return platformAppPaths{
			claudeDesktop: filepath.Join(home, ".config", "Claude", "claude_desktop_config.json"),
			vscode:        filepath.Join(home, ".config", "Code", "User", "mcp.json"),
			windsurf: chooseExistingPath([]string{
				filepath.Join(home, ".codeium", "mcp_config.json"),
				filepath.Join(home, ".codeium", "windsurf", "mcp_config.json"),
			}),
			roocode:  filepath.Join(home, ".config", "Code", "User", "globalStorage", "saoudrizwan.claude-dev", "settings", "mcp_settings.json"),
			opencode: filepath.Join(home, ".config", "opencode", "opencode.json"),
		}
	}

	return platformAppPaths{
		claudeDesktop: filepath.Join(home, "Library", "Application Support", "Claude", "claude_desktop_config.json"),
		vscode:        filepath.Join(home, ".vscode", "mcp.json"),
		windsurf:      filepath.Join(home, ".codeium", "windsurf", "mcp_config.json"),
		roocode:       filepath.Join(home, "Library", "Application Support", "Code", "User", "globalStorage", "saoudrizwan.claude-dev", "settings", "mcp_settings.json"),
		opencode:      filepath.Join(home, ".opencode.json"),
	}
}

func chooseExistingPath(candidates []string) string {
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return candidates[0]
}

func geminiFilesForOS(goos, settingsPath, legacyMCPPath string) []TargetFile {
	files := []TargetFile{
		targetFile("Gemini settings", settingsPath, FileKindMCPServers, true),
	}
	if goos != "linux" {
		files = append(files, targetFile("Gemini MCP config", legacyMCPPath, FileKindBareMCPServers, true))
	}
	return files
}

func antigravityCLIFilesForOS(goos, settingsPath, legacyMCPPath string) []TargetFile {
	files := []TargetFile{
		targetFile("Antigravity CLI settings", settingsPath, FileKindMCPServers, true),
	}
	if goos != "linux" {
		files = append(files, targetFile("Antigravity CLI MCP config", legacyMCPPath, FileKindBareMCPServers, true))
	}
	return files
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
		return "Gemini CLI (deprecated)"
	case AppAntigravity:
		return "Antigravity IDE"
	case AppAntigravityCLI:
		return "Antigravity CLI"
	case AppCodexCLI:
		return "Codex CLI"
	default:
		return string(id)
	}
}
