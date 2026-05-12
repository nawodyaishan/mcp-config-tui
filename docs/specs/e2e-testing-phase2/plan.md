# End-to-End Testing (Phase 2) Implementation Plan

## Summary
Implement automated integration tests covering the Bubbletea Terminal User Interface (TUI) and specific CLI parsing failure modes. The architecture leverages `charmbracelet/x/exp/teatest` to mock PTY interactions with the TUI and extends the existing `tests/e2e` suite to capture and assert `stderr` and non-zero exit codes for invalid executions.

## Inputs Reviewed
- `tests/e2e/e2e_test.go`
- `docs/specs/e2e-testing-phase2/spec.md`
- Exa research on Go TUI testing (`teatest`, Bubbletea testing patterns).
- `go.mod` (verified `teatest` is not yet installed).

## Assumptions
- Adding the `teatest` dependency will not introduce version conflicts with the existing `charmbracelet/bubbletea v1.3.10` or `charmbracelet/huh` ecosystem.

## Architecture Approach
1. **CLI Failure Modes**:
   - Create `TestCLI_FailureModes` in `tests/e2e/e2e_test.go`.
   - Modify `runBinary` (or create `runBinaryWithError`) to return both `stdout` and `stderr` combined, along with the error interface to assert `exit status X`.
   - Implement table-driven subtests covering: Mutual exclusivity (`--apply` + `--dry-run`), missing API keys (`no keys found`), and malformed provider keys.
2. **TUI Component Testing**:
   - Add the `github.com/charmbracelet/x/exp/teatest` dependency.
   - Create `tests/e2e/tui_test.go`.
   - Initialize the internal `tui.NewModel` wrapped in `teatest.NewTestModel`.
   - Send `tea.KeyMsg` sequences: e.g., Space (to deselect a target), Enter (to proceed to assignments), Enter (to confirm the plan).
   - Use `teatest.WaitFinished` to ensure the program concludes execution successfully.
   - Extract the generated configuration payload and assert it against a `.golden` file to prove that the TUI logic correctly passed the intended selection matrix to the `app.Manager`.

## Affected Modules
- `go.mod` / `go.sum`: Will require adding `github.com/charmbracelet/x/exp/teatest`.
- `tests/e2e/e2e_test.go`: Added CLI failure cases.
- `tests/e2e/tui_test.go`: New file to encapsulate TUI interactions.
- `pkg/tui/model.go`: (Read-only) Inspected to trace the message passing logic.

## Dependency Changes
- **Proposed Addition**: `github.com/charmbracelet/x/exp/teatest`
  - **Alternative**: Writing our own PTY testing wrapper using `github.com/creack/pty` or capturing `tea.Program` output directly using `bytes.Buffer`.
  - **Justification**: `teatest` is the official and actively maintained package by the Charmbracelet team specifically designed to test Bubbletea architectures deterministically.
  - **Approval Status**: Requires Human Approval due to adding a new external testing dependency.

## Testing Strategy
- **Golden Files**: The TUI output state (or the subsequent files it writes to disk) will be validated using golden files exactly like Phase 1, using `-update`.
- **Exit Code Checking**: The CLI test will explicitly verify that expected errors yield a status code `1` or `2` depending on whether it's a parsing error or a runtime error.

## Risks and Mitigations
- **Risk**: TUI testing is inherently asynchronous and can lead to flaky tests or hanging CI runs if state transitions lag.
  **Mitigation**: Strict use of `teatest.WithFinalTimeout()` and ensuring input is only sent after the model is fully initialized. 

## Human Architecture Approval Status
- Pending approval (specifically regarding the new dependency addition).
