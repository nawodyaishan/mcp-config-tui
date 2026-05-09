package exa

import (
	"net/url"
	"strings"
	"testing"
)

func TestBuildURLIncludesAllDefaultTools(t *testing.T) {
	key := "11111111-1111-1111-1111-111111111111"

	raw, err := BuildURL(key, DefaultTools)
	if err != nil {
		t.Fatalf("BuildURL returned error: %v", err)
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse URL: %v", err)
	}

	if parsed.Query().Get("exaApiKey") != key {
		t.Fatalf("missing key in URL query: %s", raw)
	}

	gotTools := strings.Split(parsed.Query().Get("tools"), ",")
	if len(gotTools) != len(DefaultTools) {
		t.Fatalf("expected %d tools, got %d", len(DefaultTools), len(gotTools))
	}
}
