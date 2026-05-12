# End-to-End Testing Implementation Tasks

## Track Summary
Implement the `os/exec`-based integration testing pipeline utilizing golden files for `usync`. The architecture will build the CLI binary once via `TestMain` and run prioritized scenarios covering all registered providers and target AI clients.

## Prerequisites
- Approved `spec.md` and `plan.md` in `docs/specs/e2e-testing/`.

## Task List

### Task 1: Setup TestMain and Makefile
- **Objective**: Setup the E2E testing environment so the `usync` binary is compiled exactly once before tests are run.
- **Source Artifacts**: `docs/specs/e2e-testing/plan.md`
- **Allowed Files**: `cmd/usync/main_test.go`, `Makefile`
- **Forbidden Files**: Core source files in `pkg/`.
- **Acceptance Criteria**:
  - `Makefile` has a `test-e2e` target.
  - `cmd/usync/main_test.go` contains a `TestMain` that builds the `usync` binary and cleans it up after tests.
- **Verification Command**: `make test-e2e` runs and compiles the binary.
- **Dependencies**: None
- **Risk Level**: Low
- **Status**: Pending

### Task 2: Implement Test Helpers
- **Objective**: Create the core utility functions for running the built binary and asserting against golden files.
- **Source Artifacts**: `docs/specs/e2e-testing/plan.md`
- **Allowed Files**: `tests/e2e/e2e_test.go` (new)
- **Forbidden Files**: Core source files in `pkg/`.
- **Acceptance Criteria**:
  - Implement `runBinary(args []string, homeDir string)` wrapper.
  - Implement `assertGolden(t *testing.T, actual []byte, goldenFile string)` that handles the `-update` flag.
  - Implement path scrubbing to remove absolute `t.TempDir()` paths from the actual output before golden file comparison.
- **Verification Command**: `go test ./tests/e2e -v` (with basic dummy test).
- **Dependencies**: Task 1
- **Risk Level**: Low
- **Status**: Pending

### Task 3: Implement Provider Base Case Tests
- **Objective**: Write table-driven tests asserting the correct configuration generation for each Provider (Exa, GitHub, Context7, Tavily, Playwright, Kubernetes, Terraform).
- **Source Artifacts**: `docs/specs/e2e-testing/spec.md`
- **Allowed Files**: `tests/e2e/e2e_test.go`, `tests/e2e/testdata/*.golden`
- **Forbidden Files**: Core source files in `pkg/`.
- **Acceptance Criteria**:
  - Tests simulate the injection of keys (e.g. `EXA_API_KEY`, Github PAT) and standard arguments.
  - Golden files are generated and match expected schemas for standard stdio and http transports.
- **Verification Command**: `make test-e2e -update` and `make test-e2e`.
- **Dependencies**: Task 2
- **Risk Level**: Medium
- **Status**: Pending

### Task 4: Implement Target AI Client Matrix Tests
- **Objective**: Write tests to assert output schemas across all 12 target clients.
- **Source Artifacts**: `docs/specs/e2e-testing/spec.md`
- **Allowed Files**: `tests/e2e/e2e_test.go`, `tests/e2e/testdata/*.golden`
- **Forbidden Files**: Core source files in `pkg/`.
- **Acceptance Criteria**:
  - Asserts correct formatting for: Claude Desktop (`mcpServers`), VSCode (`mcpServers`), Codex CLI (TOML), Windsurf/Cursor (`httpUrl` mapping), RooCode/OpenCode/Kiro (nested JSON logic).
- **Verification Command**: `make test-e2e`.
- **Dependencies**: Task 3
- **Risk Level**: Medium
- **Status**: Pending

### Task 5: Implement Edge Case Tests (Idempotency and Merging)
- **Objective**: Validate the robustness of the config update logic against existing files.
- **Source Artifacts**: `docs/specs/e2e-testing/spec.md`
- **Allowed Files**: `tests/e2e/e2e_test.go`, `tests/e2e/testdata/*.golden`
- **Forbidden Files**: Core source files in `pkg/`.
- **Acceptance Criteria**:
  - Verify running `usync` twice on the same `HOME` temp dir doesn't duplicate entries (Idempotency).
  - Seed an existing `claude_desktop_config.json` with manual configs and ensure `usync` preserves them when appending the new provider.
- **Verification Command**: `make test-e2e`.
- **Dependencies**: Task 4
- **Risk Level**: Low
- **Status**: Pending

## Dependency Order
Task 1 -> Task 2 -> Task 3 -> Task 4 -> Task 5

## Parallel-Safe Groups
Tasks are primarily sequential as they build upon the test infrastructure, but the actual table-driven test cases within Tasks 3, 4, and 5 can be executed in parallel during the test run (`t.Parallel()`).
