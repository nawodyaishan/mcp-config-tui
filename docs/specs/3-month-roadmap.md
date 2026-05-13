# 3-Month Roadmap: Architectural Superiority & Scale

## 1. Summary
This roadmap outlines the strategic direction for Universal MCP Sync (`usync`) over the next 3 months. Based on an analysis of competitor tools (like `mcpup`), which rely on static registries and hardcoded client adaptations, `usync` will pivot aggressively towards **Architectural Superiority**. The focus will be on building a dynamic, infinitely scalable engine using official registries, advanced remote transports, and a secure plugin architecture.

## 2. Inputs Reviewed & Competitor Analysis
*   **Competitor Reviewed:** `mcpup` (mohammedsamin/mcpup)
*   **Competitor Strengths:** Supports 13 AI clients, 97 static built-in templates, interactive CLI wizard, profiles (workspaces), diagnostics (`doctor`), and rollback.
*   **Our Strategic Advantage:** `mcpup` relies heavily on static JSON maintenance. `usync` will leverage a **Dynamic Registry Engine** and **HashiCorp go-plugin** architecture to automate discovery and handle complex, secure authentication flows that a static JSON tool cannot safely execute.

---

## 3. Timeline & Execution Plan with Granular Mechanisms

### Month 1: Foundation for Scale & Remote Transports (Aligns with Phase 3)
*Objective: Escape the static binary limitation, fully support remote MCP communication, and finalize structural decoupling.*

**Mechanism 1.1: Structural Cleanup & Capabilities Matrix (Current Phase 3A)**
*   **Decouple App from Exa:** Remove domain-specific knowledge from `pkg/app` (e.g., `exa.RedactKey`).
*   **Transport Additions:** Add `TransportStreamableHTTP` and `PackageRuntime` to provider types.
*   **Client Capabilities Matrix:** Introduce `pkg/client/capabilities.go` to explicitly define which clients support `stdio`, `streamable-http`, `sse`, or `http` natively, and define fallback bridges (e.g., using `mcp-remote` for Claude Desktop).

**Mechanism 1.2: Standard Providers (Current Phase 3B)**
*   **GitHub Provider Integration:** Implement the official MCP GitHub server as the first native `stdio` provider to validate the `pkg/client` capabilities matrix and bridge abstractions.

**Mechanism 1.3: Dynamic Registry Engine MVP**
*   **Registry Fetcher:** Implement `pkg/registry` to fetch and cache schemas from `https://registry.modelcontextprotocol.io`.
*   **Schema Translator:** Create an adapter that translates the official MCP registry JSON schema into internal `provider.MCPConfig` structs dynamically, bypassing the hardcoded catalog approach.
*   **Interactive TUI Browse:** Integrate a Bubbletea list component in the setup wizard to search, filter, and select tools directly from the dynamic registry.

### Month 2: Security & Extensibility via HashiCorp Plugins (Phase 4)
*Objective: Enable secure, custom provider authentication flows without bloating the core CLI.*

**Mechanism 2.1: gRPC Plugin Scaffolding (`hashicorp/go-plugin`)**
*   **Interface Definition:** Export the `MCPProvider` interface via protobuf to allow separate binaries (written in Go, Python, or Rust) to implement custom providers.
*   **Plugin Host Engine:** Modify `pkg/provider/registry.go` to scan a local `~/.usync/plugins/` directory, spinning up gRPC subprocesses for discovered plugins automatically.
*   **State Management:** Ensure the core `usync` engine securely passes credentials to the plugin subprocesses via secure channels and safely retrieves the generated `MCPConfig` without logging secrets.

**Mechanism 2.2: Advanced Auth Flows**
*   **OAuth2 / OIDC Support:** Build a generic OAuth2 plugin template to handle browser-based authentication flows for enterprise MCP servers.
*   **Local Secret Management:** Integrate with OS keychains (e.g., macOS Keychain) to store refresh tokens and API keys securely, allowing `usync` to re-authenticate plugins automatically.

**Mechanism 2.3: `usync doctor` & Diagnostics**
*   **Drift Detection:** Compare the active state of client config files (e.g., `.cursor/mcp.json`) against the canonical `usync` state to detect manual edits.
*   **Environment Validation:** Check the user's `PATH` for required CLI tools (e.g., `npx`, `uvx`, `docker`) before attempting to write stdio configurations that depend on them.

### Month 3: Usability, Profiles, and Polish (Phase 5)
*Objective: Deliver a superior UX using our Bubbletea/Huh foundation, rivaling and exceeding competitor tooling.*

**Mechanism 3.1: Workspaces & Profiles**
*   **Profile Data Model:** Introduce `~/.usync/profiles.json` to group specific servers and target clients into distinct workspaces (e.g., "Frontend Dev", "DevOps").
*   **Context Switching:** Add the `usync profile apply <name>` command to batch-enable or disable servers across selected clients atomically.

**Mechanism 3.2: Robust Rollback Engine**
*   **Snapshotting Ledger:** Enhance `pkg/app` to maintain a SQLite or robust JSON ledger mapping timestamped sync operations to their respective file backup paths.
*   **Atomic Revert:** Implement `usync rollback --client <id>` to instantly restore the previous known-good config state if a client fails to parse a written file.

**Mechanism 3.3: Expanded Client Support**
*   **Coverage Expansion:** Incrementally update `pkg/client/capabilities.go` and `pkg/config/*` writers to support the remaining 8 clients (reaching 13 total to match `mcpup`, including Continue, Roo Code, Amazon Q, etc.).

---

## 4. SDD & Architecture Impact
*   **Affected Modules:** `pkg/provider` (complete overhaul to plugin/dynamic architecture), `pkg/tui` (wizard and profiles addition), `pkg/app` (reconciliation logic).
*   **Data Model Changes:** Introduction of `Profile` schemas and a more robust internal state ledger for rollbacks.
*   **Security Impact:** Utilizing `hashicorp/go-plugin` creates a secure subprocess boundary. However, handling dynamic schemas and complex auth requires rigorous input validation and secure credential handoff logic.
*   **Testing Strategy:** Heavy emphasis on E2E scenario testing (`pkg/app/qa_scenarios_test.go`) across the expanded client matrix, mocking the official MCP registry, and testing gRPC plugin boundaries.
