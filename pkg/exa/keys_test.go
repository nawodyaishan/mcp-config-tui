package exa

import (
	"strings"
	"testing"
)

func TestParseKeysAcceptsRawAndLabelledInput(t *testing.T) {
	input := strings.Join([]string{
		"11111111-1111-1111-1111-111111111111",
		`key1 = "22222222-2222-2222-2222-222222222222"`,
		`key2="11111111-1111-1111-1111-111111111111"`,
	}, "\n")

	keys, err := ParseKeys(input)
	if err != nil {
		t.Fatalf("ParseKeys returned error: %v", err)
	}

	if len(keys) != 2 {
		t.Fatalf("expected 2 unique keys, got %d", len(keys))
	}
}

func TestParseKeysRejectsMissingKeys(t *testing.T) {
	if _, err := ParseKeys("key = missing"); err == nil {
		t.Fatal("expected ParseKeys to reject missing UUID-style keys")
	}
}

func TestRedactKey(t *testing.T) {
	got := RedactKey("12345678-1234-1234-1234-123456789abc")
	if got != "1234...9abc" {
		t.Fatalf("unexpected redacted key: %s", got)
	}
}
