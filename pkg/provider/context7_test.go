package provider

import (
	"strings"
	"testing"

	"github.com/nawodyaishan/universal-mcp-sync/pkg/context7"
)

func TestContext7Provider_ID(t *testing.T) {
	p := NewContext7Provider()
	if p.ID() != "context7" {
		t.Errorf("unexpected ID %s", p.ID())
	}
}

func TestContext7Provider_GenerateConfig_Valid(t *testing.T) {
	p := NewContext7Provider()
	key := "ctx7sk-06801456-a80a-4de8-b6a1-ee189c839918"
	creds := map[string]string{"CONTEXT7_API_KEY": key}
	cfg, err := p.GenerateConfig(creds)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Type != TransportStreamableHTTP {
		t.Errorf("unexpected transport %s", cfg.Type)
	}
	if cfg.URL != context7.Endpoint {
		t.Errorf("unexpected URL %s", cfg.URL)
	}
	if cfg.Headers["CONTEXT7_API_KEY"] != key {
		t.Errorf("unexpected header %s", cfg.Headers["CONTEXT7_API_KEY"])
	}
	if cfg.BridgeOverride == nil {
		t.Fatal("expected BridgeOverride")
	}
}

func TestContext7Provider_GenerateConfig_LegacyUnderscoreKey(t *testing.T) {
	p := NewContext7Provider()
	key := "ctx7sk_abcdef1234567890wxyz"
	creds := map[string]string{"CONTEXT7_API_KEY": key}
	cfg, err := p.GenerateConfig(creds)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Headers["CONTEXT7_API_KEY"] != key {
		t.Errorf("unexpected header %s", cfg.Headers["CONTEXT7_API_KEY"])
	}
}

func TestContext7Provider_GenerateConfig_InvalidKey(t *testing.T) {
	p := NewContext7Provider()
	creds := map[string]string{"CONTEXT7_API_KEY": "invalid"}
	_, err := p.GenerateConfig(creds)
	if err == nil {
		t.Error("expected error for invalid key")
	}
}

func TestContext7Provider_CredentialValidator(t *testing.T) {
	p := NewContext7Provider()
	creds := p.RequiredCredentials()
	if len(creds) != 1 {
		t.Fatalf("unexpected creds len %d", len(creds))
	}
	err := creds[0].Validator("ctx7sk_abcdef1234567890wxyz")
	if err != nil {
		t.Error(err)
	}
	err = creds[0].Validator("ctx7sk-06801456-a80a-4de8-b6a1-ee189c839918")
	if err != nil {
		t.Error(err)
	}
	err = creds[0].Validator("invalid")
	if err == nil {
		t.Error("expected error for invalid key in validator")
	}
}

func TestContext7Provider_CredentialGuidance(t *testing.T) {
	p := NewContext7Provider()
	creds := p.RequiredCredentials()
	if len(creds) != 1 {
		t.Fatalf("unexpected creds len %d", len(creds))
	}
	description := creds[0].Description
	for _, want := range []string{
		"context7.com/dashboard",
		"API Keys",
		"ctx7sk-...",
		"ctx7sk_...",
	} {
		if !strings.Contains(description, want) {
			t.Fatalf("credential description %q does not contain %q", description, want)
		}
	}
}
