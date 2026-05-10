# Contributing to Universal MCP Sync

Thank you for your interest in contributing. This project aims to make MCP configuration repeatable, reviewable, and safe across local AI assistants.

## Technical Vision
Universal MCP Sync started as an Exa-focused utility and now uses a **provider-based architecture**. New MCP servers should fit the generic provider/client pipeline instead of adding provider-specific branches to the TUI or apply flow.

## How to Contribute

### 1. Adding a New MCP Provider
If you want to add support for a new MCP server, start with [Adding a Provider Guide](docs/contributors/adding-a-provider.md). It covers interface scaffolding, credential validation, config generation, redaction, registry inclusion, and per-client adaptation through `pkg/client.Adapt()` and `pkg/client.HeadersFor()`.

### 2. Adding a New AI Client
If a new AI assistant with local MCP support is released:
1. **Add the app ID and path** in `pkg/config/paths.go`.
2. **Declare transport support** in `pkg/client/capabilities.go`.
3. **Add adaptation rules** in `pkg/client/adapter.go` when the client needs a bridge, headers, or transport-specific fields.
4. **Update config writers** in `pkg/config/` only when the client persists a new JSON/TOML shape.
5. **Add QA coverage** in `pkg/app/qa_scenarios_test.go` so existing providers keep working across the matrix.

## Development Workflow

### Requirements
- Go 1.23+
- macOS (for config path detection)
- `golangci-lint` for `make lint`
- [Lefthook](https://github.com/evilmartians/lefthook) (optional but recommended)

### First-Time Setup
```bash
go mod download
make build
./bin/usync --help
```

### Commands
```bash
make tidy          # sync module dependencies
make tidy-check    # verify go.mod and go.sum are already tidy
make mod-verify    # verify downloaded module checksums
make fmt           # format Go packages
make vet           # run go vet
make lint          # run golangci-lint
make test          # run all tests
make build         # build ./bin/usync
make dry-run KEYS_FILE=~/Downloads/exa_keys.txt
```

## Testing Standards
Reliability is our primary feature. We never ship a feature without verification:
- **Unit tests**: Cover credential parsing, generated `MCPConfig`, transport support, and config mutation.
- **Scenario tests**: Update `pkg/app/qa_scenarios_test.go` so each provider/client shape remains compatible.
- **Redaction tests**: Ensure UI output, logs, snapshots, and failures never leak raw credentials, secret URLs, or generated CLI args containing secrets.
- **Rollback tests**: Preserve backup and rollback behavior when touching apply logic or config writers.

Before opening a PR, run:
```bash
make fmt
make test
make gitignore-check
```

Run `make lint` and `make verify` when changing shared logic, release tooling, or provider/client compatibility.

## Pull Request Process
1. Create a branch with a Conventional Commit-style topic, for example `feat/context7-provider` or `fix/zed-headers`.
2. Keep provider, client, config writer, and TUI changes separated when practical.
3. Include or update tests for the behavior you changed.
4. Describe the problem, the change, and the verification commands you ran.
5. Include terminal output or screenshots only when TUI behavior changed.
