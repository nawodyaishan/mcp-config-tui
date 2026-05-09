# Universal MCP Sync (Exa-First)

<p align="center">
  <img src="assets/images/banner.jpeg" width="800" alt="Universal MCP Sync Banner">
</p>

[![Release](https://img.shields.io/github/v/release/nawodyaishan/mcp-config-tui?display_name=tag)](https://github.com/nawodyaishan/mcp-config-tui/releases)
[![CI](https://github.com/nawodyaishan/mcp-config-tui/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/nawodyaishan/mcp-config-tui/actions/workflows/ci.yml)
[![License](https://img.shields.io/github/license/nawodyaishan/mcp-config-tui)](./LICENSE)

**`exa-mcp-manager` is an Exa-first MCP configuration sync tool for 12+ local AI clients. Internally, it is evolving into a provider-based Universal MCP Sync engine.**

One source of truth for your local AI toolchain. Sync your Exa MCP configuration across Claude Desktop, Cursor, Gemini CLI, Zed, and more with dry-run previews, secret redaction, and atomic rollbacks.

## Demo

Coming soon: TUI walkthrough and dry-run preview.

## Who This Is For

Use this if you:
- Use **Exa MCP** across multiple local AI clients (Claude, Cursor, Windsurf, etc.).
- Want **dry-run previews** before any local configuration files are modified.
- Want **backups, automatic rollback**, and credential redaction.
- Are building **platform engineering tools** around MCP and need a Go library.

Not for you yet if:
- You need Windows support (macOS-first today).
- You require many providers beyond Exa (GitHub, Filesystem, etc. are on the roadmap).

## Current Status

- **Exa provider**: Supported (High-fidelity)
- **12 Local AI Clients**: Supported on macOS
- **Provider-neutral TUI**: In progress (Phase 2 complete)
- **Provider-neutral CLI flags**: Planned
- **Windows config paths**: Planned

## What It Does Today

Core value:
- **Fleet Sync**: Distribute one provider (Exa) across **12+ AI clients** with a single command.
- **Client-Specific Logic**: Automatically handles `stdio` bridges for Claude, custom root keys (`servers`, `context_servers`) for editors, and specialized fields (`httpUrl`, `serverUrl`).
- **High-Fidelity QA**: Every configuration is verified against "Golden Path" scenarios from official documentation.
- **Public API**: Core logic is exposed as a Go library under `pkg/`.

Current supported app targets on macOS:
- Claude Desktop & Code, Cursor, VS Code, Windsurf, Zed, Roo Code, OpenCode, Kiro, Codex, Antigravity, Gemini CLI.

## Quick Start

1. **Install** the binary via Homebrew:
   ```bash
   brew tap nawodyaishan/homebrew-tap
   brew install exa-mcp-manager
   ```

2. **Run a Dry Run** to preview changes:
   ```bash
   exa-mcp-manager --keys-file ./exa_keys.txt --dry-run
   ```

3. **Review Output** (Redacted for safety):
   ```text
   Exa MCP update plan
   ===================
   - Claude Desktop: Claude Desktop config
     credential: exa_****abcd
     path: ~/Library/Application Support/Claude/claude_desktop_config.json
     backup: .../claude_desktop_config.json.bak-exa-20260509-084228
     action: update existing file
   ```

4. **Apply** the changes:
   ```bash
   exa-mcp-manager --keys-file ./exa_keys.txt --apply
   ```

## What Files Can It Modify?

The tool only modifies detected MCP config files for selected clients. No files are changed unless `--apply` is used or the TUI wizard is completed.

| Client | Configuration Path |
| :--- | :--- |
| **Claude Desktop** | `~/Library/Application Support/Claude/claude_desktop_config.json` |
| **Cursor** | `~/.cursor/mcp.json` |
| **VS Code** | `~/.vscode/mcp.json` |
| **Windsurf** | `~/.codeium/windsurf/mcp_config.json` |
| **Zed** | `~/.config/zed/settings.json` |
| **Gemini CLI** | `~/.gemini/settings.json` |

*...and 6 others. A dry run always shows the exact path for your machine.*

## Safety & Trust

- **Redaction**: Full API keys and secret URLs never appear in UI, logs, or reports.
- **Atomic Writes**: Uses a write-and-rename pattern; if one file fails in a sequence, the tool attempts to roll back previous changes.
- **Backups**: Every modified file gets a timestamped `.bak-exa-...` copy in the same directory.
- **QA Suite**: Internal tests validate generated JSON/TOML against official "Golden Path" documentation.

## Go Library Usage

The core logic is available under `pkg/`. **Note**: The API is pre-stable; breaking changes may occur before v2.0.

```go
import (
    "fmt"
    "github.com/nawodyaishan/mcp-config-tui/pkg/app"
    "github.com/nawodyaishan/mcp-config-tui/pkg/provider"
    "github.com/nawodyaishan/mcp-config-tui/pkg/config"
)

func main() {
    manager, err := app.NewManager("/custom/home", nil, nil)
    if err != nil {
        panic(err)
    }

    prov := provider.NewExaProvider()
    
    // Define credentials
    profiles := []provider.CredentialProfile{{
        ProviderID: prov.ID(),
        Values:     map[string]string{"EXA_API_KEY": "YOUR_SECRET_KEY"},
        Label:      "personal",
    }}

    // Target specific apps (e.g., Claude Desktop)
    selected := map[config.AppID]bool{config.AppClaudeDesktop: true}
    assignments := map[config.AppID]int{config.AppClaudeDesktop: 0}

    plan, err := manager.PrepareProvider(prov, profiles, selected, assignments)
    if err != nil {
        panic(err)
    }

    result, err := manager.Apply(plan)
    if err != nil {
        fmt.Printf("Apply failed: %v\n", err)
    }
    fmt.Printf("Successfully updated %d targets\n", len(result.UpdatedTarget))
}
```

## Contributing

### Adding a Provider

New providers are added via the `MCPProvider` interface:

```go
type MCPProvider interface {
    ID() string
    Name() string
    Description() string
    RequiredCredentials() []CredentialSpec
    GenerateConfig(credentials map[string]string) (MCPConfig, error)
}
```

Implementation path:
1. Implement the interface in `pkg/provider/<name>.go`.
2. Register it in `DefaultRegistry()` in `pkg/provider/registry.go`.
3. Add client compatibility rules in `pkg/app/app.go` (`configForTarget`).

### Testing

Reliability is our primary feature. When contributing:
- **Provider Tests**: Validate credential parsing and config generation in `pkg/provider/...`.
- **Config Tests**: Validate JSON/TOML mutation logic in `pkg/config/...`.
- **QA Scenarios**: Update `pkg/app/qa_scenarios_test.go` to include "Golden Path" fixtures for new clients or providers.

```bash
make test               # Run all tests
go test ./pkg/provider  # Run provider logic tests
go test ./pkg/app -v    # Run E2E and QA scenario tests
```

---

## Detailed Documentation
- [Main Project Specification](docs/main-spec.md)
- [Universal Architecture Direction](docs/arch/universal-mcp-manager-plan.md)
- [Phase 2 Implementation Details](docs/plans/phase2.md)
- [Scalability & Plugin Research](docs/arch/future-scalability-research.md)
