# Architecture: Future Scalability & Plugin Research

## Objective
Research and define the long-term architectural roadmap for the Universal MCP Sync tool to ensure it scales elegantly to support hundreds of MCP servers and community-driven extensions.

## Research Findings (May 2026)

### 1. MCP Registry & Dynamic Discovery
Our research into the official Model Context Protocol ecosystem reveals a mature, centralized registry: `https://registry.modelcontextprotocol.io`.
- **API Capabilities**: The registry provides a robust REST API (`GET /v0.1/servers`) that supports cursor-based pagination, incremental syncing (`updated_since`), and substring searching.
- **Aggregator Pattern**: The ecosystem supports "Subregistries" or "Aggregators" that scrape the official registry and append custom metadata (like ratings, install instructions, or security scores) to the `_meta` field.

### 2. Go Plugin Architecture (HashiCorp `go-plugin`)
To scale beyond hardcoded Go structs for each `MCPProvider` (as implemented in Phase 1), we researched the industry standard for Go extensibility: `hashicorp/go-plugin`.
- **RPC over Localhost**: Instead of dynamic library loading (`plugin` stdlib), HashiCorp uses an RPC/gRPC model where each plugin runs as a separate, isolated subprocess.
- **Benefits for MCP Sync**:
  - **Isolation**: A poorly written community provider plugin crashing won't crash the main CLI wizard.
  - **Polyglot**: Community members could write MCP Sync providers in Python or TypeScript, not just Go.
  - **Security**: Plugins execute with restricted permissions compared to the host process.

## Proposed Future Architecture (Phase 3+)

To scale the Universal MCP Sync tool, we propose shifting from a statically compiled registry to a hybrid **Dynamic Registry + RPC Plugin** model.

### 1. The Dynamic Provider Engine
Instead of manually defining `NewGitHubProvider()` or `NewNotionProvider()`, the core engine will act as a Subregistry Aggregator.

**Workflow:**
1. **Fetch**: On startup, the CLI fetches a cached list of popular MCP servers from the official registry (`registry.modelcontextprotocol.io`).
2. **Generic Translation**: For 80% of standard MCP servers, the CLI uses a **Generic Provider**. The Generic Provider reads the JSON schema defined in the official registry and dynamically generates the `huh` TUI forms for required environment variables (e.g., `GITHUB_PAT`).
3. **Execution**: It uses the generic `MCPConfig` structure established in Phase 1 to write the `npx` or `docker` command required to run the server.

### 2. The Custom RPC Plugin System
For the 20% of MCP servers that require complex, bespoke setup logic (e.g., triggering a local OAuth flow, scanning the local filesystem to auto-configure paths, or migrating legacy configs like our `ExaProvider`), we will implement `hashicorp/go-plugin`.

**Implementation:**
1. We will extract our `MCPProvider` interface (defined in Phase 1) into a standalone Go module: `github.com/nawodyaishan/mcp-sync-sdk`.
2. We will wrap this interface in a `plugin.Plugin` gRPC server.
3. The core CLI will scan a `~/.config/mcp-sync/plugins/` directory on startup. If it finds a binary (e.g., `provider-custom-auth`), it spins it up via RPC and adds it to the TUI wizard alongside the dynamically fetched generic providers.

## Summary
By combining the **Official MCP Registry REST API** for infinite scale with **HashiCorp's `go-plugin`** for infinite depth, the Universal MCP Sync tool will become the definitive, future-proof platform engineering utility for AI-native developers.
