package config

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nawodyaishan/universal-mcp-sync/pkg/provider"
)

func TestUpdateCodexTOMLReplacesExistingExaBlockOnce(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "codex.toml"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	exaURL := "https://mcp.exa.ai/mcp?exaApiKey=11111111-1111-1111-1111-111111111111&tools=web_search_exa"
	cfg := provider.MCPConfig{Type: provider.TransportHTTP, URL: exaURL}
	updated, err := UpdateCodexTOML(data, "exa", cfg)
	if err != nil {
		t.Fatalf("UpdateCodexTOML returned error: %v", err)
	}

	text := string(updated)
	if strings.Count(text, "[mcp_servers.exa]") != 1 {
		t.Fatalf("expected exactly one Exa block, got:\n%s", text)
	}
	if !strings.Contains(text, `url = "`+exaURL+`"`) {
		t.Fatalf("expected updated Exa URL, got:\n%s", text)
	}
	if !strings.Contains(text, "[mcp_servers.context7]") {
		t.Fatalf("expected unrelated TOML sections to remain, got:\n%s", text)
	}
}

func TestUpdateCodexTOML_WritesHttpHeaders(t *testing.T) {
	cfg := provider.MCPConfig{
		Type:    provider.TransportStreamableHTTP,
		URL:     "https://mcp.context7.com/mcp",
		Headers: map[string]string{"CONTEXT7_API_KEY": "ctx7sk_test"},
	}
	result, err := UpdateCodexTOML([]byte(""), "context7", cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(result, []byte(`http_headers`)) {
		t.Errorf("expected http_headers in TOML:\n%s", result)
	}
	if !bytes.Contains(result, []byte(`"CONTEXT7_API_KEY"`)) {
		t.Errorf("expected header key:\n%s", result)
	}
}
