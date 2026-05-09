# Contributing to Universal MCP Sync

Thank you for your interest in contributing! This project aims to be the standard way to synchronize MCP configurations across all local AI assistants.

## Technical Vision
We are transitioning from an Exa-only utility to a **Provider-based Architecture**. This allows any MCP server to be added by implementing a single interface and registering it in the registry.

## How to Contribute

### 1. Adding a New MCP Provider
If you want to add support for a new MCP server (e.g., GitHub, Postgres, Notion):
1.  **Implement the Interface**: Create a new file in `pkg/provider/` (e.g., `github.go`) and implement the `MCPProvider` interface.
2.  **Define Credentials**: Use `RequiredCredentials()` to specify which tokens or paths the TUI should collect.
3.  **Register**: Add your new provider to `DefaultRegistry()` in `pkg/provider/registry.go`.
4.  **Client Compatibility**: If the provider requires specific transport shapes (like stdio wrapping), update `configForTarget` in `pkg/app/app.go`.

### 2. Adding a New AI Client
If a new AI assistant with local MCP support is released:
1.  **Add Path**: Update `pkg/config/paths.go` with the default configuration path.
2.  **Add Rules**: If the client uses non-standard JSON keys or requires extra fields, update `prepareFileOperation` in `pkg/app/app.go`.

## Development Workflow

### Requirements
- Go 1.23+
- macOS (for config path detection)
- [Lefthook](https://github.com/evilmartians/lefthook) (optional but recommended)

### Commands
```bash
make tidy    # Clean up dependencies
make vet     # Run go vet
make lint    # Run golangci-lint
make test    # Run all tests
make build   # Build the usync binary
```

## Testing Standards
Reliability is our primary feature. We never ship a feature without verification:
-   **Unit Tests**: For individual logic components.
-   **Scenario Tests**: Update `pkg/app/qa_scenarios_test.go` to ensure your changes don't break the "Golden Path" for other clients.
-   **Redaction**: Ensure no tests or logs leak raw secrets.

## Pull Request Process
1. Create a branch for your feature or fix.
2. Ensure `make test` passes.
3. Submit a PR with a clear description of the changes.
4. Every PR must include or update relevant tests.
