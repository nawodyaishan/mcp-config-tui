package provider

type TransportType string

const (
	TransportStdio          TransportType = "stdio"
	TransportStreamableHTTP TransportType = "streamable-http"
	TransportSSE            TransportType = "sse"
	TransportHTTP           TransportType = "http" // legacy; kept for VS Code "type":"http" compat
)

// PackageRuntime describes the packaging type of a stdio server.
// Used to communicate install context (npm, pypi, oci) to UI layers.
// Nil means the provider is a remote HTTP server.
type PackageRuntime struct {
	Type string // "npm" | "pypi" | "oci" | "mcpb"
}

// BridgeConfig describes a stdio wrapper that proxies a remote transport.
type BridgeConfig struct {
	Command string
	Args    []string
}

// MCPConfig is a provider-agnostic description of one MCP server connection.
type MCPConfig struct {
	Type           TransportType
	URL            string            // HTTP / SSE / StreamableHTTP
	Command        string            // stdio: executable name, e.g. "npx"
	Args           []string          // stdio: arguments after command
	Env            map[string]string // stdio: env vars injected into the subprocess
	Headers        map[string]string // Per-server HTTP headers for remote transports. Nil for stdio.
	Runtime        *PackageRuntime   // non-nil for packaged stdio servers; nil for remote
	BridgeOverride *BridgeConfig     // when non-nil, used by client.Adapt in place of Matrix bridge
}

// CredentialValidator validates one credential string value.
type CredentialValidator func(string) error

// CredentialSpec describes one credential field required by a provider.
type CredentialSpec struct {
	Key         string
	Label       string
	Description string
	Secret      bool
	MultiValue  bool
	Validator   CredentialValidator
}

// CredentialProfile is a collected set of credentials for one provider instance.
type CredentialProfile struct {
	ProviderID string
	Values     map[string]string
	Label      string // redacted display string shown in UI
}

// MCPProvider is the contract every MCP server plugin must implement.
type MCPProvider interface {
	ID() string
	Name() string
	Description() string
	RequiredCredentials() []CredentialSpec
	GenerateConfig(credentials map[string]string) (MCPConfig, error)
}

// MultiValueParser is an optional interface for providers whose credential
// input accepts multiple values in one text area (e.g. Exa's multi-key paste field).
// If a provider does not implement this interface, the TUI creates one profile
// per form submission using the raw input values directly.
type MultiValueParser interface {
	// ParseMultiValue parses raw text for credential key into one or more profiles.
	// The returned profiles each represent one independent credential set.
	ParseMultiValue(credentialKey string, raw string) ([]CredentialProfile, error)
}
