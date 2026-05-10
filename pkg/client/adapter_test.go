package client_test

import (
	"testing"

	"github.com/nawodyaishan/universal-mcp-sync/pkg/client"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/config"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/provider"
)

func TestAdapt(t *testing.T) {
	remoteHTTP := provider.MCPConfig{
		Type: provider.TransportStreamableHTTP,
		URL:  "https://mcp.exa.ai/mcp?exaApiKey=test",
	}
	remoteHTTPLegacy := provider.MCPConfig{
		Type: provider.TransportHTTP,
		URL:  "https://mcp.exa.ai/mcp?exaApiKey=test",
	}
	stdioGitHub := provider.MCPConfig{
		Type:    provider.TransportStdio,
		Command: "npx",
		Args:    []string{"-y", "@modelcontextprotocol/server-github"},
		Env:     map[string]string{"GITHUB_PERSONAL_ACCESS_TOKEN": "ghp_test"},
	}

	tests := []struct {
		name     string
		appID    config.AppID
		input    provider.MCPConfig
		wantType provider.TransportType
		wantCmd  string
		wantURL  string // non-empty means check URL preserved in bridge args
	}{
		{
			name:     "ClaudeDesktop bridges StreamableHTTP to stdio",
			appID:    config.AppClaudeDesktop,
			input:    remoteHTTP,
			wantType: provider.TransportStdio,
			wantCmd:  "npx",
			wantURL:  remoteHTTP.URL,
		},
		{
			name:     "ClaudeDesktop bridges legacy HTTP to stdio",
			appID:    config.AppClaudeDesktop,
			input:    remoteHTTPLegacy,
			wantType: provider.TransportStdio,
			wantCmd:  "npx",
			wantURL:  remoteHTTPLegacy.URL,
		},
		{
			name:     "ClaudeDesktop passes stdio through unchanged",
			appID:    config.AppClaudeDesktop,
			input:    stdioGitHub,
			wantType: provider.TransportStdio,
			wantCmd:  "npx",
		},
		{
			name:     "Cursor passes StreamableHTTP through unchanged",
			appID:    config.AppCursor,
			input:    remoteHTTP,
			wantType: provider.TransportStreamableHTTP,
		},
		{
			name:     "Cursor passes stdio through unchanged",
			appID:    config.AppCursor,
			input:    stdioGitHub,
			wantType: provider.TransportStdio,
			wantCmd:  "npx",
		},
		{
			name:     "GeminiCLI passes StreamableHTTP unchanged (no bridge needed)",
			appID:    config.AppGeminiCLI,
			input:    remoteHTTP,
			wantType: provider.TransportStreamableHTTP,
		},
		{
			name:     "GeminiCLI returns stdio unchanged (caller must check CanHandle)",
			appID:    config.AppGeminiCLI,
			input:    stdioGitHub,
			wantType: provider.TransportStdio, // pass-through, CanHandle=false
		},
		{
			name:     "Unknown AppID returns cfg unchanged",
			appID:    config.AppID("does-not-exist"),
			input:    remoteHTTP,
			wantType: provider.TransportStreamableHTTP,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := client.Adapt(tt.appID, tt.input)
			if got.Type != tt.wantType {
				t.Errorf("Type: got %q, want %q", got.Type, tt.wantType)
			}
			if tt.wantCmd != "" && got.Command != tt.wantCmd {
				t.Errorf("Command: got %q, want %q", got.Command, tt.wantCmd)
			}
			if tt.wantURL != "" {
				found := false
				for _, arg := range got.Args {
					if arg == tt.wantURL {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("URL %q not found in bridge args %v", got.Args, tt.wantURL)
				}
			}
		})
	}
}

func TestCanHandle(t *testing.T) {
	tests := []struct {
		appID     config.AppID
		transport provider.TransportType
		want      bool
	}{
		{config.AppClaudeDesktop, provider.TransportStdio, true},
		{config.AppClaudeDesktop, provider.TransportStreamableHTTP, true}, // via bridge
		{config.AppClaudeDesktop, provider.TransportHTTP, true},           // via bridge
		{config.AppGeminiCLI, provider.TransportStreamableHTTP, true},
		{config.AppGeminiCLI, provider.TransportStdio, false}, // no support, no bridge
		{config.AppAntigravity, provider.TransportStdio, false},
		{config.AppCursor, provider.TransportStdio, true},
		{config.AppID("unknown"), provider.TransportStdio, false},
	}
	for _, tt := range tests {
		got := client.CanHandle(tt.appID, tt.transport)
		if got != tt.want {
			t.Errorf("CanHandle(%q, %q) = %v, want %v", tt.appID, tt.transport, got, tt.want)
		}
	}
}

func TestHeadersFor_GeminiAddsAccept(t *testing.T) {
	base := map[string]string{"CONTEXT7_API_KEY": "ctx7sk_test"}
	got := client.HeadersFor(config.AppGeminiCLI, base)
	if got["Accept"] == "" {
		t.Error("expected Accept header for Gemini CLI")
	}
}

func TestHeadersFor_NilBaseReturnsNil(t *testing.T) {
	if client.HeadersFor(config.AppCursor, nil) != nil {
		t.Error("nil base must return nil (no empty headers map)")
	}
}

func TestHeadersFor_CursorUnchanged(t *testing.T) {
	base := map[string]string{"CONTEXT7_API_KEY": "ctx7sk_test"}
	got := client.HeadersFor(config.AppCursor, base)
	if _, ok := got["Accept"]; ok {
		t.Error("Cursor must not gain Accept header")
	}
}
