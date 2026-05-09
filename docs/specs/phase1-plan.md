# Phase 1 Implementation Plan: Core Interface & Exa Abstraction

## Objective
Establish the foundational Provider/Registry architecture by decoupling the `exa-mcp-manager` from its hardcoded Exa implementation. This phase will introduce the generic `provider` package and migrate the existing Exa logic into an `ExaProvider`, ensuring all current functionality remains intact before we scale to new providers.

## Task Breakdown

### Task 1: Establish `internal/provider` Package
1. **Create Directory**: Create the `internal/provider` directory.
2. **Define Types (`types.go`)**:
   - Define `TransportType` (`http`, `stdio`, `sse`).
   - Define `MCPConfig` struct to hold `Type`, `URL`, `Command`, `Args`, and `Env`.
   - Define `MCPProvider` interface with methods: `ID()`, `Name()`, `Description()`, `RequiredCredentials()`, and `GenerateConfig(credentials map[string]string)`.
3. **Implement Exa Provider (`exa.go`)**:
   - Create `ExaProvider` struct implementing `MCPProvider`.
   - Implement `GenerateConfig` to use `exa.BuildURL` and return an `MCPConfig` with `Type: TransportHTTP` and the generated URL.

### Task 2: Refactor Configuration Mutators (`internal/config`)
1. **JSON Mutators (`json_update.go`)**:
   - Create a helper function `buildConfigMap(cfg provider.MCPConfig, urlFieldName string) map[string]any` to dynamically build the JSON object based on the transport type (e.g., `command`/`args`/`env` for `stdio`, vs `url`/`httpUrl` for `http`).
   - Update `UpdateMCPServersJSON`, `UpdateBareMCPServersJSON`, and `UpdateNamedServerJSON` to accept `providerID string` and `cfg provider.MCPConfig` instead of just an `exaURL string`.
2. **TOML Mutators (`toml_update.go`)**:
   - Update `UpdateCodexTOML` to accept `providerID` and `cfg provider.MCPConfig`. (For Phase 1, only handle HTTP URL insertion to maintain parity).
3. **Update Tests (`json_update_test.go`, `toml_update_test.go`)**:
   - Fix all test cases to pass dummy `MCPConfig` objects and provider IDs instead of raw strings.

### Task 3: Refactor Orchestration (`internal/app`)
1. **Update `Operation` Struct (`app.go`)**:
   - Replace the `URL string` field with `ProviderID string` and `Config provider.MCPConfig`.
2. **Update CLI Generation Logic**:
   - In `Prepare`, specifically for `AppClaudeCode`, update `CLIAddArgs` logic to dynamically use `--transport` based on `Config.Type`. 
3. **Update `Prepare` Method**:
   - Instantiate `ExaProvider`.
   - Iterate through assigned keys, creating a `credentials` map (`{"EXA_API_KEY": key}`).
   - Call `GenerateConfig` and store the resulting `MCPConfig` in the `Operation`.
4. **Update `prepareFileOperation`**:
   - Pass `op.ProviderID` and `op.Config` to the configuration mutators instead of `op.URL`.
5. **Update Tests (`app_test.go`)**:
   - Fix all test cases to initialize dummy `MCPConfig` structs.

### Task 4: TUI & Verification
1. **Update TUI (Minor)**:
   - Ensure that any references in `internal/tui` to the old `app.Operation.URL` (if any, like in previews) are updated to print the `Config.URL` or a summarized config block.
2. **Run Verification**:
   - `make test`: Ensure 100% test pass rate.
   - `golangci-lint run ./...`: Ensure no new linting errors.
   - `make build`: Verify successful compilation.
   - Manually run the wizard (dry-run) to ensure the generated configuration output exactly matches the pre-refactor output for Exa.

## Rollback Plan
If tests fail or regressions are found in the generated config structure, revert the commits applying this phase. The boundary of this phase is strictly internal; no user-facing changes should occur.
