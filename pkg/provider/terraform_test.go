package provider

import "testing"

func TestTerraformProvider_Metadata(t *testing.T) {
	p := NewTerraformProvider()

	if p.ID() != "terraform" {
		t.Errorf("expected ID terraform, got %q", p.ID())
	}
	if p.Name() != "Terraform" {
		t.Errorf("expected name Terraform, got %q", p.Name())
	}
	if p.Description() == "" {
		t.Error("expected description")
	}
	if len(p.RequiredCredentials()) != 0 {
		t.Fatalf("expected no required credentials, got %d", len(p.RequiredCredentials()))
	}
}

func TestTerraformProvider_GenerateConfig(t *testing.T) {
	p := NewTerraformProvider()

	cfg, err := p.GenerateConfig(nil)
	if err != nil {
		t.Fatalf("GenerateConfig returned error: %v", err)
	}

	if cfg.Type != TransportStdio {
		t.Errorf("expected TransportStdio, got %s", cfg.Type)
	}
	if cfg.Command != "docker" {
		t.Errorf("expected command docker, got %q", cfg.Command)
	}
	wantArgs := []string{
		"run",
		"-i",
		"--rm",
		"-e",
		"ENABLE_TF_OPERATIONS=false",
		"hashicorp/terraform-mcp-server:0.5.2",
	}
	if len(cfg.Args) != len(wantArgs) {
		t.Fatalf("expected args %v, got %v", wantArgs, cfg.Args)
	}
	for i, want := range wantArgs {
		if cfg.Args[i] != want {
			t.Errorf("arg %d: got %q, want %q", i, cfg.Args[i], want)
		}
	}
	if len(cfg.Env) != 0 {
		t.Errorf("expected no process env, got %v", cfg.Env)
	}
	if len(cfg.Headers) != 0 {
		t.Errorf("expected no headers, got %v", cfg.Headers)
	}
	if cfg.Runtime == nil || cfg.Runtime.Type != "oci" {
		t.Errorf("expected oci runtime, got %v", cfg.Runtime)
	}
}
