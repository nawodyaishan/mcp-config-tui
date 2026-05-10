package config

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/nawodyaishan/universal-mcp-sync/pkg/provider"
)

func TestUpdateMCPServersJSONPreservesUnrelatedFields(t *testing.T) {
	data := mustReadFixture(t, "claude_desktop.json")
	exaURL := "https://mcp.exa.ai/mcp?exaApiKey=11111111-1111-1111-1111-111111111111&tools=web_search_exa"
	cfg := provider.MCPConfig{Type: provider.TransportHTTP, URL: exaURL}

	updated, err := UpdateMCPServersJSON(data, "exa", "mcpServers", "url", cfg, nil)
	if err != nil {
		t.Fatalf("UpdateMCPServersJSON returned error: %v", err)
	}

	root := decodeJSONForTest(t, updated)
	if root["theme"] != "dark" {
		t.Fatalf("expected unrelated theme field to survive, got %#v", root["theme"])
	}

	servers := root["mcpServers"].(map[string]any)
	if _, ok := servers["context7"]; !ok {
		t.Fatal("expected existing context7 server to remain")
	}
	exa := servers["exa"].(map[string]any)
	if exa["url"] != exaURL {
		t.Fatalf("expected Exa URL to be updated, got %#v", exa["url"])
	}
}

func TestUpdateMCPServersJSONSupportsStdioServers(t *testing.T) {
	data := mustReadFixture(t, "claude_desktop.json")
	cfg := provider.MCPConfig{
		Type:    provider.TransportStdio,
		Command: "npx",
		Args:    []string{"-y", "mcp-remote", "https://mcp.exa.ai/mcp?exaApiKey=11111111-1111-1111-1111-111111111111&tools=web_search_exa,web_search_advanced_exa,web_fetch_exa"},
	}

	updated, err := UpdateMCPServersJSON(data, "exa", "mcpServers", "url", cfg, nil)
	if err != nil {
		t.Fatalf("UpdateMCPServersJSON returned error: %v", err)
	}

	root := decodeJSONForTest(t, updated)
	servers := root["mcpServers"].(map[string]any)
	exa := servers["exa"].(map[string]any)
	if exa["command"] != "npx" {
		t.Fatalf("expected stdio command to be set, got %#v", exa["command"])
	}
	args := exa["args"].([]any)
	if len(args) != 3 || args[1] != "mcp-remote" {
		t.Fatalf("expected mcp-remote args, got %#v", args)
	}
	if _, ok := exa["url"]; ok {
		t.Fatalf("did not expect url field for stdio config, got %#v", exa["url"])
	}
}

func TestUpdateGeminiSettingsPreservesUISecurity(t *testing.T) {
	data := mustReadFixture(t, "gemini_settings.json")
	exaURL := "https://mcp.exa.ai/mcp?exaApiKey=11111111-1111-1111-1111-111111111111&tools=web_search_exa"
	cfg := provider.MCPConfig{Type: provider.TransportHTTP, URL: exaURL}

	updated, err := UpdateMCPServersJSON(data, "exa", "mcpServers", "httpUrl", cfg, nil)
	if err != nil {
		t.Fatalf("UpdateMCPServersJSON returned error: %v", err)
	}

	root := decodeJSONForTest(t, updated)
	if _, ok := root["ui"].(map[string]any); !ok {
		t.Fatal("expected ui field to remain")
	}
	if _, ok := root["security"].(map[string]any); !ok {
		t.Fatal("expected security field to remain")
	}
	servers := root["mcpServers"].(map[string]any)
	exa := servers["exa"].(map[string]any)
	if exa["httpUrl"] != exaURL {
		t.Fatalf("expected httpUrl to be set, got %#v", exa["httpUrl"])
	}
}

