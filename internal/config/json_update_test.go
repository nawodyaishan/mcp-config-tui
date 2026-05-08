package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestUpdateMCPServersJSONPreservesUnrelatedFields(t *testing.T) {
	data := mustReadFixture(t, "claude_desktop.json")
	exaURL := "https://mcp.exa.ai/mcp?exaApiKey=11111111-1111-1111-1111-111111111111&tools=web_search_exa"

	updated, err := UpdateMCPServersJSON(data, exaURL)
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

func TestUpdateGeminiSettingsPreservesUISecurity(t *testing.T) {
	data := mustReadFixture(t, "gemini_settings.json")
	exaURL := "https://mcp.exa.ai/mcp?exaApiKey=11111111-1111-1111-1111-111111111111&tools=web_search_exa"

	updated, err := UpdateMCPServersJSON(data, exaURL)
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
}

func TestUpdateNamedServerJSONReplacesMalformedAntigravityURL(t *testing.T) {
	data := mustReadFixture(t, "antigravity.json")
	exaURL := "https://mcp.exa.ai/mcp?exaApiKey=11111111-1111-1111-1111-111111111111&tools=web_search_exa"

	updated, err := UpdateNamedServerJSON(data, "exa", "serverUrl", exaURL)
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
