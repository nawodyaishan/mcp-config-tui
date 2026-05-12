package provider

import (
	"testing"
)

func TestTavilyProvider_ID(t *testing.T) {
	p := NewTavilyProvider()
	if p.ID() != "tavily" {
		t.Errorf("expected ID 'tavily', got '%s'", p.ID())
	}
}

func TestTavilyProvider_GenerateConfig(t *testing.T) {
	p := NewTavilyProvider()

	validKey := "tvly-abcdef1234567890wxyz"
	cfg, err := p.GenerateConfig(map[string]string{
		"TAVILY_API_KEY": validKey,
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.Type != TransportStdio {
		t.Errorf("expected TransportStdio, got %v", cfg.Type)
	}
	if cfg.Command != "npx" {
		t.Errorf("expected command 'npx', got %q", cfg.Command)
	}
	if len(cfg.Args) != 2 || cfg.Args[1] != "tavily-mcp@latest" {
		t.Errorf("expected args [-y tavily-mcp@latest], got %v", cfg.Args)
	}
	if cfg.Env["TAVILY_API_KEY"] != validKey {
		t.Errorf("expected env TAVILY_API_KEY to be set, got %v", cfg.Env)
	}
	if cfg.Runtime == nil || cfg.Runtime.Type != "npm" {
		t.Errorf("expected runtime npm, got %v", cfg.Runtime)
	}
}

func TestTavilyProvider_GenerateConfig_Invalid(t *testing.T) {
	p := NewTavilyProvider()

	_, err := p.GenerateConfig(map[string]string{
		"TAVILY_API_KEY": "invalid-key",
	})

	if err == nil {
		t.Fatal("expected error for invalid key")
	}
}