func TestUpdateBareMCPServersJSON(t *testing.T) {
	data := []byte("{\n  \"other\": {\n    \"url\": \"https://example.com\"\n  }\n}\n")
	exaURL := "https://mcp.exa.ai/mcp?exaApiKey=11111111-1111-1111-1111-111111111111&tools=web_search_exa"
	cfg := provider.MCPConfig{Type: provider.TransportHTTP, URL: exaURL}

	updated, err := UpdateBareMCPServersJSON(data, "exa", "httpUrl", cfg, nil)
	if err != nil {
		t.Fatalf("UpdateBareMCPServersJSON returned error: %v", err)
	}

	root := decodeJSONForTest(t, updated)
	if _, ok := root["mcpServers"]; ok {
		t.Fatal("did not expect mcpServers root key")
	}
	exa := root["exa"].(map[string]any)
	if exa["httpUrl"] != exaURL {
		t.Fatalf("expected Exa URL to be updated, got %#v", exa["httpUrl"])
	}
	if _, ok := root["other"].(map[string]any); !ok {
		t.Fatal("expected unrelated server entries to remain")
	}
}

func TestUpdateNamedServerJSONReplacesMalformedAntigravityURL(t *testing.T) {
	data := mustReadFixture(t, "antigravity.json")
	exaURL := "https://mcp.exa.ai/mcp?exaApiKey=11111111-1111-1111-1111-111111111111&tools=web_search_exa"
	cfg := provider.MCPConfig{Type: provider.TransportHTTP, URL: exaURL}

	updated, err := UpdateNamedServerJSON(data, "exa", "", "serverUrl", cfg, nil)
	if err != nil {
		t.Fatalf("UpdateNamedServerJSON returned error: %v", err)
	}

	root := decodeJSONForTest(t, updated)
	exa := root["exa"].(map[string]any)
	if exa["serverUrl"] != exaURL {
		t.Fatalf("expected Exa serverUrl to be replaced, got %#v", exa["serverUrl"])
	}
	if _, ok := root["other"].(map[string]any); !ok {
		t.Fatal("expected unrelated server entries to remain")
	}
}

func TestUpdateMCPServersJSONWithExtraFields(t *testing.T) {
	exaURL := "https://mcp.exa.ai/mcp"
	cfg := provider.MCPConfig{Type: provider.TransportHTTP, URL: exaURL}
	extra := map[string]any{"type": "streamable-http"}

	updated, err := UpdateMCPServersJSON(nil, "exa", "mcpServers", "url", cfg, extra)
	if err != nil {
		t.Fatalf("UpdateMCPServersJSON: %v", err)
	}

	root := decodeJSONForTest(t, updated)
	exa := root["mcpServers"].(map[string]any)["exa"].(map[string]any)
	if exa["type"] != "streamable-http" {
		t.Errorf("expected extra field 'type', got %#v", exa["type"])
	}
}

func mustReadFixture(t *testing.T, name string) []byte {
	t.Helper()
	path := filepath.Join("testdata", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", path, err)
	}
	return data
}

func decodeJSONForTest(t *testing.T, data []byte) map[string]any {
	t.Helper()
	root := make(map[string]any)
	if err := json.Unmarshal(data, &root); err != nil {
		t.Fatalf("parse JSON: %v", err)
	}
	return root
}

func TestBuildConfigMap_EmitsHeadersWhenPresent(t *testing.T) {
	cfg := provider.MCPConfig{
		Type:    provider.TransportStreamableHTTP,
		URL:     "https://mcp.context7.com/mcp",
		Headers: map[string]string{"CONTEXT7_API_KEY": "ctx7sk_test"},
	}
	result, _ := UpdateMCPServersJSON([]byte("{}"), "context7", "mcpServers", "url", cfg, nil)
	if !bytes.Contains(result, []byte(`"headers"`)) {
		t.Errorf("expected headers in output:\n%s", result)
	}
	if !bytes.Contains(result, []byte(`"CONTEXT7_API_KEY"`)) {
		t.Errorf("expected header key in output:\n%s", result)
	}
}

func TestUpdateNamedServerJSON_Stdio(t *testing.T) {
	data := []byte(`{"mcp":{}}`)
	cfg := provider.MCPConfig{
		Type:    provider.TransportStdio,
		Command: "npx",
		Args:    []string{"-y", "mcp-remote", "url"},
	}
	updated, err := UpdateNamedServerJSON(data, "exa", "mcp", "url", cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	root := decodeJSONForTest(t, updated)
	mcp := root["mcp"].(map[string]any)
	exa := mcp["exa"].(map[string]any)
	if exa["command"] != "npx" {
		t.Errorf("expected command npx, got %v", exa["command"])
	}
}
