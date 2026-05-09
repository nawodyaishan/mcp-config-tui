# QA & Usability Strategy: High-Fidelity Validation

## 1. Objective
Ensure the `Universal MCP Sync Engine` is robust, compatible with official documentation, and provides a delightful, accessible user experience for developers.

## 2. Broadened Test Coverage (QA)

### A. "Golden Path" Validation (Official README Scenarios)
- **Claude Desktop**: Verify `stdio` bridge (`npx -y mcp-remote`) correctly wraps the remote URL.
- **Gemini CLI**: Verify `httpUrl` field usage in `~/.gemini/settings.json`.
- **Windsurf/Antigravity**: Verify `serverUrl` field usage.
- **Codex CLI**: Verify TOML `url = "..."` format.
- **Cursor/Zed**: Verify standard `url` field usage.

### B. Error Handling & Edge Cases
- **Invalid Inputs**: Test with malformed UUIDs, non-string values, and empty strings.
- **File System Obstacles**:
    - Target file is a directory.
    - Target file is read-only (0400).
    - Missing parent directories (ensure `MkdirAll` is robust).
- **Dependency Missing**: Simulate `claude` CLI or `npx` not being in `$PATH`.

### C. Idempotency & Stability
- **Repeated Runs**: Running the sync tool multiple times with the same key should NOT change the file content.
- **Config Merge Logic**: Ensure that adding a new MCP server doesn't delete existing, manually added servers in the target config.

## 3. Product Usability (UX)

### A. Interface Design (TUI)
- **Progressive Disclosure**: Only show credential fields relevant to the selected provider.
- **Visual Feedback**: Use `lipgloss` to create visual separation between steps and clearer status indicators.

### B. Error Messaging & Remediation
- **Context-Aware Errors**: Suggest specific fixes (e.g., "Check file permissions (0600 expected)").
- **Redaction Safety**: Confirm no full API keys or secret URLs are leaked to logs or the TUI even in error messages.

## 4. Implementation Strategy
- **Go Test Suite**: Implement `pkg/app/qa_scenarios_test.go`.
- **Regression**: Add `make verify-scenarios` to the `Makefile`.

## 5. Verification
- All tests pass: `go test ./...`
- Manual UX audit of the TUI.
