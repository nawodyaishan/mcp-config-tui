# Next Phase Plan: Reliability, Logging, and Security Hardening

## Summary

The current Exa MCP Config Manager is functionally scaffolded, but the next phase must make real config mutation safer before users run it against live macOS app configs. This phase addresses three review findings:

- High: apply can leave a machine partially updated if a later target fails.
- Medium: missing optional CLIs are reported as failed verification.
- Medium: the TUI can display full API keys loaded from `--keys` or `--keys-file`.

This plan hardens apply behavior, error handling, logging, and key redaction while keeping the existing package boundaries: `pkg/app` orchestrates, `pkg/config` mutates files, `pkg/verify` verifies, and `pkg/tui` renders the user flow.

## Review Findings

1. Multi-target apply is not rollback-safe.
   - Current behavior mutates targets sequentially and returns on the first error.
   - Earlier configs remain changed if a later file write or CLI operation fails.
   - This is risky because one failed target can leave Claude Desktop, Gemini, Antigravity, and Codex in different states.

2. Optional CLI verification is too strict.
   - `codex mcp get exa` and `claude mcp get exa` are optional checks in the spec.
   - Missing CLIs should not mark verification as failed when file verification passes.
   - Missing CLIs should be reported as warning or skipped status.

3. TUI key handling can expose secrets.
   - Raw key material loaded from `--keys` or `--keys-file` can be seeded into the textarea and rendered.
   - The spec requires full keys to never appear in UI, logs, dry-run output, or verification output.
   - All user-facing output should show only redacted labels such as `6ea4...7887`.

## Error Handling Plan

- Add an apply preflight phase before any mutation:
  - validate selected operations and key assignments
  - read all existing file targets
  - generate all updated file contents in memory
  - validate parent directories can be created or written
  - validate backup paths are available for existing files
- Make file apply transactional at the file-operation level:
  - backup existing files before writing
  - write new content through a temp file in the same directory
  - atomically rename temp files into place
  - track every changed file and backup path in apply metadata
- Add rollback for file targets:
  - if a later file target fails, restore previous files from their backups
  - if a newly created file must be rolled back, remove only files created by this apply run
  - report rollback success/failure per path
- Treat Claude Code CLI as explicitly non-rollbackable:
  - keep direct `~/.claude.json` mutation skipped
  - run `claude mcp remove/add` after file-backed targets succeed
  - report CLI failure clearly without pretending file rollback covers it
- Improve user-facing errors:
  - include app name, target label, and path for file errors
  - never include full Exa URLs or full keys in errors
  - include recovery instructions when rollback fails

## Logging Plan

- Use standard-library `log/slog`; do not add a third-party logging dependency.
- Inject a logger into `app.Manager` so tests can capture logs.
- Define log levels:
  - `debug`: preflight details, target enumeration, generated operation counts
  - `info`: apply start, target updated, verification completed
  - `warn`: optional CLI unavailable, rollback attempted, recoverable skips
  - `error`: apply failure, rollback failure, verification failure
- Keep diagnostic logs separate from normal CLI/TUI output.
- Redact before logging:
  - raw UUID-style keys
  - `exaApiKey` query values
  - full Exa MCP URLs
  - command output from `codex` and `claude`
- Add a single redaction helper used by CLI formatting, TUI rendering, logging, and verification summaries.

## Security Hardening Plan

- Never render full keys in TUI:
  - if keys are loaded from flags or files, show only parsed redacted labels
  - keep the textarea empty unless the user is manually entering keys
  - after parsing manual entry, clear or replace raw input with redacted summary before moving screens
- Avoid printing full Exa MCP URLs:
  - plan and apply output should show host/tool count/key label, not the full URL
  - errors should reference target paths and operation names, not secret-bearing URLs
- Tighten file permissions:
  - write configs, backups, and temp files with `0600`
  - keep parent directories at `0700` where newly created under user home
  - preserve existing file permissions only if doing so does not make secret-bearing configs world-readable
- Validate target paths:
  - ensure detected paths stay under the configured home directory
  - reject empty paths and paths that resolve outside the configured home override
- Minimize secret lifetime:
  - avoid storing raw keys in long-lived TUI model fields once URLs are generated
  - do not persist raw keys to logs, errors, test output, or snapshots

## Implementation Changes

- Extend apply result types:
  - add rollback status metadata
  - distinguish file updates, file creations, skipped operations, and external CLI operations
  - include warnings separately from failures
- Extend verification result types:
  - replace boolean-only status with `ok`, `warning`, `skipped`, and `failed`
  - treat unavailable optional CLIs as `skipped`
  - keep failed file verification as `failed`
- Refactor write helpers:
  - add atomic write with temp file and rename
  - return enough metadata for rollback
  - use stricter file permissions
- Refactor TUI key flow:
  - keep raw typed input only while on the key entry screen
  - render parsed key labels from redacted values only
  - never initialize the textarea with flag/file contents
- Add logging:
  - wire `slog.Logger` through the manager
  - add redaction at logger boundary
  - test log output for secret leakage

## Test Plan

- Unit tests:
  - forced second-file write failure restores first file from backup
  - newly created files are removed during rollback if apply fails later
  - rollback failure is reported with path-level metadata
  - missing `codex` and `claude` CLIs produce skipped verification, not failed verification
  - TUI initialized from `--keys` or `--keys-file` does not render full keys
  - logs redact UUID-style keys and `exaApiKey` URL values
  - config, backup, and temp writes use `0600`
- Fixture tests:
  - successful apply across Claude Desktop, Gemini settings, Gemini MCP config, Antigravity, and Codex
  - failed apply after partial file success with rollback verification
  - Claude Code CLI unavailable path remains warning/skipped only
  - malformed existing Exa URL is replaced without leaking the old value
- Regression checks:
  - `go test ./...`
  - `go build ./...`
  - `bash tests/gitignore_test.sh`

## Acceptance Criteria

- A failed file apply leaves all previously existing config files in their original content state.
- Missing optional CLIs do not cause overall verification failure.
- No full UUID-style Exa key appears in TUI views, CLI plan output, apply output, verification details, errors, or logs.
- No full Exa MCP URL with `exaApiKey` appears in user-facing or diagnostic output.
- File-backed config writes and backups are owner-readable/writable only.
- The full test suite and gitignore validation pass.

## Assumptions

- Optional CLI checks are advisory and should be represented as skipped when the binary is missing.
- File-backed app configs should be rollback-capable; Claude Code CLI operations are external and explicitly non-rollbackable.
- `log/slog` is sufficient for this tool and avoids another dependency.
- The next implementation phase should prioritize safety and secrecy over expanding app support or adding new MCP tools.
