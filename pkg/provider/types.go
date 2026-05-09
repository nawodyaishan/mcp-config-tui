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

// CredentialValidator is a function that validates a credential string.
type CredentialValidator func(string) error

// CredentialSpec describes a credential required by an MCP provider.
type CredentialSpec struct {
	Key         string
	Label       string
	Description string
	Secret      bool
	MultiValue  bool
	Validator   CredentialValidator
}

// CredentialProfile represents a collected set of credentials for a specific provider.
type CredentialProfile struct {
	ProviderID string
	Values     map[string]string
	Label      string
}

// MCPProvider defines the contract for any MCP server we support.
type MCPProvider interface {
	ID() string
	Name() string
	Description() string
	// RequiredCredentials returns an ordered list of credential metadata.
	RequiredCredentials() []CredentialSpec
	// GenerateConfig builds the final MCPConfig based on the user's provided credentials.
	GenerateConfig(credentials map[string]string) (MCPConfig, error)
}
