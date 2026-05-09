package tui

import (
	"testing"

	"github.com/nawodyaishan/mcp-config-tui/pkg/app"
	"github.com/nawodyaishan/mcp-config-tui/pkg/config"
	"github.com/nawodyaishan/mcp-config-tui/pkg/provider"
)

func TestSetupFormSyncsToContext(t *testing.T) {
	manager, _ := app.NewManager("/tmp/test", nil, nil)
	registry := provider.DefaultRegistry()
	ctx := &wizardContext{
		manager:  manager,
		registry: registry,
		selected: make(map[config.AppID]bool),
	}
	sf := newSetupForm(ctx, "11111111-1111-1111-1111-111111111111")
	sf.selectedProvider = "exa"
	sf.selectedSlice = []config.AppID{config.AppClaudeDesktop}
	sf.syncToContext()

	if !ctx.selected[config.AppClaudeDesktop] {
		t.Fatal("expected Claude Desktop to be selected in context")
	}
	if len(ctx.profiles) != 1 || ctx.profiles[0].Values["EXA_API_KEY"] != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("expected profile to be synced, got %#v", ctx.profiles)
	}
}
