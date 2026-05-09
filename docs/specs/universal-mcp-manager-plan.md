# Architectural Plan: Universal MCP Configuration Manager

## 1. Product Vision & Problem Statement

### The Audience
- **AI-Native Developers:** Engineers building, testing, and debugging with multiple local AI assistants (Claude Desktop, Cursor, Zed, Gemini CLI).
- **Platform Engineers:** Internal tooling teams trying to standardize AI toolchains and MCP environments across their organization.
- **Power Users:** Enthusiasts who want to wire up their local files, GitHub repos, and web search to their AI agents without becoming JSON configuration experts.

### The Problem: The "N x M" Configuration Nightmare
The Model Context Protocol (MCP) is standardizing how AI agents communicate with tools, but **how humans configure those agents remains a fragmented mess.**
- Every AI client has a bespoke configuration file format, location, and syntax.
- Keeping MCP servers synchronized across Claude, Gemini, and Cursor leads to configuration drift.
- Finding the correct execution method for a popular server (e.g., `npx -y @modelcontextprotocol/server-github` vs a Streamable HTTP endpoint) requires digging through READMEs.

### The Solution
Transform the `exa-mcp-manager` into a **Universal MCP Sync Engine**. It will serve as the single source of truth for an engineer's local AI environment. A developer selects a capability (e.g., "GitHub Integration"), provides their token once via a guided TUI wizard, and the tool intelligently translates and distributes that configuration across all supported AI clients.

---

## 2. Research Findings: Target MCP Integrations
Based on community usage data (Smithery.ai, official MCP GitHub org), the architecture must scale to support highly-utilized server types:
- **Development & Version Control**: GitHub (official), GitLab
- **Web Search**: Exa, Brave Search
- **Productivity & Knowledge**: Notion, Sentry, Google Drive
- **Utility / Agentic**: Sequential Thinking, Memory/Filesystem

**Key Insight**: Unlike Exa (which primarily uses HTTP endpoints), many of these servers require local execution via `npx`, Docker, or standard `stdio` transports. Our engine must support both local processes and remote endpoints.

---

## 3. Core Architectural Shifts

To support multiple MCP types without creating a tangled monolith, we must transition to a **Provider/Registry Architecture**.

### A. The Provider Interface (`internal/provider`)
We will decouple the hardcoded Exa logic by introducing a standard `MCPProvider` interface. Every supported MCP server will implement this interface.

```go
type TransportType string
const (
    TransportHTTP  TransportType = "http"
    TransportStdio TransportType = "stdio"
    TransportSSE   TransportType = "sse"
)

type MCPConfig struct {
    Type      TransportType
    URL       string            // For HTTP/SSE
    Command   string            // For stdio (e.g., "npx")
    Args      []string          // For stdio
    Env       map[string]string // Environment variables (e.g., GITHUB_PERSONAL_ACCESS_TOKEN)
}

type MCPProvider interface {
    // Unique identifier (e.g., "github", "exa")
    ID() string
    
    // Display name for the TUI (e.g., "GitHub (Official)")
    Name() string
    
    // Explanation of what the tool does
    Description() string
    
    // Prompts required from the user
    // Returns a map of env keys to prompt descriptions (e.g., {"GITHUB_PAT": "Enter GitHub Personal Access Token"})
    RequiredCredentials() map[string]string
    
    // Generates the final configuration block based on user inputs
    GenerateConfig(credentials map[string]string) (MCPConfig, error)
}
```

### B. The Provider Registry
A central registry will hold all supported providers. Scaling support for a new MCP server simply requires creating a new struct implementing `MCPProvider` and registering it.

```go
// internal/provider/registry.go
var Registry = []MCPProvider{
    NewExaProvider(),
    NewGitHubProvider(),
    NewSequentialThinkingProvider(),
}
```

### C. Refactoring Configuration Mutators (`internal/config`)
Currently, `json_update.go` and `toml_update.go` hardcode the "exa" object key and expect a single URL. They must be refactored to accept the generic `MCPConfig` struct.

*Example target JSON structure generation for stdio:*
```json
"mcpServers": {
  "<Provider.ID()>": {
    "command": "<Config.Command>",
    "args": [<Config.Args>],
    "env": {
      "KEY": "VALUE"
    }
  }
}
```

### D. TUI Wizard Evolution
The TUI Wizard (`internal/tui`) will be updated to a robust, multi-step flow using `huh`:

1. **Provider Discovery**: A searchable `huh.Select` displaying the list of servers from the Provider Registry, categorized by capability.
2. **Credential Collection**: Dynamically generated `huh.Text` or `huh.Password` fields based on the selected provider's `RequiredCredentials()`.
3. **Target Fleet Selection**: Selecting which AI clients (Claude, Gemini, etc.) should receive the server.
4. **Intelligent Preview & Apply**: Rendering the translated `stdio` or `http` config blocks before committing atomic writes.

---

## 4. Phased Implementation Strategy

### Phase 1: Core Interface & Exa Abstraction
- Define `MCPProvider`, `MCPConfig`, and `TransportType` in a new `internal/provider` package.
- Abstract the existing Exa logic into `ExaProvider`.
- Overhaul `internal/config` mutators to accept the generic `MCPConfig` struct instead of hardcoded string URLs.
- *Validation*: Ensure the tool still works identically for Exa before introducing new providers.

### Phase 2: Dynamic TUI Wizard
- Update `internal/tui/setup_form.go`.
- Introduce a Provider Discovery screen.
- Replace the hardcoded Exa key input with a dynamic credential collector that queries the selected `MCPProvider`.

### Phase 3: The `stdio` Engine & High-Value Providers
- **GitHub Provider**: Implement `NewGitHubProvider()`. This will be the flagship `stdio` provider, requiring a `GITHUB_PERSONAL_ACCESS_TOKEN` and generating an `npx -y @modelcontextprotocol/server-github` configuration.
- Update the JSON writing logic to properly format `command`, `args`, and `env` arrays for standard MCP clients.

### Phase 4: Capability Mapping & Safety Fallbacks
- **The Protocol Matrix**: Some clients (like Antigravity) currently only support remote HTTP endpoints, not local `stdio` processes.
- Update `internal/app/app.go` to maintain a capability matrix. If a user attempts to install a `stdio`-only provider (like GitHub) into a client that only supports `http`, the manager must gracefully warn and skip that target rather than writing a broken configuration.

---

## 5. Naming & Branding
To reflect the shift from a niche Exa utility to a generalized platform engineering tool, the binary and repository will be rebranded to **Universal MCP Sync** (`mcp-sync` or `umcp`).
