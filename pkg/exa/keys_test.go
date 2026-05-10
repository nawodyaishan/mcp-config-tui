package exa

import (
	"os"
	"path/filepath"
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

	short := RedactKey("short")
	if short != "short" {
		t.Errorf("expected short key to be unchanged, got %s", short)
	}
}

func TestParseKeysCSV(t *testing.T) {
	input := "11111111-1111-1111-1111-111111111111, 22222222-2222-2222-2222-222222222222"
	keys, err := ParseKeysCSV(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 2 {
		t.Errorf("expected 2 keys, got %d", len(keys))
	}
}

func TestParseKeysFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "keys.txt")
	key := "11111111-1111-1111-1111-111111111111"
	_ = os.WriteFile(path, []byte(key), 0600)

	keys, err := ParseKeysFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 1 || keys[0] != key {
		t.Errorf("expected key %s, got %v", key, keys)
	}

	if _, err := ParseKeysFile("nonexistent"); err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestRedactText(t *testing.T) {
	input := "api key 11111111-1111-1111-1111-111111111111 and url https://mcp.exa.ai/mcp?exaApiKey=22222222-2222-2222-2222-222222222222"
	got := RedactText(input)
	if strings.Contains(got, "11111111") && strings.Contains(got, "11111111-1111") {
		t.Errorf("text still contains full UUID: %s", got)
	}
	if strings.Contains(got, "22222222") {
		t.Errorf("text still contains full API key in URL: %s", got)
	}
}
