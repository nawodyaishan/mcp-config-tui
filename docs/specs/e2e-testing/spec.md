# End-to-End Testing Prioritization and Strategy

## Problem Statement
The current E2E test coverage (`TestQAExaReadmeScenarios`) validates that `usync` modifies the configuration files of the target AI clients, but it uses mock `App` internals rather than invoking the real CLI binary. The tests do not rigorously validate all edge cases, missing file paths, idempotency, or specific behavior per provider and target client format. We need a comprehensive testing mechanism prioritizing critical paths for each provider and target combination.

## Goals
- Design a matrix of the most prioritized E2E test cases for each supported MCP provider and target AI client.
- Implement an `os/exec`-based integration testing approach for the Go CLI that invokes the real compiled binary.
- Use golden files (`.golden`) to strictly assert the output configuration file schemas across 12 different targets.
- Validate idempotency, missing directory scaffolding, and config merging logic.

## Non-Goals
- Testing the actual remote MCP services (e.g., calling the Exa API). The scope is bounded to configuration file generation and CLI orchestration.
- Modifying the underlying Bubbletea TUI behavior.

## User Journeys
1. **CLI Execution**: The system tests run `make test-e2e`, which builds the `usync` binary and executes it against a temporary `$HOME` directory.
2. **Target Assertion**: The test suite evaluates the generated target files against known-good golden files to catch regressions.

## Functional Requirements
- **FR-1**: The E2E test suite must compile the CLI binary once before running tests (via `TestMain` or Makefile targets).
- **FR-2**: The test framework must redirect `$HOME` to a `t.TempDir()` per test case.
- **FR-3**: Must implement a `-update` flag for `go test` to easily update golden files.
- **FR-4**: Must validate specific schemas for targets: `mcpServers` (Claude Desktop/VSCode), TOML (Codex), specific HTTP schemas (Cursor/Windsurf), and custom nested structures (RooCode/OpenCode).
- **FR-5**: Must validate the correct configuration generation for each Provider (Exa, GitHub, Context7, Tavily, Playwright, Kubernetes, Terraform) with their specific transport types (stdio vs http) and arguments.

## Prioritized Test Cases
### By Provider
1. **ExaProvider**: Validate default settings, verify `EXA_API_KEY` is correctly placed in `env` (stdio) or URL (http).
2. **GitHubProvider**: Verify the Github PAT is placed safely in `env`.
3. **Context7Provider**: Validate injection of `keys.json` logic into the HTTP URL format.
4. **Terraform/Kubernetes/Playwright**: Validate local binary pathing and `args` in stdio transports.

### By Target
1. **Claude Desktop**: Validate atomic writes and correct `mcpServers` format.
2. **VSCode/Zed**: Validate array/object append without erasing existing manual configs.
3. **Codex CLI**: Validate TOML translation and escaping.
4. **Windsurf/Cursor**: Validate HTTP URL mapping.
5. **RooCode/OpenCode/Kiro**: Validate accurate JSON schema nesting and boolean flags.

## Acceptance Criteria
- Running `make test-e2e` successfully passes using `os/exec` running the built CLI.
- 100% of the target AI clients have at least one E2E golden file test validating output.
- All existing providers are tested for their base case.

## Edge Cases
- Target configuration files already contain conflicting configs.
- The `keys_file.txt` is missing or malformed.
- The Target directory does not exist and needs to be scaffolded.

## Human Approval Status
- Needs review by maintainers before implementation.
