# Universal MCP Sync

[![Release](https://img.shields.io/github/v/release/nawodyaishan/universal-mcp-sync?display_name=tag)](https://github.com/nawodyaishan/universal-mcp-sync/releases)
[![CI](https://github.com/nawodyaishan/universal-mcp-sync/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/nawodyaishan/universal-mcp-sync/actions/workflows/ci.yml)
[![License](https://img.shields.io/github/license/nawodyaishan/universal-mcp-sync)](./LICENSE)

<p align="center">
  <img src="assets/images/banner.jpeg" width="800" alt="Universal MCP Sync Banner">
</p>

> [!IMPORTANT]
> **Universal MCP Sync** (`usync`) is an Exa-first MCP configuration sync tool for 12+ local AI clients. Previously shipped as `exa-mcp-manager`.


One source of truth for your local AI toolchain. Sync your Exa MCP configuration across Claude Desktop, Cursor, Gemini CLI, Zed, and more with dry-run previews, secret redaction, and atomic rollbacks.

## Demo

<p align="center">
  <img src="assets/gif/demo.gif" width="800" alt="Universal MCP Sync TUI Demo">
</p>

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
- **Provider-neutral TUI foundation**: Complete (Phase 2)
- **Provider-neutral CLI flags**: Planned
- **Windows config paths**: Planned

## Why a dedicated tool?

AI agents *can* edit configs, but they lack the structural guardrails needed for fleet-wide management. `usync` ensures MCP configuration is **repeatable, testable, and safe**:

- **Safety First**: Dry-run previews and automatic secret redaction keep sensitive keys secure.
- **Resilience**: Atomic writes with automatic backups and instant rollback capabilities.
- **Standardization**: High-fidelity QA ensures every config matches official "Golden Path" documentation.

`usync` guarantees the structural integrity of your AI toolchain—not just once, but every time.

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
   brew install usync
   ```

2. **Run a Dry Run** to preview changes:
   ```bash
   usync sync --keys-file ./exa_keys.txt --dry-run
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
   usync sync --keys-file ./exa_keys.txt --apply
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
    "github.com/nawodyaishan/universal-mcp-sync/pkg/app"
    "github.com/nawodyaishan/universal-mcp-sync/pkg/provider"
    "github.com/nawodyaishan/universal-mcp-sync/pkg/config"
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
    fmt.Printf("Successfully updated %d targets\n", len(result.UpdatedTargets))
}
```

## Development

### Workflow
```bash
make tidy    # Clean up dependencies
make lint    # Run golangci-lint
make test    # Run all tests
make build   # Build the binary
```

### Git Hooks
This repo uses [Lefthook](https://github.com/evilmartians/lefthook) to ensure code quality before every commit.

```bash
brew install lefthook
lefthook install
```

Current hooks:
- **`pre-commit`**: Runs `gofmt`, `make vet`, `make lint`, and `make gitignore-check`.
- **`pre-push`**: Runs `make test`, `make build`, and `make gitignore-check`.

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
