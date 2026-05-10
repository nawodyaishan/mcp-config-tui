package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/nawodyaishan/universal-mcp-sync/pkg/config"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/provider"
)

func TestSelectedAppIDs(t *testing.T) {
	apps := []config.AppConfig{
		{ID: config.AppCursor},
		{ID: config.AppVSCode},
	}
	selected := map[config.AppID]bool{
		config.AppCursor: true,
	}
	got := selectedAppIDs(apps, selected)
	if len(got) != 1 || got[0] != config.AppCursor {
		t.Errorf("expected [cursor], got %v", got)
	}
}

func TestAssignmentLabel(t *testing.T) {
	profiles := []provider.CredentialProfile{
		{Label: "prof1"},
	}
	if got := assignmentLabel(profiles, 0); got != "prof1" {
		t.Errorf("expected prof1, got %s", got)
	}
	if got := assignmentLabel(profiles, 1); got != "unassigned" {
		t.Errorf("expected unassigned, got %s", got)
	}
}

func TestRenderError(t *testing.T) {
	if renderError(nil) != "" {
		t.Error("expected empty string for nil error")
	}
	err := fmt.Errorf("test error")
	got := renderError(err)
	if !strings.Contains(got, "test error") {
		t.Errorf("expected error message in output, got %s", got)
	}
}

func TestRenderKeyHelp(t *testing.T) {
	got := renderKeyHelp("tab", "enter")
	if got != "[tab] [enter]" {
		t.Errorf("expected [tab] [enter], got %s", got)
	}
}

func TestRenderSection(t *testing.T) {
	got := renderSection("title", "body", "help")
	if !strings.Contains(got, "title") || !strings.Contains(got, "body") || !strings.Contains(got, "help") {
		t.Errorf("expected all parts in output, got %s", got)
	}
}

func TestRenderShell(t *testing.T) {
	got := renderShell("test body", stageSetup, 100)
	if !strings.Contains(got, "test body") || !strings.Contains(got, "1 Setup") {
		t.Errorf("expected body and stage in output, got %s", got)
	}
}
