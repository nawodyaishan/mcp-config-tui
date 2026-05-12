# End-to-End Testing (Phase 2) Implementation Tasks

## Track Summary
Implement automated integration tests covering the Bubbletea Terminal User Interface (TUI) and specific CLI parsing failure modes, leveraging `teatest` for PTY mocking and extending the `os/exec` patterns from Phase 1.

## Prerequisites
- Approved `spec.md` and `plan.md` in `docs/specs/e2e-testing-phase2/`.

## Task List

### Task 1: Add Dependency and Refactor Test Helpers
- **Objective**: Add `github.com/charmbracelet/x/exp/teatest` and enhance `runBinary` to capture error states.
- **Source Artifacts**: `docs/specs/e2e-testing-phase2/plan.md`
- **Allowed Files**: `go.mod`, `go.sum`, `tests/e2e/e2e_test.go`
- **Forbidden Files**: Core source files in `pkg/`.
- **Acceptance Criteria**:
  - Run `go get github.com/charmbracelet/x/exp/teatest@latest`.
  - Introduce `runBinaryWithError(t *testing.T, args []string, homeDir string) ([]byte, error)` to safely expose both the combined output and the `error` from `exec.Command`.
- **Verification Command**: `go mod tidy` and `make test-e2e`.
- **Dependencies**: None
- **Risk Level**: Low
- **Status**: Pending

### Task 2: Implement CLI Failure Modes Tests
- **Objective**: Validate the CLI gracefully rejects invalid arguments or missing prerequisites.
- **Source Artifacts**: `docs/specs/e2e-testing-phase2/spec.md`
- **Allowed Files**: `tests/e2e/e2e_test.go`
- **Forbidden Files**: Core source files in `pkg/`.
- **Acceptance Criteria**:
  - Implement `TestCLI_FailureModes(t *testing.T)`.
  - Validate `--apply` and `--dry-run` together yields exit code 2 and standard error message.
  - Validate missing `--keys` flag (when no interactive TUI is desired but apply is passed, or when TUI is not triggered) yields appropriate error exit.
  - Assert the actual `exit status` from the `exec.ExitError` returned by `runBinaryWithError`.
- **Verification Command**: `make test-e2e`.
- **Dependencies**: Task 1
- **Risk Level**: Low
- **Status**: Pending

### Task 3: Implement TUI Interaction Test
- **Objective**: Wrap the TUI model with `teatest` to simulate user input and assert golden output.
- **Source Artifacts**: `docs/specs/e2e-testing-phase2/spec.md`
- **Allowed Files**: `tests/e2e/tui_test.go` (new), `tests/e2e/testdata/*.golden`
- **Forbidden Files**: Core source files in `pkg/`.
- **Acceptance Criteria**:
  - Create `tui_test.go` using `teatest.NewTestModel()`.
  - Construct the `tui.NewModel` utilizing `app.Manager` and dummy keys.
  - Send `tea.KeyMsg` sequences (e.g. `tea.KeyEnter`) using `tm.Send()`.
  - Call `tm.WaitFinished()` with a timeout.
  - Assert the resulting terminal output against a new `.golden` file using `teatest.RequireEqualOutput(t, out)`.
- **Verification Command**: `make test-e2e -update` followed by `make test-e2e`.
- **Dependencies**: Task 1
- **Risk Level**: Medium
- **Status**: Pending

## Dependency Order
Task 1 -> Task 2 -> Task 3

## Parallel-Safe Groups
Tasks 2 and 3 can be executed independently once Task 1 is complete.
