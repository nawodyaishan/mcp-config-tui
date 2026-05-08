package tui

import (
	"strings"
	"testing"

	"github.com/nawodyaishan/mcp-config-tui/internal/app"
)

func TestNewModelDoesNotRenderPreloadedRawKeys(t *testing.T) {
	manager, err := app.NewManager("/tmp/exa-mcp-manager-test", nil, nil)
	if err != nil {
		t.Fatalf("NewManager returned error: %v", err)
	}

	key := "11111111-1111-1111-1111-111111111111"
	model := NewModel(manager, []string{key}, key)
	view := renderKeys(model)

	if strings.Contains(view, key) {
		t.Fatalf("expected key screen to hide preloaded raw key, got:\n%s", view)
	}
	if !strings.Contains(view, "1111...1111") {
		t.Fatalf("expected redacted key label, got:\n%s", view)
	}
}
