# Linux Support Test Plan

## Verification Commands
- `make fmt`
- `make test`
- `make test-e2e`
- `make coverage-check`
- `make gitignore-check` if new golden fixtures are added

## Unit Test Coverage
### Path Detection
- `TestDetectAppConfigsForOS_DarwinPreservesCurrentPaths`
  - Asserts current macOS paths for Claude Desktop, Roo Code, OpenCode, VS Code, Windsurf, and existing cross-platform paths.
- `TestDetectAppConfigsForOS_LinuxUsesNativePaths`
  - Asserts Linux paths for Claude Desktop, VS Code, Roo Code, OpenCode, Windsurf, Zed, Kiro, Gemini CLI, Antigravity, Codex CLI, Cursor, and Claude Code.
- `TestDetectAppConfigsForOS_EmptyHomeErrors`
  - Preserves existing missing-home behavior.
- `TestDetectAppConfigsForOS_LinuxMarksExistingFiles`
  - Creates selected Linux files and asserts `Exists` is true.
- `TestDetectAppConfigsForOS_LinuxWindsurfPrefersExistingDefaultPath`
  - If `~/.codeium/mcp_config.json` exists, it is selected.
- `TestDetectAppConfigsForOS_LinuxWindsurfPrefersExistingLegacyPath`
  - If only `~/.codeium/windsurf/mcp_config.json` exists, it is selected.
- `TestDetectAppConfigsForOS_LinuxWindsurfDefaultsToDefaultPath`
  - If neither exists, default creation path is `~/.codeium/mcp_config.json`.

### OpenCode Writer
- `TestUpdateOpenCodeJSON_RemoteProvider`
  - Expects `mcp.<provider>.type = remote`, `url`, `headers`, `enabled = true`.
- `TestUpdateOpenCodeJSON_LocalProvider`
  - Expects `mcp.<provider>.type = local`, `command` array containing command and args, environment fields, `enabled = true`.
- `TestUpdateOpenCodeJSON_PreservesUnrelatedSettings`
  - Existing model/provider/settings keys survive.
- `TestUpdateOpenCodeJSON_PreservesOtherMCPServers`
  - Existing unrelated MCP entries survive.
- `TestUpdateOpenCodeJSON_ReplacesProviderOnly`
  - Existing same-provider entry is replaced exactly once.

## App-Level Coverage
- `TestManagerPrepareProvider_LinuxPaths`
  - Manager/app setup using Linux-resolved app configs produces operations pointing at Linux paths.
- `TestFormatPlan_LinuxPathsRedacted`
  - Dry-run plan includes Linux paths and does not include full credential values.
- `TestApply_LinuxCreatesMissingTargets`
  - Missing Linux target files are created and updated.
- `TestApply_LinuxBacksUpExistingTargets`
  - Existing files receive backup paths with the existing suffix format.
- `TestApply_LinuxPreservesManualServers`
  - Existing unrelated MCP servers remain after apply.
- `TestApply_LinuxIdempotent`
  - Two applies do not duplicate provider entries.
- `TestApply_LinuxOpenCodeRemoteAndLocal`
  - Remote and stdio providers produce correct OpenCode config shape through the full app path.

## E2E or Golden Coverage
- Keep existing CLI e2e coverage for host-platform behavior.
- Add Linux golden-style coverage using injected Linux app paths if CLI execution cannot safely spoof `runtime.GOOS`.
- Suggested scenarios:
  - `linux_provider_exa_default`
  - `linux_provider_context7_remote_headers`
  - `linux_provider_playwright_stdio`
  - `linux_edge_case_merging`
  - `linux_edge_case_idempotency`

## Regression Coverage
- Existing provider registry tests must pass.
- Existing client capability tests must pass.
- Existing redaction tests must pass.
- Existing JSON/TOML mutation tests must pass.
- Existing rollback tests must pass.
- Existing e2e golden tests must pass or be intentionally updated for OpenCode schema correction.

## Coverage Boundaries
- Do not require real Linux AI clients to be installed.
- Do not call remote MCP provider APIs.
- Do not use real credentials.
- Do not add a user-visible target OS flag solely for testing.
