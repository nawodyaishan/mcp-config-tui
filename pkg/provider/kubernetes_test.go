package provider

import "testing"

func TestKubernetesProvider_Metadata(t *testing.T) {
	p := NewKubernetesProvider()

	if p.ID() != "kubernetes" {
		t.Errorf("expected ID kubernetes, got %q", p.ID())
	}
	if p.Name() != "Kubernetes" {
		t.Errorf("expected name Kubernetes, got %q", p.Name())
	}
	if p.Description() == "" {
		t.Error("expected description")
	}
	if len(p.RequiredCredentials()) != 0 {
		t.Fatalf("expected no required credentials, got %d", len(p.RequiredCredentials()))
	}
}

func TestKubernetesProvider_GenerateConfig(t *testing.T) {
	p := NewKubernetesProvider()

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
	wantArgs := []string{"-y", "kubernetes-mcp-server@latest", "--read-only"}
	if len(cfg.Args) != len(wantArgs) {
		t.Fatalf("expected args %v, got %v", wantArgs, cfg.Args)
	}
	for i, want := range wantArgs {
		if cfg.Args[i] != want {
			t.Errorf("arg %d: got %q, want %q", i, cfg.Args[i], want)
		}
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
