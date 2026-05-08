# Exa MCP Config Manager

`exa-mcp-manager` is a macOS-first developer utility for wiring Exa MCP into the AI tools you already use.

Instead of updating Claude Desktop, Claude Code, Gemini CLI, Antigravity, and Codex by hand, this project gives you one place to load keys, distribute them across apps, preview changes, apply safely, and verify the result.

## Why This Exists

If you use multiple local AI tools, MCP setup tends to drift:

- one app points at an old server URL
- another has a stale tool list
- one key is getting all the traffic
- one config is right, but the others are not

This project is meant to remove that friction for developers who want a repeatable local setup instead of a pile of one-off edits.

## Current Scope

The current implementation targets these apps on macOS:

- Claude Desktop
- Claude Code
- Gemini CLI
- Antigravity
- Codex CLI

It currently supports:

- UUID-style Exa key parsing from flags, files, and TUI input
- redacted previews instead of full key output
- multi-app key assignment
- JSON and TOML config updates for supported apps
- timestamped backups
- transactional file updates with rollback on later file-write failure
- optional CLI verification for `codex` and `claude`

Reference docs:

- [Product Spec](docs/exa-mcp-manager-spec.md)
- [Next Phase Plan](docs/next-phase-plan.md)

## Developer Experience

This repo is built for developers who want to work on the tool itself as well as developers who want to use it locally.

What matters here:

- Bubble Tea TUI for the interactive flow
- standard library config mutation and verification
- repo-local Go cache defaults in the `Makefile`
- tests that cover rollback, redaction, verification semantics, and config fixtures

## Quick Start

Requirements:

- Go `1.22+`
- macOS target environment

Common flows:

```bash
make tidy
make test
make build
make run
```

Non-interactive examples:

```bash
make dry-run KEYS_FILE=~/Downloads/exa_keys.txt
make apply KEYS_FILE=~/Downloads/exa_keys.txt
```

Direct CLI usage:

```bash
go run ./cmd/exa-mcp-manager
go run ./cmd/exa-mcp-manager --keys-file ~/Downloads/exa_keys.txt --dry-run
go run ./cmd/exa-mcp-manager --keys-file ~/Downloads/exa_keys.txt --apply
```

## Project Layout

```text
cmd/exa-mcp-manager/   CLI entrypoint
internal/app/          orchestration, apply flow, rollback, formatting
internal/config/       path detection and file mutation helpers
internal/exa/          key parsing, redaction, Exa URL construction
internal/tui/          Bubble Tea screens and interaction flow
internal/verify/       config and optional CLI verification
docs/                  spec and implementation plans
tests/                 repo-level validation scripts
```

## Safety Notes

The tool is built around local developer configs, so safety matters more than speed.

- Full API keys should never be shown in UI, logs, dry-run output, or verification output.
- File-backed updates use backups and rollback-aware writes.
- Optional CLI verification should not fail the run just because a CLI is not installed.
- Claude Code is still handled through CLI commands rather than direct `~/.claude.json` mutation.

## Roadmap

Short-term roadmap:

- richer apply summaries with rollback diagnostics
- stronger logging controls for troubleshooting
- better fixture-driven acceptance coverage across app variations
- improved TUI affordances around key distribution and warnings

Future MCP support beyond Exa:

- configurable MCP server templates instead of hard-coded Exa-only wiring
- Context7 setup management
- filesystem and local tool MCP presets
- browser and automation MCP presets
- multi-server profile management per app
- pluggable support for additional MCP vendors without changing the core apply engine

Broader platform roadmap:

- Linux path support where the target apps have stable config locations
- Windows support if config-path behavior is predictable enough to automate safely
- export/import of reusable MCP rollout profiles

## Contributing

Use the `Makefile` targets as the default workflow. The repository is set up so `go test` and `go build` use repo-local cache directories through `make`, which avoids polluting global caches and keeps the development loop reproducible inside constrained environments.
