package provider

import "testing"

func TestPlaywrightProvider_Metadata(t *testing.T) {
	p := NewPlaywrightProvider()

	if p.ID() != "playwright" {
		t.Errorf("expected ID playwright, got %q", p.ID())
	}
	if p.Name() != "Playwright" {
		t.Errorf("expected name Playwright, got %q", p.Name())
	}
	if p.Description() == "" {
		t.Error("expected description")
	}
	if len(p.RequiredCredentials()) != 0 {
		t.Fatalf("expected no required credentials, got %d", len(p.RequiredCredentials()))
	}
}

func TestPlaywrightProvider_GenerateConfig(t *testing.T) {
	p := NewPlaywrightProvider()

	cfg, err := p.GenerateConfig(nil)
	if err != nil {
		t.Fatalf("GenerateConfig returned error: %v", err)
	}

	if cfg.Type != TransportStdio {
		t.Errorf("expected TransportStdio, got %s", cfg.Type)
	}
	if cfg.Command != "npx" {
		t.Errorf("expected command npx, got %q", cfg.Command)
	}
	if len(cfg.Args) != 1 || cfg.Args[0] != "@playwright/mcp@latest" {
		t.Errorf("expected args [@playwright/mcp@latest], got %v", cfg.Args)
	}
	if len(cfg.Env) != 0 {
		t.Errorf("expected no env, got %v", cfg.Env)
	}
	if len(cfg.Headers) != 0 {
		t.Errorf("expected no headers, got %v", cfg.Headers)
	}
	if cfg.Runtime == nil || cfg.Runtime.Type != "npm" {
		t.Errorf("expected npm runtime, got %v", cfg.Runtime)
	}
}
