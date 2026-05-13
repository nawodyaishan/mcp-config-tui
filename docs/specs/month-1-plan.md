# Month 1 Execution Plan: Foundation for Scale & Remote Transports

## 1. Overview
Month 1 focuses on escaping the "static binary" limitations of the early architecture. We will establish a robust capabilities matrix for AI clients, introduce full support for remote and standard `stdio` transports, and build the MVP for our **Dynamic MCP Registry Engine**. This engine will heavily utilize the official Model Context Protocol API (`registry.modelcontextprotocol.io`) to discover and configure servers dynamically.

---

## 2. Phase 3A: Structural Cleanup & Client Capabilities

### Objective
Decouple the core application (`pkg/app`) from specific provider logic (like `pkg/exa`) and formalize how `usync` understands the transport capabilities of different AI clients.

### Breakdown
1. **Transport Enum Expansion (`pkg/provider/types.go`)**
   - Add `TransportStreamableHTTP` and `TransportSSE` constants.
   - Introduce `PackageRuntime` to define how a `stdio` server is executed (e.g., `npx`, `uvx`, `docker`).
2. **Generic Secret Redaction (`pkg/redact`)**
   - Move UUID redaction logic out of `pkg/exa` into a generic `pkg/redact` package.
   - Update `app.go` logging to use this generic redactor, completely severing the `pkg/app` -> `pkg/exa` import dependency.
3. **Capabilities Matrix (`pkg/client/capabilities.go`)**
   - Define a `TransportSupport` struct mapping what transports a client supports natively (e.g., Claude Desktop supports `stdio`, Gemini CLI supports `streamable-http`).
   - Define a `BridgeConfig` for clients requiring fallback mechanisms (e.g., configuring `mcp-remote` bridging for Claude Desktop to connect to HTTP servers).
4. **Adapter Engine (`pkg/client/adapter.go`)**
   - Implement `Adapt(appID, cfg)` which returns an unmodified config if supported natively, or applies the appropriate `BridgeConfig` if fallback is required.

---

## 3. Phase 3B: Standard Native Providers (`stdio`)

### Objective
Validate the new capabilities matrix and `stdio` configurations by implementing a high-value, standard MCP server natively.

### Breakdown
1. **GitHub Provider (`pkg/provider/github.go`)**
   - Implement the `MCPProvider` interface for `@modelcontextprotocol/server-github`.
   - Configure it to return `TransportStdio` with the command `npx` and required environment variable `GITHUB_PERSONAL_ACCESS_TOKEN`.
2. **Provider Verification Updates (`pkg/verify/verify.go`)**
   - Remove hardcoded "Exa" verifications.
   - Implement `verifyGenericStdioServer` (validating command structures) and `verifyGenericHTTPServer` (validating URL parsing).
3. **E2E Validation**
   - Update `qa_scenarios_test.go` to ensure `usync` successfully skips writing the GitHub provider to clients that *only* support remote HTTP (like Gemini CLI), while successfully installing it to `stdio`-capable clients (like Cursor or VS Code).

---

## 4. Phase 3C: Dynamic MCP Registry Engine MVP

### Objective
Transition `usync` from a hardcoded set of provider templates to dynamically fetching configurations directly from the official MCP community registry. This heavily leverages MCP architecture standards.

### Breakdown
1. **Registry Client (`pkg/registry/client.go`)**
   - Implement an HTTP client targeting `https://registry.modelcontextprotocol.io/v0.1/servers`.
   - Implement paginated fetching using the `metadata.nextCursor` token.
2. **Schema Parsing (`pkg/registry/schema.go`)**
   - Parse the incoming `ServerJSON` format conforming to `https://static.modelcontextprotocol.io/schemas/2025-12-11/server.schema.json`.
   - Extract critical `Package` array data:
     - `registryType` (e.g., `npm`, `pypi`, `docker`)
     - `identifier` (e.g., `@modelcontextprotocol/server-postgres`)
     - `transport` configuration constraints.
3. **Dynamic Provider Adapter (`pkg/provider/dynamic.go`)**
   - Create a generic `DynamicProvider` struct that satisfies `MCPProvider`.
   - Automatically map `environmentVariables` required by the `ServerJSON` schema into `usync`'s internal `CredentialSpec` prompts.
   - Map `runtimeHint` (e.g., `npx`) into the `Command` field of our `MCPConfig`, and the `identifier` into the `Args`.
4. **TUI Registry Browser (`pkg/tui/registry_browser.go`)**
   - Add a new Bubbletea view allowing the user to search the fetched registry list.
   - Display the `icons`, `title`, and `description` sourced directly from the MCP registry.
   - On selection, dynamically instantiate a `DynamicProvider` and route the user to the credential entry form.

---

## 5. Month 1 Exit Criteria
*   **Architecture:** `pkg/app` has 0 dependencies on any specific provider package.
*   **Capabilities:** `usync` correctly routes `stdio` vs `streamable-http` configurations to the 12 core clients based on their matrix support.
*   **MCP Integration:** Users can fetch, configure, and install any `stdio` server listed on the official `registry.modelcontextprotocol.io` without requiring a CLI update.
*   **Testing:** Total Go statement coverage remains above 65%, enforced by CI gates.