package verify

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nawodyaishan/universal-mcp-sync/pkg/config"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/provider"
)

func TestVerifyBareMCPServersFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp_config.json")
	content := `{
  "exa": {
    "httpUrl": "https://mcp.exa.ai/mcp?exaApiKey=11111111-1111-1111-1111-111111111111&tools=web_search_exa,web_search_advanced_exa,web_fetch_exa"
  }
}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	result := VerifyFile(path, config.FileKindBareMCPServers, 3)
	if result.Status != StatusOK {
		t.Fatalf("expected status OK, got %s: %v", result.Status, result.Details)
	}
}

func TestVerifyMCPServersFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	content := `{
  "mcpServers": {
    "exa": {
      "url": "https://mcp.exa.ai/mcp?exaApiKey=11111111-1111-1111-1111-111111111111&tools=web_search_exa,web_search_advanced_exa,web_fetch_exa"
    }
  }
}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	result := VerifyFile(path, config.FileKindMCPServers, 3)
	if result.Status != StatusOK {
		t.Fatalf("expected status OK, got %s: %v", result.Status, result.Details)
	}
}

func TestVerifyProviderFileSupportsClaudeDesktopStdioBridge(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "claude_desktop_config.json")
	content := `{
  "mcpServers": {
    "exa": {
      "command": "npx",
      "args": ["-y", "mcp-remote", "https://mcp.exa.ai/mcp?exaApiKey=11111111-1111-1111-1111-111111111111&tools=web_search_exa,web_search_advanced_exa,web_fetch_exa"]
    }
  }
}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	result := VerifyProviderFile(path, config.FileKindMCPServers, "exa", provider.MCPConfig{
		Type:    provider.TransportStdio,
		Command: "npx",
		Args:    []string{"-y", "mcp-remote", "https://mcp.exa.ai/mcp?exaApiKey=11111111-1111-1111-1111-111111111111&tools=web_search_exa,web_search_advanced_exa,web_fetch_exa"},
	})
	if result.Status != StatusOK {
		t.Fatalf("expected status OK, got %s: %v", result.Status, result.Details)
	}
}

func TestVerifyProviderFile_GenericStdio(t *testing.T) {
    dir := t.TempDir()
    path := filepath.Join(dir, "config.json")
    content := `{"mcpServers":{"github":{"command":"npx","args":["-y","@modelcontextprotocol/server-github"]}}}`
    _ = os.WriteFile(path, []byte(content), 0o600)

    cfg := provider.MCPConfig{Type: provider.TransportStdio, Command: "npx"}
    result := VerifyProviderFile(path, config.FileKindMCPServers, "github", cfg)
    if result.Status != StatusOK {
        t.Errorf("expected OK, got %s: %v", result.Status, result.Details)
    }
}

func TestVerifyProviderFile_GenericHTTP(t *testing.T) {
    dir := t.TempDir()
    path := filepath.Join(dir, "config.json")
    content := `{"mcpServers":{"brave":{"url":"https://api.brave.com/mcp"}}}`
    _ = os.WriteFile(path, []byte(content), 0o600)

    cfg := provider.MCPConfig{Type: provider.TransportStreamableHTTP, URL: "https://api.brave.com/mcp"}
    result := VerifyProviderFile(path, config.FileKindMCPServers, "brave", cfg)
    if result.Status != StatusOK {
        t.Errorf("expected OK, got %s: %v", result.Status, result.Details)
    }
}

func TestVerifyProviderFile_GenericMissingEntry(t *testing.T) {
    dir := t.TempDir()
    path := filepath.Join(dir, "config.json")
    _ = os.WriteFile(path, []byte(`{"mcpServers":{}}`), 0o600)

    cfg := provider.MCPConfig{Type: provider.TransportStdio, Command: "npx"}
    result := VerifyProviderFile(path, config.FileKindMCPServers, "github", cfg)
    if result.Status != StatusFailed {
        t.Errorf("expected Failed for missing entry, got %s", result.Status)
    }
}
