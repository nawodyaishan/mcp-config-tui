package provider

import (
	"testing"
)

func TestDefaultRegistryContainsExa(t *testing.T) {
	r := DefaultRegistry()
	all := r.All()

	if len(all) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(all))
	}

	if all[0].ID() != "exa" {
		t.Fatalf("expected first provider to be exa, got %s", all[0].ID())
	}

	p, ok := r.Get("exa")
	if !ok {
		t.Fatal("expected to find exa provider by ID")
	}

	if p.Name() != "Exa AI Search" {
		t.Fatalf("unexpected provider name: %s", p.Name())
	}
}

func TestExaProviderCredentialSpec(t *testing.T) {
	p := NewExaProvider()
	specs := p.RequiredCredentials()

	if len(specs) != 1 {
		t.Fatalf("expected 1 credential spec, got %d", len(specs))
	}

	spec := specs[0]
	if spec.Key != "EXA_API_KEY" {
		t.Fatalf("unexpected spec key: %s", spec.Key)
	}
	if !spec.Secret {
		t.Fatal("expected spec to be marked as secret")
	}
	if !spec.MultiValue {
		t.Fatal("expected spec to be marked as multi-value")
	}

	// Test validator
	err := spec.Validator("11111111-1111-1111-1111-111111111111")
	if err != nil {
		t.Fatalf("expected valid key to pass validation, got error: %v", err)
	}

	err = spec.Validator("invalid")
	if err == nil {
		t.Fatal("expected invalid key to fail validation")
	}
}
