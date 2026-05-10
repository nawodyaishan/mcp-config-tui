package client

import (
    "strings"

    "github.com/nawodyaishan/universal-mcp-sync/pkg/config"
    "github.com/nawodyaishan/universal-mcp-sync/pkg/provider"
)

// Adapt returns a transport config suitable for appID.
//
// If the client supports cfg.Type natively, cfg is returned unchanged.
// If the client has a bridge for cfg.Type, the bridged stdio config is returned.
// If neither, cfg is returned unchanged — callers must check CanHandle first
// and set a SkipReason if it returns false.
func Adapt(appID config.AppID, cfg provider.MCPConfig) provider.MCPConfig {
	// Provider-specified override takes priority over matrix bridge
	if cfg.BridgeOverride != nil {
		cap, ok := Matrix[appID]
		if ok && !supportsTransport(cap.Supports, cfg.Type) {
			return applyBridge(cfg.BridgeOverride, cfg)
		}
	}

	cap, ok := Matrix[appID]
	if !ok {
		return cfg
	}
	if supportsTransport(cap.Supports, cfg.Type) {
		return cfg
	}
	if bridge, ok := cap.Bridge[cfg.Type]; ok {
		return applyBridge(bridge, cfg)
	}
	return cfg
}

// CanHandle reports whether appID can handle transport, either natively or via a bridge.
// Returns false if the client has no support and no bridge for the given transport.
func CanHandle(appID config.AppID, transport provider.TransportType) bool {
    cap, ok := Matrix[appID]
    if !ok {
        return false
    }
    if supportsTransport(cap.Supports, transport) {
        return true
    }
    _, hasBridge := cap.Bridge[transport]
    return hasBridge
}

func supportsTransport(s TransportSupport, t provider.TransportType) bool {
    switch t {
    case provider.TransportStdio:
        return s.Stdio
    case provider.TransportStreamableHTTP:
        return s.StreamableHTTP
    case provider.TransportSSE:
        return s.SSE
    case provider.TransportHTTP:
        return s.HTTP
    }
    return false
}

func applyBridge(bridge *provider.BridgeConfig, cfg provider.MCPConfig) provider.MCPConfig {
	args := make([]string, len(bridge.Args))
	for i, arg := range bridge.Args {
		// Substitute {url}
		arg = strings.ReplaceAll(arg, "{url}", cfg.URL)
		// Substitute {header:KEY} with the header value
		for k, v := range cfg.Headers {
			arg = strings.ReplaceAll(arg, "{header:"+k+"}", v)
		}
		args[i] = arg
	}
	return provider.MCPConfig{
		Type:    provider.TransportStdio,
		Command: bridge.Command,
		Args:    args,
	}
}

// HeadersFor returns the headers map to write for appID.
// Gemini CLI requires an extra Accept header for SSE streaming.
// Returns nil when base is empty (prevents serializing "headers": {}).
func HeadersFor(appID config.AppID, base map[string]string) map[string]string {
    if len(base) == 0 {
        return nil
    }
    out := make(map[string]string, len(base)+1)
    for k, v := range base {
        out[k] = v
    }
    if appID == config.AppGeminiCLI {
        out["Accept"] = "application/json, text/event-stream"
    }
    return out
}
