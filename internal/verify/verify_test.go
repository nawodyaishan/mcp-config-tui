package verify

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nawodyaishan/mcp-config-tui/internal/config"
)

func TestVerifyBareMCPServersFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp_config.json")
	content := `{
  "exa": {
    "type": "sse",
    "url": "https://mcp.exa.ai/mcp?exaApiKey=11111111-1111-1111-1111-111111111111&tools=web_search_exa"
  }
}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	result := VerifyFile(path, config.FileKindBareMCPServers, 1)
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
      "type": "sse",
      "url": "https://mcp.exa.ai/mcp?exaApiKey=11111111-1111-1111-1111-111111111111&tools=web_search_exa"
    }
  }
}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	result := VerifyFile(path, config.FileKindMCPServers, 1)
	if result.Status != StatusOK {
		t.Fatalf("expected status OK, got %s: %v", result.Status, result.Details)
	}
}
