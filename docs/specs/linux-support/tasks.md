# Linux Support Tasks

## Track Summary
Implement Linux support as a config-path and target-writer update, preserving provider behavior and existing macOS behavior.

## Prerequisites
- `docs/specs/linux-support/spec.md` is approved for planning.
- `docs/specs/linux-support/plan.md` is approved for implementation.
- `docs/specs/linux-support/test-plan.md` defines verification coverage.

## Task List

### LNX-1: Platform-Aware Target Path Detection
- **Objective**: Add OS-aware app config path detection with Linux-native paths and macOS regression safety.
- **Source Artifacts**: `spec.md`, `plan.md`, `test-plan.md`.
- **Allowed Files or Directories**: `pkg/config/paths.go`, `pkg/config/paths_test.go`.
- **Forbidden Files or Directories**: Provider packages, dependency files, production config.
- **Acceptance Criteria**:
  - `DetectAppConfigs` preserves its public behavior.
  - A testable OS-aware resolver exists.
  - Linux paths match the spec defaults.
  - macOS paths match current behavior.
  - Windsurf Linux candidate selection is deterministic.
- **Verification**: `go test ./pkg/config`.
- **Dependencies**: None.
- **Risk Level**: Low.
- **Approval Needed**: No.
- **Status**: Approved.

### LNX-2: OpenCode Schema Writer
- **Objective**: Emit official OpenCode MCP schema for remote and local providers.
- **Source Artifacts**: `spec.md`, `plan.md`, `test-plan.md`.
- **Allowed Files or Directories**: `pkg/config/json_update.go`, `pkg/config/json_update_test.go`, `pkg/config/paths.go`, `pkg/app/app.go`.
- **Forbidden Files or Directories**: Provider packages, dependency files.
- **Acceptance Criteria**:
  - Remote OpenCode entries use `type: "remote"`, `url`, optional `headers`, and `enabled: true`.
  - Local OpenCode entries use `type: "local"`, `command` array, optional environment map, and `enabled: true`.
  - Existing unrelated OpenCode settings and MCP entries are preserved.
- **Verification**: `go test ./pkg/config ./pkg/app`.
- **Dependencies**: LNX-1.
- **Risk Level**: Medium.
- **Approval Needed**: No.
- **Status**: Approved.

### LNX-3: App-Level Linux Coverage
- **Objective**: Add manager/app tests for Linux prepare/apply behavior, redaction, idempotency, and merge preservation.
- **Source Artifacts**: `test-plan.md`.
- **Allowed Files or Directories**: `pkg/app/app_test.go`, `pkg/app/qa_scenarios_test.go`, `tests/e2e/e2e_test.go`, `tests/e2e/testdata/`.
- **Forbidden Files or Directories**: Provider implementation files unless a test reveals a provider bug.
- **Acceptance Criteria**:
  - Linux paths appear in prepared operations.
  - Apply creates and backs up Linux target files.
  - OpenCode is covered through app-level apply.
  - Existing host-platform e2e tests remain valid or are intentionally updated.
- **Verification**: `go test ./pkg/app ./tests/e2e/...`.
- **Dependencies**: LNX-1, LNX-2.
- **Risk Level**: Medium.
- **Approval Needed**: No.
- **Status**: Approved.

### LNX-4: Documentation Update
- **Objective**: Document Linux support and supported target paths.
- **Source Artifacts**: `spec.md`, `plan.md`.
- **Allowed Files or Directories**: `README.md`, `docs/specs/linux-support/`.
- **Forbidden Files or Directories**: Source code.
- **Acceptance Criteria**:
  - README includes Linux target paths.
  - Deferred VS Code workspace, VSCodium, and Insiders support is documented or omitted without misleading claims.
- **Verification**: Documentation review.
- **Dependencies**: LNX-1.
- **Risk Level**: Low.
- **Approval Needed**: No.
- **Status**: Approved.

## Dependency Order
1. LNX-1
2. LNX-2
3. LNX-3
4. LNX-4

## Parallel-Safe Groups
- LNX-4 can run after LNX-1 in parallel with LNX-3 if source behavior is stable.
- LNX-1 and LNX-2 should not run in parallel because both may touch `pkg/config/paths.go`.

## Verification Matrix
- Path detection: `go test ./pkg/config`
- Writer behavior: `go test ./pkg/config`
- Manager/app integration: `go test ./pkg/app`
- Full suite: `make test`
- E2E: `make test-e2e`
- Coverage gate: `make coverage-check`

## Blocked or Approval-Required Work
- No dependency changes are approved or planned.
- Workspace-scoped VS Code support is deferred.
- VSCodium and VS Code Insiders support is deferred.
