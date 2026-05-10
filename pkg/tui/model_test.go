package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/app"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/config"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/provider"
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

func TestModel(t *testing.T) {
	manager, _ := app.NewManager("/tmp/test", nil, nil)
	m := NewModel(manager, nil, "")
	
	// Init
	m.Init()
	
	// Update WindowSize
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 100})
	if m2.(Model).width != 100 {
		t.Errorf("expected width 100, got %d", m2.(Model).width)
	}
	
	// View
	m2.View()
	
	// Err
	if m2.(Model).Err() != nil {
		t.Errorf("expected nil error")
	}
}

func TestModelPreloaded(t *testing.T) {
	manager, _ := app.NewManager("/tmp/test", nil, nil)
	keys := []string{"11111111-1111-1111-1111-111111111111"}
	m := NewModel(manager, keys, "")
	if !m.ctx.isPreloaded {
		t.Error("expected preloaded true")
	}
	if len(m.ctx.profiles) != 1 {
		t.Errorf("expected 1 profile, got %d", len(m.ctx.profiles))
	}
}
