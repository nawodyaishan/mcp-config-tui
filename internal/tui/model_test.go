package tui

import (
	"testing"

	"github.com/nawodyaishan/mcp-config-tui/internal/app"
	"github.com/nawodyaishan/mcp-config-tui/internal/config"
)

func TestSetupFormSyncsToContext(t *testing.T) {
	manager, _ := app.NewManager("/tmp/test", nil, nil)
	ctx := &wizardContext{
		manager:  manager,
		selected: make(map[config.AppID]bool),
	}
	sf := newSetupForm(ctx, "11111111-1111-1111-1111-111111111111")
	sf.selectedSlice = []config.AppID{config.AppClaudeDesktop}
	sf.syncToContext()

	if !ctx.selected[config.AppClaudeDesktop] {
		t.Fatal("expected Claude Desktop to be selected in context")
	}
	if len(ctx.keys) != 1 || ctx.keys[0] != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("expected key to be synced, got %#v", ctx.keys)
	}
}
