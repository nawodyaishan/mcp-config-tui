# End-to-End Testing Implementation Plan

## Summary
Implement an `os/exec`-based integration testing pipeline utilizing golden files for `usync`. The architecture will build the CLI binary once via `TestMain` and run prioritized scenarios covering all registered providers and target AI clients.

## Inputs Reviewed
- `pkg/app/qa_scenarios_test.go`
- `docs/specs/e2e-testing/spec.md`
- Advanced Go Testing Mechanisms research via Exa (`TestMain` hijack, `os/exec`, Golden files).

## Architecture Approach
1. **TestMain Orchestration**: In `cmd/usync/main_test.go`, implement `TestMain(m *testing.M)`. This will build the `usync` binary into a temporary directory before running the test suite, allowing all `Test*` functions to call `exec.Command(binaryPath)`.
2. **Test Helper Functions**:
   - `runBinary(args []string, homeDir string)`: Helper to run the CLI, injecting the required `HOME` environment variable.
   - `assertGolden(t *testing.T, actual []byte, goldenFile string)`: Helper to compare the actual file output with the golden file, automatically updating if the `-update` flag is passed.
3. **Table-Driven E2E Scenarios**: Create a comprehensive slice of `struct` detailing test scenarios. Each scenario specifies:
   - Target Provider (e.g., Exa, GitHub)
   - Simulated `HOME` structure (what files exist before the test)
   - CLI flags (e.g., `--apply`, `--keys-file`)
   - Expected Output Golden Files.

## Affected Modules
- `cmd/usync/main_test.go`: Added to orchestrate integration tests.
- `tests/e2e/e2e_test.go`: New file to hold the integration scenarios.
- `Makefile`: Addition of `test-e2e` target.

## Testing Strategy & Prioritized Cases
- **Idempotency Check**: Run the CLI twice; verify the final golden file state is identical and no duplicate entries are appended.
- **Provider Matrix Test**: A single test case that applies all 7 providers simultaneously and validates the output against golden files for the 12 target AI clients.
- **Merge Logic Test**: Seed the temporary `$HOME` with an existing `claude_desktop_config.json` containing manual `mcpServers`. Run `usync` and verify the golden file preserves the manual servers.

## Risks and Mitigations
- **Risk**: Golden files get out of sync due to absolute paths in outputs.
  **Mitigation**: The test helpers must scrub or sanitize absolute paths (e.g., replacing the temporary `$HOME` path with a placeholder like `{{HOME}}` before comparing to the golden file).
- **Risk**: Test suite is slow due to binary recompilation.
  **Mitigation**: `TestMain` ensures the binary is compiled exactly once for the entire package.

## Human Architecture Approval Status
- Pending approval.
