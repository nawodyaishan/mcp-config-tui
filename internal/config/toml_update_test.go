package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpdateCodexTOMLReplacesExistingExaBlockOnce(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "codex.toml"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	exaURL := "https://mcp.exa.ai/mcp?exaApiKey=11111111-1111-1111-1111-111111111111&tools=web_search_exa"
	updated, err := UpdateCodexTOML(data, exaURL)
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
