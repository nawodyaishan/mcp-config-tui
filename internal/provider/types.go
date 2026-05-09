package provider

type TransportType string

const (
	TransportHTTP  TransportType = "http"
	TransportStdio TransportType = "stdio"
	TransportSSE   TransportType = "sse"
)

// MCPConfig represents a generalized, provider-agnostic MCP server configuration.
type MCPConfig struct {
	Type    TransportType
	URL     string            // For HTTP/SSE
	Command string            // For stdio
	Args    []string          // For stdio
	Env     map[string]string // Environment variables required by the server
}

// MCPProvider defines the contract for any MCP server we support.
type MCPProvider interface {
	ID() string
	Name() string
	Description() string
	// RequiredCredentials returns a map of environment variable keys to user-facing prompts.
	RequiredCredentials() map[string]string
	// GenerateConfig builds the final MCPConfig based on the user's provided credentials.
	GenerateConfig(credentials map[string]string) (MCPConfig, error)
}
