package client

import (
    "github.com/nawodyaishan/universal-mcp-sync/pkg/config"
    "github.com/nawodyaishan/universal-mcp-sync/pkg/provider"
)

// TransportSupport declares which MCP transport types a client handles natively.
type TransportSupport struct {
    Stdio          bool
    StreamableHTTP bool
    SSE            bool
    HTTP           bool // legacy; VS Code uses "type":"http"
}

// BridgeConfig describes a stdio wrapper that proxies a remote transport.
// {url} in Args is replaced with the actual server URL at adapt time.
type BridgeConfig struct {
    Command string
    Args    []string
}

// Capability is the full capability profile of one AI client.
type Capability struct {
    Supports TransportSupport
    // Bridge maps a transport type the client cannot handle natively to a
    // stdio bridge that wraps it. If no bridge is declared and the client
    // does not support the transport, CanHandle returns false.
    Bridge map[provider.TransportType]*BridgeConfig
}

// Matrix is the authoritative source of what each AI client supports.
// When adding a new client: add its AppID here with accurate capabilities.
// When a client gains a new transport: update its TransportSupport.
var Matrix = map[config.AppID]Capability{
    config.AppClaudeDesktop: {
        // Claude Desktop only speaks stdio natively.
        // Remote HTTP/StreamableHTTP servers are bridged via mcp-remote.
        Supports: TransportSupport{Stdio: true},
        Bridge: map[provider.TransportType]*BridgeConfig{
            provider.TransportStreamableHTTP: {
                Command: "npx",
                Args:    []string{"-y", "mcp-remote", "{url}"},
            },
            provider.TransportHTTP: {
                Command: "npx",
                Args:    []string{"-y", "mcp-remote", "{url}"},
            },
        },
    },
    config.AppClaudeCode: {
        // Managed via `claude mcp add` CLI; supports both transports.
        Supports: TransportSupport{StreamableHTTP: true, HTTP: true, Stdio: true},
    },
    config.AppCursor: {
        Supports: TransportSupport{StreamableHTTP: true, HTTP: true, Stdio: true},
    },
    config.AppVSCode: {
        // VS Code uses "type":"http" (not streamable-http) for HTTP servers.
        Supports: TransportSupport{HTTP: true, Stdio: true},
    },
    config.AppWindsurf: {
        Supports: TransportSupport{StreamableHTTP: true, HTTP: true, Stdio: true},
    },
    config.AppZed: {
        Supports: TransportSupport{StreamableHTTP: true, HTTP: true, Stdio: true},
    },
    config.AppRooCode: {
        // Roo Code uses "type":"streamable-http" extra field.
        Supports: TransportSupport{StreamableHTTP: true, HTTP: true, Stdio: true},
    },
    config.AppOpenCode: {
        Supports: TransportSupport{StreamableHTTP: true, HTTP: true, Stdio: true},
    },
    config.AppKiro: {
        Supports: TransportSupport{StreamableHTTP: true, HTTP: true, Stdio: true},
    },
    config.AppGeminiCLI: {
        // Gemini CLI does not support local stdio subprocess servers.
        Supports: TransportSupport{StreamableHTTP: true, HTTP: true},
    },
    config.AppAntigravity: {
        Supports: TransportSupport{StreamableHTTP: true, HTTP: true},
    },
    config.AppCodexCLI: {
        // TODO: Codex TOML writer does not support stdio yet, so mark false.
        Supports: TransportSupport{StreamableHTTP: true, HTTP: true, Stdio: false},
    },
}
