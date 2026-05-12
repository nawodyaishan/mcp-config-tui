# Linux Support Implementation Plan

## Summary
Add Linux support by making target config path detection platform-aware, preserving current macOS behavior, and tightening target-specific config writers where the Linux work exposes schema drift. The change stays in the config/app/test/docs layers and does not alter provider interfaces or provider-generated MCP configs.

## Inputs Reviewed
- `docs/specs/linux-support/spec.md`
- `pkg/config/paths.go`
- `pkg/config/paths_test.go`
- `pkg/config/json_update.go`
- `pkg/config/toml_update.go`
- `pkg/app/app.go`
- `pkg/client/capabilities.go`
- `tests/e2e/e2e_test.go`
- `README.md`
- Official/current client documentation reviewed during research:
  - VS Code MCP configuration
  - Windsurf MCP raw config
  - Zed MCP settings
  - OpenCode config and MCP server schema
  - Kiro MCP configuration
  - Gemini CLI MCP docs
  - Codex MCP docs
  - Linux MCP client configuration examples

## Assumptions
- Linux support targets user-level config files.
- Workspace-scoped config management is deferred.
- Existing macOS paths must not change.
- Tests must be able to validate Linux behavior on non-Linux developer machines.
- No new runtime dependency is needed.
- No provider API changes are needed.

## Architecture Approach
1. **Introduce platform-aware path resolution**
   - Keep `DetectAppConfigs(home string)` as the existing public entry point.
   - Add a testable internal/public helper such as `DetectAppConfigsForOS(home, goos string)` or `DetectAppConfigsWithOptions`.
   - `DetectAppConfigs` calls the helper with `runtime.GOOS`.
   - Preserve current macOS path output exactly.
   - Add Linux path output for all supported targets.

2. **Centralize per-platform path candidates**
   - Model path decisions as data, not conditionals scattered through the app.
   - For targets with compatibility candidates, choose the first existing candidate.
   - If none exist, choose the default creation candidate.
   - Initial compatibility candidates:
     - Windsurf Linux: `~/.codeium/mcp_config.json`, `~/.codeium/windsurf/mcp_config.json`
   - Deferred candidates:
     - VSCodium
     - VS Code Insiders
     - workspace `.vscode/mcp.json`

3. **Keep providers platform-neutral**
   - `provider.MCPProvider.GenerateConfig` remains unchanged.
   - `client.Adapt` and capability checks remain transport-focused, not OS-focused.

4. **Fix OpenCode output schema**
   - Add a dedicated writer path for OpenCode rather than relying on generic named-server output.
   - Remote MCP config:
     - root key: `mcp`
     - server fields: `type: "remote"`, `url`, optional `headers`, `enabled: true`
   - Local MCP config:
     - root key: `mcp`
     - server fields: `type: "local"`, `command: ["cmd", "...args"]`, optional environment map, `enabled: true`
   - Preserve unrelated OpenCode settings and unrelated MCP entries.

5. **Make app-level tests OS-injectable**
   - Avoid adding a user-visible `--target-os` flag.
   - Unit and app tests should call the OS-aware path resolver directly.
   - If e2e golden coverage needs Linux path output without running on Linux, add a test helper that constructs `app.Manager` with injected `Apps` rather than invoking the CLI binary.

6. **Update docs**
   - README supported target table should identify Linux support and Linux paths.
   - Note path compatibility behavior for Windsurf.
   - Note deferred scope for VS Code workspace/VSCodium/Insiders.

## Affected Modules
- `pkg/config/paths.go`
  - Add platform-aware path resolution and candidate path support.
- `pkg/config/paths_test.go`
  - Add macOS regression tests and Linux path tests.
- `pkg/config/json_update.go`
  - Add or support a dedicated OpenCode config builder.
- `pkg/config/json_update_test.go`
  - Add OpenCode remote/local schema tests.
- `pkg/app/app.go`
  - Wire any new `FileKind` or app-specific writer case.
- `pkg/app/app_test.go`
  - Add Linux prepare/apply tests.
- `pkg/app/qa_scenarios_test.go`
  - Update OpenCode expectations if schema changes affect current tests.
- `tests/e2e/e2e_test.go`
  - Add OS-specific fixture helper or keep CLI e2e macOS-shaped and add Linux golden coverage at app level.
- `README.md`
  - Document Linux target support.

## API and Contract Changes
- No provider interface changes.
- No public CLI flag changes.
- Possible internal config contract addition:
  - New `FileKindOpenCode`, or an equivalent writer selector, to avoid encoding OpenCode-specific schema through generic named-server extras.
- Possible test helper contract:
  - `DetectAppConfigsForOS(home, goos string)` if exported for tests.

## Data Model Changes
- No persisted `usync` data model changes.
- Target path metadata may gain candidate/default semantics internally.

## Dependency Changes
- No new dependency is planned.

## Security Impact
- Credential redaction requirements remain unchanged.
- Linux config files and backups must continue to be written with private permissions.
- Tests must use fake credentials only.
- Platform path resolution must keep the existing home-boundary validation before writes.

## Authorization Boundaries
- No auth or authorization flow changes.
- Claude Code remains CLI-managed; missing `claude` CLI should warn and skip as it does now.

## Observability Impact
- Dry-run and apply output should show platform-correct target paths.
- Existing redacted output behavior should not reveal full secrets.
- No additional logs are required.

## Testing Strategy
- Unit test path detection for macOS and Linux.
- Unit test compatibility candidate selection.
- Unit test OpenCode schema generation for remote and local providers.
- App-level test Linux dry-run/prepare/apply behavior.
- Golden-style tests for Linux-generated configs using injected Linux app paths.
- Existing full test suite must remain green.

Detailed coverage is defined in `docs/specs/linux-support/test-plan.md`.

## Failure Modes
- Unsupported `goos` value returns a clear error or falls back to current Unix-like paths by design.
- Existing invalid JSON/TOML returns the existing parse error path and does not write a partial update.
- Missing parent directories are created with private directory permissions.
- Existing config files are backed up before writes.
- Candidate path ambiguity is resolved deterministically by first existing candidate, then default creation candidate.

## Rollback and Recovery
- Runtime rollback remains the existing backup and rollback behavior in `pkg/app`.
- Code rollback is isolated to config path detection, OpenCode writer logic, docs, and tests.
- If OpenCode schema changes are risky, they can be split into a separate PR, but Linux support should not claim OpenCode correctness until that is fixed.

## Risks and Mitigations
- **Risk**: Linux target docs change or conflict across client versions.
  **Mitigation**: Keep path resolution candidate-based where evidence conflicts, and document defaults.
- **Risk**: Existing macOS users see changed paths.
  **Mitigation**: Add explicit macOS regression path tests.
- **Risk**: Linux behavior is hard to test from macOS CI.
  **Mitigation**: Inject `goos` into path resolver tests and use app-level golden tests with injected app configs.
- **Risk**: OpenCode schema fix changes current golden files.
  **Mitigation**: Add focused OpenCode tests and update golden files intentionally.

## Human Architecture Approval Status
- Approved for implementation on 2026-05-12.
