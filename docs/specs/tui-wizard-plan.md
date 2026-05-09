# TUI Wizard Implementation & Exa Configuration Fix Plan

## Objective
Refactor the Exa MCP Config Manager's interactive TUI into a production-grade, multi-step flow wizard using `charmbracelet/huh` and `charmbracelet/bubbletea`. Additionally, resolve critical legacy configuration issues related to the Exa MCP server, aligning with the modern Streamable HTTP transport and optimized toolset.

## Part 1: Exa MCP Configuration Fixes (Pre-requisite / Integration)
Based on the latest Exa documentation and sync reports, the current codebase hardcodes deprecated SSE transports and outdated tool sets.

**Identified Issues & Solutions:**
1. **Tool Set Optimization**: 
   - **Issue**: `internal/exa/tools.go` includes deprecated tools (`get_code_context_exa`, `company_research_exa`, etc.).
   - **Solution**: Reduce `DefaultTools` to the core trio: `web_search_exa`, `web_search_advanced_exa`, and `web_fetch_exa`.
2. **Claude Code Transport Modernization**:
   - **Issue**: `internal/app/app.go` hardcodes `--transport sse` for Claude Code CLI ops.
   - **Solution**: Change `CLIAddArgs` to use `--transport http` according to modern remote execution standards.
3. **Claude Desktop Configuration**:
   - **Issue**: `UpdateMCPServersJSON` writes `"type": "sse"` and generic `"url"`.
   - **Solution**: Refactor to support the `npx mcp-remote` command wrapper structure or native HTTP integrations as required by Claude Desktop.
4. **Gemini CLI Configuration**:
   - **Issue**: Gemini CLI uses `"httpUrl"` instead of `"url"` and does not use `"type": "sse"`.
   - **Solution**: Introduce specific JSON update logic for Gemini that writes the `httpUrl` field correctly without legacy transport type markers.
5. **Antigravity Configuration**:
   - **Issue**: Requires `"serverUrl"`. Currently mostly correct, but the JSON generators must not force `"type": "sse"` generically.

## Part 2: TUI Wizard Architecture & Modularity Principles
To ensure high code quality, we will move away from a single, monolithic `Model` (the "God Object" anti-pattern) and adopt a modular **Router/Sub-Model architecture**.

1. **State Machine Router**: The main `tui.Model` will act strictly as a state machine and router. It will not contain form rendering logic. It will orchestrate transitions between sub-models (the `Huh` form, the Assignments screen, the Preview screen, etc.).
2. **Encapsulated Sub-Models**: Each major phase of the wizard will be its own encapsulated struct that implements `tea.Model` (or exposes `Init`, `Update`, `View` methods).
3. **Shared State Context**: A central `wizardContext` struct will hold the aggregated data (keys, selected apps, assignments). Sub-models will receive a pointer to this context to read/write data seamlessly.
4. **Validation at Source**: Data validation will occur immediately within the `Huh` fields before the state machine is allowed to progress.

## Phased Implementation Plan

### Phase 1: Context & Core Refactor
- Define `wizardContext` and refactor `Model` into the Router pattern.
- Move existing preview and results logic into `previewModel` and `resultsModel`.
- **Config Fixes**: Apply fixes to `internal/exa/tools.go` to remove deprecated tools. Overhaul `internal/config/json_update.go` and `internal/app/app.go` to remove legacy `sse` transport markers, using the proper keys (`httpUrl`, `serverUrl`, or `npx` commands) depending on the target app.

### Phase 2: Huh Integration
- Add the `huh` dependency (`go get github.com/charmbracelet/huh`).
- Implement `newSetupForm`.
- Integrate the form's `Init`, `Update`, and `View` into the Main Router.
- Wire the state transition: `if m.setupForm.State == huh.StateCompleted { m.stage = stageAssignments }`.

### Phase 3: Assignments Extraction
- Extract the dynamic assignment logic into `assignmentModel`.
- Wire up the transition from `setupForm` -> `assignments` -> `preview`.

### Phase 4: Testing & Polish
- Ensure the non-interactive CLI logic remains unchanged.
- Run `make test` to ensure `wizardContext` integration didn't break core flows. Fix test suites in `internal/verify` and `internal/config` that relied on old tool counts and JSON structures.
- Verify layout responsiveness and keyboard navigation across the boundaries of the `Huh` form and the custom sub-models.
