package provider

import (
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
	creds := map[string]string{"CONTEXT7_API_KEY": "ctx7sk_abcdef1234567890wxyz"}
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
	if cfg.Headers["CONTEXT7_API_KEY"] != "ctx7sk_abcdef1234567890wxyz" {
		t.Errorf("unexpected header %s", cfg.Headers["CONTEXT7_API_KEY"])
	}
	if cfg.BridgeOverride == nil {
		t.Fatal("expected BridgeOverride")
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
	err = creds[0].Validator("invalid")
	if err == nil {
		t.Error("expected error for invalid key in validator")
	}
}