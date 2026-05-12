# Linux Support for Universal MCP Sync

## Problem Statement
`usync` currently documents and detects target MCP configuration paths as a macOS-first workflow. Several supported AI clients use different native configuration locations on Linux, especially clients that store settings under XDG-style directories such as `~/.config`. This causes Linux users to preview or apply changes to paths that their local clients do not read.

Linux support must make `usync` detect, preview, write, and verify the correct native MCP configuration files for supported Linux clients while preserving the current macOS behavior.

## Goals
- Support Linux as a first-class target platform for native MCP config sync.
- Preserve current macOS path detection, config writing, backup, redaction, and verification behavior.
- Ensure dry-run and apply output show the correct Linux target paths.
- Keep provider behavior platform-neutral; Linux support should not require provider-specific branches.
- Cover Linux path and config-shape behavior with automated tests.
- Document Linux target paths and any compatibility fallbacks.

## Non-Goals
- Adding Windows support.
- Testing real remote MCP provider APIs.
- Installing or launching target AI clients.
- Changing the interactive TUI workflow beyond showing correct target paths.
- Replacing CLI-managed flows where a client already has a supported CLI integration.
- Adding new MCP providers.

## Users or Actors
- Linux developers using `usync` to configure MCP servers across local AI clients.
- Contributors maintaining target client path detection and config writers.
- Maintainers reviewing cross-platform regressions before release.

## User Journeys
1. **Linux dry-run preview**
   - A Linux user runs `usync --dry-run`.
   - The plan shows Linux-native target paths for selected clients.
   - Missing creatable config files are marked as new files, and no credentials are printed in full.

2. **Linux apply**
   - A Linux user runs `usync --apply`.
   - `usync` writes the provider config to the target Linux client config files.
   - Existing unrelated settings and unrelated MCP servers are preserved.
   - Existing files are backed up before modification.

3. **macOS regression protection**
   - A macOS user runs the same workflow as before.
   - Existing macOS paths and generated config shapes do not change unless separately approved.

4. **Contributor verification**
   - A contributor runs the test suite.
   - Tests validate Linux path detection, platform-specific fixtures, config schemas, idempotency, and merge behavior without requiring real Linux AI clients.

## Functional Requirements
- **FR-1**: `usync` must detect target config paths using the current platform.
- **FR-2**: Linux path detection must use Linux-native locations for supported clients.
- **FR-3**: macOS path detection must remain backward-compatible with the current documented paths.
- **FR-4**: Dry-run plans must display platform-correct paths and backup behavior.
- **FR-5**: Apply must create missing Linux parent directories and files for creatable targets using private permissions consistent with existing behavior.
- **FR-6**: Apply must preserve unrelated client settings and unrelated MCP servers.
- **FR-7**: Existing verification behavior must work for Linux target paths.
- **FR-8**: CLI-managed clients must remain CLI-managed when that is the existing behavior.
- **FR-9**: Provider config generation must remain platform-neutral.
- **FR-10**: Linux support must include automated tests for all supported target clients that have Linux paths.

## Linux Target Expectations
- Claude Desktop: user config under `~/.config/Claude/claude_desktop_config.json`.
- Claude Code: continue to use the `claude mcp` CLI flow when available.
- Cursor: use `~/.cursor/mcp.json`.
- VS Code: use a Linux-native user MCP config location rather than a macOS application-support path.
- Windsurf: support Linux user MCP config locations and preserve compatibility for existing current repo path users.
- Zed: use `~/.config/zed/settings.json`.
- Roo Code: use the Linux VS Code extension storage location for global MCP settings.
- OpenCode: use `~/.config/opencode/opencode.json`.
- Kiro: use `~/.kiro/settings/mcp.json`.
- Gemini CLI: use `~/.gemini/settings.json` for user MCP settings.
- Antigravity: use `~/.gemini/antigravity/mcp_config.json`.
- Codex CLI: use `~/.codex/config.toml`.

## Acceptance Criteria
- **AC-1**: Linux path detection returns Linux-native paths for every supported Linux target.
- **AC-2**: macOS path detection returns the same paths currently used by the repo.
- **AC-3**: A dry-run on a Linux-resolved home directory shows Linux target paths and no full credentials.
- **AC-4**: Applying to a Linux-resolved home directory creates missing creatable files and backs up existing files.
- **AC-5**: Existing manual MCP servers and unrelated settings survive Linux apply.
- **AC-6**: Re-running apply on Linux is idempotent and does not duplicate provider entries.
- **AC-7**: OpenCode output matches the current OpenCode MCP schema for both remote and local providers.
- **AC-8**: Test coverage includes unit tests, manager/app tests, and e2e or golden tests for Linux behavior.
- **AC-9**: README or equivalent docs list Linux support and supported Linux target paths.
- **AC-10**: `make test` and `make test-e2e` pass after implementation.

## Success Criteria
- Linux users can configure the same provider set as macOS users for clients available on Linux.
- Cross-platform path behavior is deterministic and testable without relying on the host OS.
- Future target-client path changes can be added without provider-specific branching.
- The Linux implementation does not regress existing provider, redaction, backup, rollback, or verification tests.

## Edge Cases
- A Linux user has an existing legacy Windsurf config path and no newer documented config path.
- A Linux user has both legacy and current candidate paths for the same client.
- A config file exists but contains invalid JSON or TOML.
- A parent directory is missing.
- A target client is selected but its CLI prerequisite is missing.
- A provider uses stdio transport and a target supports only remote transports.
- A provider requires Docker but Docker is not installed or not running.
- Existing configs contain a malformed provider entry.
- A target config path is outside the user home directory.

## Data Sensitivity and Compliance Notes
- No full API keys, tokens, provider URLs containing secrets, or generated CLI arguments containing secrets may be printed in docs, tests, logs, dry-run output, or verification output.
- Test fixtures must use fake credentials only.
- Config files and backups should continue to be written with private file permissions.

## API or Integration Expectations
- The public CLI behavior should remain stable.
- Existing provider interfaces should not change solely for Linux support.
- Existing config writer contracts should remain stable unless a target client schema requires a dedicated writer.
- Platform detection must be testable without requiring tests to run on Linux.

## Testing Requirements
- Unit tests must cover platform-specific path detection for macOS and Linux.
- Unit tests must cover target-client schema differences affected by Linux support.
- App-level tests must cover dry-run/prepare/apply behavior using Linux-resolved paths.
- E2E or golden tests must validate generated Linux config files for representative remote and stdio providers.
- Tests must assert that credentials remain redacted in user-visible output.
- Tests must assert idempotency and preservation of unrelated config data.

## Assumptions
- Linux support is scoped to user-level config files, not workspace-level config files, unless a target client only supports workspace-level MCP settings.
- Linux target paths may be resolved by platform plus explicit compatibility candidates where clients have changed documented paths.
- Linux support can be validated with temporary home directories and platform-injected path resolution.
- Existing macOS users should not see path changes in this feature.

## Open Questions
- Should VS Code default to user-level `~/.config/Code/User/mcp.json`, workspace `.vscode/mcp.json`, or support both with an explicit target selection later?
- For Windsurf, should new Linux files be created at `~/.codeium/mcp_config.json` or the existing repo path `~/.codeium/windsurf/mcp_config.json` when neither exists?
- Should VSCodium and VS Code Insiders extension storage paths be included in this feature or deferred?

## Human Approval Status
- Pending maintainer review and approval before technical planning.
