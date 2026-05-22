package validate

import (
	"strings"
	"testing"
)

func TestParseKeyFileSupportsCommentsQuotesAndDuplicates(t *testing.T) {
	parsed, err := ParseKeyFile([]byte(`
# comment
EXA_API_KEY=11111111-1111-1111-1111-111111111111
CONTEXT7_API_KEY="ctx7sk-abcdef1234567890"
EXA_API_KEY=22222222-2222-2222-2222-222222222222
`))
	if err != nil {
		t.Fatalf("ParseKeyFile returned error: %v", err)
	}

	values := parsed.Values()
	if got, want := values["CONTEXT7_API_KEY"], "ctx7sk-abcdef1234567890"; got != want {
		t.Fatalf("unexpected quoted value: got %q want %q", got, want)
	}

	exaValues := parsed.ValuesForKey("EXA_API_KEY")
	if len(exaValues) != 2 {
		t.Fatalf("expected 2 EXA_API_KEY entries, got %d", len(exaValues))
	}
	if exaValues[0] != "11111111-1111-1111-1111-111111111111" || exaValues[1] != "22222222-2222-2222-2222-222222222222" {
		t.Fatalf("unexpected duplicate values: %#v", exaValues)
	}
}

func TestParseKeyFileRejectsMalformedLineWithoutLeakingValue(t *testing.T) {
	_, err := ParseKeyFile([]byte("BROKEN LINE secret-value"))
	if err == nil {
		t.Fatal("expected ParseKeyFile to fail")
	}
	if !strings.Contains(err.Error(), "line 1") {
		t.Fatalf("expected line number in error, got %v", err)
	}
	if strings.Contains(err.Error(), "secret-value") {
		t.Fatalf("parse error leaked raw value: %v", err)
	}
}
