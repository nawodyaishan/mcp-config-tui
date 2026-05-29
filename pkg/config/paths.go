package config

import (
	"fmt"
	"os"
	"runtime"

	"github.com/nawodyaishan/universal-mcp-sync/pkg/manifest"
)

type AppID string

const (
	AppClaudeDesktop  AppID = "claude-desktop"
	AppClaudeCode     AppID = "claude-code"
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
	FileKindCodexCLIAdd    FileKind = "codexCLIAdd" // user-scope codex mcp add; per-project is trust-gated
)

type TargetFile struct {
	Label      string
	Path       string
	Kind       FileKind
	Exists     bool
	Creatable  bool
	Scope      string
	GitWarning bool
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
	clients := manifest.ForPlatform(manifest.AllClients(), goos)
	byID := make(map[AppID]manifest.ClientManifest, len(clients))
	for _, client := range clients {
		byID[AppID(client.ID)] = client
	}

	apps := make([]AppConfig, 0, len(AppOrder))
	for _, appID := range AppOrder {
		client, ok := byID[appID]
		if !ok {
			continue
		}

		files, err := targetFilesForLegacyClient(home, client)
		if err != nil {
			return nil, err
		}
		if len(files) == 0 {
			continue
		}

		apps = append(apps, AppConfig{
			ID:    appID,
			Name:  client.Name,
			Files: files,
		})
	}

	return apps, nil
}

func targetFile(label, path string, kind FileKind, creatable bool, scope string, gitWarning bool) TargetFile {
	_, err := os.Stat(path)
	return TargetFile{
		Label:      label,
		Path:       path,
		Kind:       kind,
		Exists:     err == nil,
		Creatable:  creatable,
		Scope:      scope,
		GitWarning: gitWarning,
	}
}

func targetFilesForLegacyClient(home string, client manifest.ClientManifest) ([]TargetFile, error) {
	candidates := make([]manifest.ConfigCandidate, 0, len(client.Candidates))
	for _, candidate := range client.Candidates {
		if candidate.Scope == manifest.ScopeProject || candidate.Scope == manifest.ScopeWorkspace || candidate.Scope == manifest.ScopeManaged {
			continue
		}
		candidates = append(candidates, candidate)
	}

	if len(candidates) == 0 {
		return nil, nil
	}

	if client.ID == manifest.ClientWindsurf {
		candidate, path, err := preferredLegacyCandidate(home, candidates)
		if err != nil {
			return nil, err
		}
		return []TargetFile{targetFile(
			legacyTargetLabel(AppID(client.ID), candidate.Label),
			path,
			fileKindForMutation(candidate.MutationKind),
			candidate.Creatable,
			string(candidate.Scope),
			candidate.GitWarning,
		)}, nil
	}

	files := make([]TargetFile, 0, len(candidates))
	for _, candidate := range candidates {
		path, err := manifest.ExpandPath(candidate.PathTemplate, manifest.PathVars{Home: home})
		if err != nil {
			return nil, fmt.Errorf("%s %s: %w", client.ID, candidate.Label, err)
		}
		files = append(files, targetFile(
			legacyTargetLabel(AppID(client.ID), candidate.Label),
			path,
			fileKindForMutation(candidate.MutationKind),
			candidate.Creatable,
			string(candidate.Scope),
			candidate.GitWarning,
		))
	}
	return files, nil
}

func preferredLegacyCandidate(home string, candidates []manifest.ConfigCandidate) (manifest.ConfigCandidate, string, error) {
	type resolvedCandidate struct {
		candidate manifest.ConfigCandidate
		path      string
		exists    bool
	}

	resolved := make([]resolvedCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		path, err := manifest.ExpandPath(candidate.PathTemplate, manifest.PathVars{Home: home})
		if err != nil {
			return manifest.ConfigCandidate{}, "", err
		}
		_, statErr := os.Stat(path)
		resolved = append(resolved, resolvedCandidate{
			candidate: candidate,
			path:      path,
			exists:    statErr == nil,
		})
	}

	best := resolved[0]
	for _, next := range resolved[1:] {
		switch {
		case next.exists && !best.exists:
			best = next
		case next.exists == best.exists && next.candidate.Precedence < best.candidate.Precedence:
			best = next
		}
	}

	return best.candidate, best.path, nil
}

func fileKindForMutation(kind manifest.MutationKind) FileKind {
	switch kind {
	case manifest.MutationBareMCPServers:
		return FileKindBareMCPServers
	case manifest.MutationNamedServer:
		return FileKindNamedServer
	case manifest.MutationCodexTOML:
		return FileKindCodexTOML
	case manifest.MutationClaudeCodeCLI:
		return FileKindClaudeCodeCLI
	default:
		return FileKindMCPServers
	}
}

func legacyTargetLabel(appID AppID, candidateLabel string) string {
	switch appID {
	case AppClaudeDesktop:
		return "Claude Desktop config"
	case AppClaudeCode:
		return "Claude Code user config"
	case AppCursor:
		return "Cursor MCP config"
	case AppVSCode:
		return "VS Code MCP config"
	case AppWindsurf:
		return "Windsurf MCP config"
	case AppZed:
		return "Zed settings"
	case AppRooCode:
		return "Roo Code settings"
	case AppOpenCode:
		return "OpenCode config"
	case AppKiro:
		return "Kiro settings"
	case AppAntigravityCLI:
		switch candidateLabel {
		case "legacy-gemini-config":
			return "Antigravity CLI legacy config (gemini)"
		case "legacy-pre-io26":
			return "Antigravity CLI legacy config (pre-IO26)"
		default:
			return "Antigravity CLI settings"
		}
	case AppAntigravity:
		return "Antigravity IDE MCP config"
	case AppCodexCLI:
		return "Codex config"
	default:
		return AppName(appID)
	}
}

func AppName(id AppID) string {
	switch id {
	case AppClaudeDesktop:
		return "Claude Desktop"
	case AppClaudeCode:
		return "Claude Code"
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
