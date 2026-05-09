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

### A. The Provider Interface (`pkg/provider`)
We decoupled the hardcoded Exa logic by introducing a standard `MCPProvider` interface. To support dynamic TUI generation, the provider exposes structured credential metadata.

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
    Env       map[string]string // Environment variables
}

type CredentialSpec struct {
    Key         string
    Label       string
    Description string
    Secret      bool
    MultiValue  bool
    Validator   func(string) error
}

type MCPProvider interface {
    ID() string
    Name() string
    Description() string
    
    // RequiredCredentials returns an ordered list of credential metadata
    RequiredCredentials() []CredentialSpec
    
    // Generates the final configuration block based on user inputs
    GenerateConfig(credentials map[string]string) (MCPConfig, error)
}
```

### B. The Explicit Provider Registry
An explicit, static registry holds all supported providers, making discovery easy to test and render without complex plugin loading overhead during early phases.

```go
// pkg/provider/registry.go
type Registry struct {
    providers map[string]MCPProvider
    order     []string
}

func DefaultRegistry() Registry {
    // Registers ExaProvider, GitHubProvider, etc.
}
```

### C. Provider-Aware Planning (`pkg/app`)
Operations no longer store raw secret material. We introduce `CredentialProfile` to separate user inputs from the generated config payload. `app.Manager` is updated to plan across providers.

```go
type CredentialProfile struct {
    ProviderID string
    Values     map[string]string
    Label      string
}

func (m *Manager) PrepareProvider(
    prov provider.MCPProvider,
    profiles []provider.CredentialProfile,
    selected map[config.AppID]bool,
    assignments map[config.AppID]int,
) (ExecutionPlan, error)
```

### D. TUI Wizard Evolution
The TUI Wizard (`pkg/tui`) will act as a Bubble Tea router with a 6-stage flow using `huh` for structured inputs:

1. **Provider Setup**: A searchable `huh.Select` displaying the list of servers from the Provider Registry.
2. **Credential Collection**: Dynamically generated fields (`huh.Text`, `huh.Password`) based on the selected provider's `RequiredCredentials()`. Exa multi-value fields fall back to text areas.
3. **Target Fleet Selection**: Selecting which AI clients (Claude, Gemini, etc.) should receive the server.
4. **Assignment**: Distributing generated credential profiles across the selected fleet.
5. **Preview**: Rendering the translated `stdio` or `http` config blocks with fully redacted secrets.
6. **Results**: Presenting atomic apply outputs and provider-aware verification.

---

## 4. Phased Implementation Strategy

### Phase 1: Core Interface & Exa Abstraction (Completed)
- [x] Define `MCPProvider`, `MCPConfig`, and `TransportType` in `pkg/provider`.
- [x] Abstract existing Exa logic into `ExaProvider`.
- [x] Overhaul `pkg/config` mutators to accept the generic `MCPConfig` struct instead of hardcoded strings.
- [x] Refactor core orchestration to generate configs via providers.
- [x] Verify total backward compatibility for existing Exa usage.

### Phase 2: Provider Registry and Dynamic TUI Wizard (Completed)
- [x] **Task 1: Registry & Credential Specs**: Implement the `Registry` and `CredentialSpec` types. Update `ExaProvider` to return ordered credential metadata.
- [x] **Task 2: Provider-Aware Planning**: Implement `PrepareProvider` using `CredentialProfile`. Remove raw `Operation.Key` storage in favor of `CredentialLabel`.
- [x] **Task 3: Dynamic Setup Form**: Update the `huh` wizard to include a Provider Selection step and dynamically generate credential input fields based on registry data.
- [x] **Task 4: Assignment & Preview Refactor**: Move assignment logic to operate on `CredentialProfile` slices. Update the preview screens to be provider-neutral.
- [x] **Task 5: Compatibility & Tests**: Ensure existing Exa `--keys` flags continue to work seamlessly via backward-compatibility wrappers. Ensure robust secret redaction.

### Phase 3: The `stdio` Engine & High-Value Providers
- **GitHub Provider**: Implement `NewGitHubProvider()`. This will be the flagship `stdio` provider, requiring a `GITHUB_PERSONAL_ACCESS_TOKEN` and generating an `npx -y @modelcontextprotocol/server-github` configuration.
- Update the JSON writing logic to properly format `command`, `args`, and `env` arrays for standard MCP clients.

### Phase 4: Capability Mapping & Safety Fallbacks
- **The Protocol Matrix**: Some clients (like Antigravity) currently only support remote HTTP endpoints, not local `stdio` processes.
- Update `pkg/app/app.go` to maintain a capability matrix. If a user attempts to install a `stdio`-only provider (like GitHub) into a client that only supports `http`, the manager must gracefully warn and skip that target rather than writing a broken configuration.

### Phase 5: Infinite Scale (Hybrid Dynamic Registry + RPC Plugins)
*Based on the scalability research, this phase transitions the tool from a statically compiled registry to a dynamically extensible engine.*
- **Dynamic Provider Engine**: Act as a Subregistry Aggregator. Fetch schemas directly from `registry.modelcontextprotocol.io` on startup and use a "Generic Provider" to dynamically generate `huh` forms for standard MCP servers without requiring Go code changes.
- **Custom RPC Plugin System**: Implement `hashicorp/go-plugin`. For complex servers requiring local OAuth flows or bespoke setup logic, developers can write isolated provider plugins (in Go, Python, etc.) that the CLI wizard communicates with over local RPC.

---

## 5. Naming & Branding
To reflect the shift from a niche Exa utility to a generalized platform engineering tool, the binary and repository will be rebranded to **Universal MCP Sync** (`mcp-sync` or `umcp`).
