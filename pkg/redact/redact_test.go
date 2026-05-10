package redact_test

import (
    "testing"
    "github.com/nawodyaishan/universal-mcp-sync/pkg/redact"
)

func TestKey(t *testing.T) {
    tests := []struct{ in, want string }{
        {"11111111-1111-1111-1111-111111111111", "1111...1111"},
        {"abcd", "abcd"},                          // too short — unchanged
        {"12345678", "12345678"},                   // exactly 8 — unchanged
        {"123456789", "1234...6789"},               // 9 chars — truncated
    }
    for _, tt := range tests {
        if got := redact.Key(tt.in); got != tt.want {
            t.Errorf("Key(%q) = %q, want %q", tt.in, got, tt.want)
        }
    }
}

func TestText(t *testing.T) {
    uuid := "11111111-1111-1111-1111-111111111111"
    input := "error: key " + uuid + " rejected"
    got := redact.Text(input)
    if got == input {
        t.Fatal("Text should redact UUID-shaped substrings")
    }
    if contains(got, uuid) {
        t.Fatalf("Text output still contains full UUID: %s", got)
    }
    // non-UUID text preserved
    if !contains(got, "error: key") || !contains(got, "rejected") {
        t.Errorf("Text should preserve non-UUID parts: %s", got)
    }
}

func TestTextNoUUID(t *testing.T) {
    in := "plain text without secrets"
    if got := redact.Text(in); got != in {
        t.Errorf("Text should not modify strings without UUIDs")
    }
}

func TestAttrs(t *testing.T) {
    uuid := "11111111-1111-1111-1111-111111111111"
    attrs := []any{"error", uuid, "count", 42}
    got := redact.Attrs(attrs)
    if len(got) != 4 {
        t.Fatalf("Attrs should preserve length")
    }
    if s, ok := got[1].(string); !ok || s == uuid {
        t.Errorf("Attrs should redact UUID string values")
    }
    if n, ok := got[3].(int); !ok || n != 42 {
        t.Errorf("Attrs should pass non-string values through unchanged")
    }
}

func contains(s, sub string) bool {
    return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
        func() bool {
            for i := 0; i+len(sub) <= len(s); i++ {
                if s[i:i+len(sub)] == sub { return true }
            }
            return false
        }())
}
