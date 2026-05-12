# TUI and CLI Error Handling E2E Testing (Phase 2)

## Problem Statement
The current E2E test suite successfully validates the “Golden Path” for all providers and targets when running the CLI in headless `--apply` mode. However, the interactive Terminal User Interface (TUI) powered by Bubbletea, and the various CLI failure modes (invalid keys, missing permissions, flag combinations) remain untested at an integration level. This leaves the core user experience vulnerable to regressions.

## Goals
- Validate the behavior of the Bubbletea TUI component through automated integration testing.
- Validate CLI failure modes, ensuring appropriate error messages and exit codes are returned for malformed input or missing prerequisites.
- Incorporate `charmbracelet/x/exp/teatest` to safely orchestrate TUI component testing and assert against interactive golden files.

## Non-Goals
- Testing the actual remote provider APIs.
- Replacing unit tests for individual functions; this is strictly an integration/E2E layer.
- Expanding provider capabilities.

## Users or Actors
- Contributors running `make test-e2e` to verify no regressions in the UI or CLI parsing logic.

## User Journeys
1. **Interactive TUI Flow**: The test suite launches the TUI, simulates keyboard input (e.g., Space to toggle, Enter to confirm), and asserts that the resulting config generated matches expected golden files.
2. **CLI Error Handling**: The test suite launches the CLI with an invalid flag combination (`--dry-run` and `--apply` together) and asserts a non-zero exit code and the correct standard error output.

## Functional Requirements
- **FR-1**: The E2E suite must use `teatest.NewTestModel` to wrap the `usync` Bubbletea model and simulate user interaction.
- **FR-2**: The test suite must simulate standard input commands (`tea.KeyMsg`) to navigate through the interactive selection wizard.
- **FR-3**: TUI visual output (or final model state) must be validated using `teatest.RequireEqualOutput` or similar assertions.
- **FR-4**: Add a table-driven test matrix verifying CLI exit codes (e.g., `os.Exit(1)` or `os.Exit(2)`) and `stderr` content for specific failure modes.

## Acceptance Criteria
- A `TestTUI_InteractiveFlow` test simulates entering the TUI, selecting specific targets, and applying changes, resulting in a successful configuration matching a golden file.
- A `TestCLI_FailureModes` test verifies at least 3 distinct error conditions (e.g., mutually exclusive flags, completely invalid API key format, missing required flags).
- Running `make test-e2e` successfully passes both new suites without hanging or requiring manual input.

## Success Criteria
- TUI component interactions are covered by E2E tests, verifying that the interactive flow successfully delegates to the `app.Manager`.
- Test coverage for `cmd/usync` and `pkg/tui` increases significantly.

## Edge Cases
- TUI execution hanging indefinitely in CI if `teatest.WaitFinished` is not configured with timeouts.
- Inconsistent terminal sizes causing visual golden file mismatches across platforms.

## Data Sensitivity and Compliance Notes
- No real credentials should be used or logged; fake UUIDs and mocked tokens must be strictly maintained.

## Assumptions
- `charmbracelet/x/exp/teatest` is fully compatible with the specific version of Bubbletea currently used by the project.
- Tests run in environments capable of handling pseudo-terminals (PTYs), which `teatest` manages internally.

## Open Questions
- Do we want to assert the intermediate visual output frames of the TUI, or just the final state and config mutations?
  - *Deferred Decision*: We will start by asserting the final output and config mutations to avoid brittle visual tests; visual frame assertions can be added later if needed.

## Human Approval Status
- Pending human review and approval.
