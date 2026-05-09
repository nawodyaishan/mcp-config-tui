# MCP Config TUI

[![Release](https://img.shields.io/github/v/release/nawodyaishan/mcp-config-tui?display_name=tag)](https://github.com/nawodyaishan/mcp-config-tui/releases)
[![CI](https://github.com/nawodyaishan/mcp-config-tui/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/nawodyaishan/mcp-config-tui/actions/workflows/ci.yml)
[![License](https://img.shields.io/github/license/nawodyaishan/mcp-config-tui)](./LICENSE)

`mcp-config-tui` is a macOS-first utility for developers who use multiple local AI tools and want one safe way to manage MCP configuration across them.

The current shipped CLI is `exa-mcp-manager`. Today it automates Exa MCP rollout. The codebase is already being refactored toward a provider-based MCP sync tool, but the user-facing product is still Exa-first.

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

Current supported app targets on macOS:

- Claude Desktop
- Claude Code
- Gemini CLI
- Antigravity
- Codex CLI

Current provider support:

- Exa

Current capabilities:

- parse Exa API keys from flags, files, or TUI input
- distribute multiple keys across supported apps
- preview redacted changes before apply
- update JSON and TOML client configs in client-specific formats
- back up touched files
- roll back file updates if a later file write fails
- verify updated file state and run optional CLI checks when available

## Why Use It

MCP configuration drifts quickly when you use more than one local AI client:

- each client wants a different config shape
- one tool may still point at an old MCP endpoint
- one credential may end up taking all traffic
- manual edits are easy to get wrong and hard to verify

This tool gives you a single flow to detect targets, generate the correct config form for each client, preview changes, apply safely, and verify the result.

## Install

Requirements:

- Go `1.23+` for local builds
- macOS for the currently supported config-path workflow

Release distribution currently includes:

- Homebrew formula via `nawodyaishan/homebrew-tap`
- release archives for macOS, Linux, and Windows
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

## Project Layout

```text
cmd/exa-mcp-manager/   current CLI entrypoint
internal/app/          planning, apply flow, rollback, formatting
internal/config/       path detection and file mutation helpers
internal/exa/          Exa key parsing, redaction, URL construction
internal/provider/     provider abstraction and provider implementations
internal/tui/          Bubble Tea router and TUI screens
internal/verify/       file and optional CLI verification
docs/                  product, architecture, and phase plans
tests/                 repo-level validation scripts
```

## Architecture Direction

The runtime product is still Exa-first, but the internals are moving toward a provider-based MCP manager.

That direction already shows up in the code:

- `internal/provider` defines `MCPProvider`, `MCPConfig`, and transport types
- `internal/config` now mutates client config from provider-generated config rather than raw Exa-only strings
- `internal/app` owns planning, apply, rollback, and verification orchestration

The next major step is a provider registry and dynamic credential-driven TUI setup so Exa can run through the same path future providers will use.

## Docs

Primary references:

- [Product Spec](docs/exa-mcp-manager-spec.md)
- [Next Phase Plan](docs/next-phase-plan.md)
- [Phase 2 Plan](docs/specs/phase2-plan.md)
- [Universal MCP Architecture Plan](docs/specs/universal-mcp-manager-plan.md)

## Roadmap

Near term:

- complete the provider registry and dynamic credential-driven setup flow
- remove the remaining Exa-specific wiring from planning and TUI setup
- keep Exa behavior backward compatible while shifting to provider-neutral internals
- improve previews, summaries, warnings, and test coverage

After that:

- add stdio-capable providers such as GitHub
- add client capability checks so unsupported transport combinations are skipped safely
- expand from Exa-only rollout into a reusable MCP sync tool
- revisit binary and repository branding once the user-facing product is no longer Exa-only
