# MCP Config

[![Release](https://img.shields.io/github/v/release/nawodyaishan/mcp-config-tui?display_name=tag)](https://github.com/nawodyaishan/mcp-config-tui/releases)
[![CI](https://github.com/nawodyaishan/mcp-config-tui/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/nawodyaishan/mcp-config-tui/actions/workflows/ci.yml)
[![License](https://img.shields.io/github/license/nawodyaishan/mcp-config-tui)](./LICENSE)

> [!NOTE]
> **Project Status:** Universal MCP manager direction, with Exa as the launch provider today.

One source of truth for your local AI toolchain. Sync MCP server configuration across Claude Desktop, Claude Code, Gemini CLI, Antigravity, and Codex with dry-run previews, redaction, backups, and rollback.

`mcp-config-tui` is a macOS-first MCP sync utility for developers who run multiple local AI tools on the same machine. The current shipped binary is `exa-mcp-manager`, which provides first-class Exa support on top of a provider-based architecture that is being generalized into a broader MCP manager.

## Who This Is For

Primary audience:

- developers using Claude Desktop, Claude Code, Gemini CLI, Antigravity, or Codex on the same machine
- developers who want to stop hand-editing several MCP config files every time a server URL, tool list, or credential changes
- developers who care about dry-run previews, rollback, redaction, and verification before touching local config

Secondary audience:

- contributors extending the tool from Exa-only rollout into a broader provider-driven MCP manager
- internal tooling engineers experimenting with a local MCP sync workflow before generalizing it

This is not yet a generic MCP installer for every server or client. It is a focused tool with a clear next step toward that architecture.

## What It Does Today

Core value:

- unified, safe MCP configuration for a multi-client local AI setup

Current supported app targets on macOS:

- Claude Desktop
- Claude Code
- Gemini CLI
- Antigravity
- Codex CLI

Current provider support:

- Exa

Client-specific notes:

- Claude Desktop file-based setup uses a `stdio` bridge for Exa instead of a raw remote URL entry
- Claude Desktop also supports Exa as a native connector outside this tool's file-mutation path

Current capabilities:

- select an MCP provider from the provider registry
- collect provider-defined credentials through the TUI
- preview redacted config changes before apply
- back up touched files and roll back failed write sequences
- verify updated file state and run optional CLI checks when available
- distribute multiple keys across supported apps
- update JSON and TOML client configs in client-specific formats
- parse Exa API keys from flags, files, or TUI input

## Why Use It

MCP configuration drifts quickly when you use more than one local AI client:

- each client wants a different config shape
- one tool may still point at an old MCP endpoint
- one credential may end up taking all traffic
- manual edits are easy to get wrong and hard to verify

This tool gives you a single flow to detect targets, generate the correct config form for each client, preview changes, apply safely, and verify the result.

Today that flow is Exa-first. The engine underneath is already provider-shaped, so new MCP servers can plug into the same setup, planning, and apply path instead of introducing one-off config mutations for each client.

## Install

Requirements:

- Go `1.23+` for local builds
- macOS for the currently supported config-path workflow

Release distribution currently includes:

- Homebrew formula via `nawodyaishan/homebrew-tap`
- release archives for macOS and Linux
- `deb` and `rpm` packages via GoReleaser/nFPM

Homebrew:

```bash
brew tap nawodyaishan/homebrew-tap
brew install exa-mcp-manager
```

Build locally:

```bash
make build
```

Run the compiled binary:

```bash
./bin/exa-mcp-manager --version
./bin/exa-mcp-manager
```

## Use

Interactive TUI:

```bash
make run
```

Dry run:

```bash
make dry-run KEYS_FILE=~/Downloads/exa_keys.txt
```

Apply without launching the TUI:

```bash
make apply KEYS_FILE=~/Downloads/exa_keys.txt
```

Direct CLI usage:

```bash
./bin/exa-mcp-manager --version
./bin/exa-mcp-manager --keys-file ~/Downloads/exa_keys.txt --dry-run
./bin/exa-mcp-manager --keys-file ~/Downloads/exa_keys.txt --apply
go run ./cmd/exa-mcp-manager
go run ./cmd/exa-mcp-manager --keys-file ~/Downloads/exa_keys.txt --dry-run
go run ./cmd/exa-mcp-manager --keys-file ~/Downloads/exa_keys.txt --apply
```

Current non-interactive flags are still Exa-specific:

- `--keys`
- `--keys-file`
- `--dry-run`
- `--apply`

## Safety Model

This tool edits local developer config, so the design favors correctness over speed.

- full API keys should never appear in UI, logs, dry-run output, apply output, or verification output
- file-backed updates create backups and use rollback-aware writes
- optional CLI verification should not fail the run only because a CLI is missing
- Claude Code is handled through CLI commands rather than direct `~/.claude.json` mutation

## Development

Default workflow:

```bash
make tidy
make vet
make lint
make test
make build
make gitignore-check
```

The repo uses local caches for Go build, module, and lint artifacts through `make`, which keeps the development loop reproducible and avoids polluting global caches.

### Git Hooks

This repo uses [Lefthook](https://github.com/evilmartians/lefthook) as a local guard for the same classes of failures CI should catch.

Install and enable it:

```bash
brew install lefthook
lefthook install
```

Current hooks:

- `pre-commit`: `gofmt` on staged Go files with auto-restaging, `make vet`, and `make gitignore-check`
- `pre-push`: `make lint`, `make test`, `make build`, and `make gitignore-check`

### Adding New MCP Servers

This tool is designed to grow into a universal MCP manager. New MCP servers should be added through the provider abstraction rather than by adding server-specific branches to the app or TUI flow.

Implementation path:

1. Add a provider in `pkg/provider/`, for example `github.go`.
2. Implement `MCPProvider`:
   - `ID()` returns the stable config key, such as `"github"`.
   - `Name()` and `Description()` provide TUI display text.
   - `RequiredCredentials()` describes the credential fields the TUI should collect.
   - `GenerateConfig()` converts credential values into `MCPConfig` using `http`, `sse`, or `stdio`.
3. Register the provider in `DefaultRegistry()` in `pkg/provider/registry.go`.
4. Verify target compatibility. The existing config writers can persist provider-generated `MCPConfig`, but each local client may support a different transport shape.
5. Add focused tests for credential validation, generated config, registry inclusion, redaction, and any target-specific behavior.
6. Update docs when the provider changes user-facing setup, flags, or supported transports.

Once registered, the TUI can list the provider and build credential inputs from `RequiredCredentials()`. Non-interactive CLI flags are still Exa-specific today; provider-neutral CLI arguments are part of the next product step.

For the deeper technical direction, see the [Universal MCP Architecture Plan](docs/arch/universal-mcp-manager-plan.md). For long-term plugin and registry research, see [Future Scalability Research](docs/arch/future-scalability-research.md).

## Project Layout

```text
cmd/exa-mcp-manager/   current CLI entrypoint
pkg/app/          planning, apply flow, rollback, formatting
pkg/config/       path detection and file mutation helpers
pkg/exa/          Exa key parsing, redaction, URL construction
pkg/provider/     provider abstraction and provider implementations
pkg/tui/          Bubble Tea router and TUI screens
pkg/verify/       file and optional CLI verification
docs/                  product, architecture, and phase plans
tests/                 repo-level validation scripts
```

## Architecture Direction

The runtime product is still Exa-first, but the internals now use a provider-based MCP manager shape.

That direction shows up in the code:

- `pkg/provider` defines `MCPProvider`, `MCPConfig`, and transport types
- `pkg/provider.DefaultRegistry()` controls which MCP providers the TUI can offer
- `pkg/config` now mutates client config from provider-generated config rather than raw Exa-only strings
- `pkg/app.PrepareProvider` owns provider-aware planning before apply, rollback, and verification
- `pkg/tui` builds provider and credential setup screens from registry metadata

The next major step is removing the remaining Exa-specific CLI path so interactive and non-interactive usage both run through the same provider-neutral workflow.

## Docs

Primary references:

- [Product Spec](docs/main-spec.md)
- [Next Phase Plan](docs/plans/next-phase.md)
- [Phase 2 Plan](docs/plans/phase2.md)
- [Universal MCP Architecture Plan](docs/arch/universal-mcp-manager-plan.md)
- [Future Scalability Research](docs/arch/future-scalability-research.md)

## Roadmap

Near term:

- remove the remaining Exa-specific non-interactive CLI wiring
- keep Exa behavior backward compatible while shifting to provider-neutral internals
- improve previews, summaries, warnings, and test coverage

After that:

- add stdio-capable providers such as GitHub
- add client capability checks so unsupported transport combinations are skipped safely
- expand from Exa-only rollout into a reusable MCP sync tool
- revisit binary and repository branding once the user-facing product is no longer Exa-only
