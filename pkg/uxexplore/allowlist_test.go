package uxexplore

import (
	"strings"
	"testing"
	"time"
)

func TestParseAllowlist_Roundtrip(t *testing.T) {
	yaml := `# header
allowlist:
  - matrix_id: DM-Pdeadbeef
    reason: "test"
    expires_at: 2030-01-01
  - matrix_id: DM-Pcafef00d
    reason: "two"
    expires_at: 2029-06-15
`
	entries, err := ParseAllowlist(strings.NewReader(yaml))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("want 2 entries, got %d", len(entries))
	}
	if entries[0].MatrixID != "DM-Pdeadbeef" {
		t.Errorf("first matrix_id = %q", entries[0].MatrixID)
	}
	if entries[1].ExpiresAt.Year() != 2029 {
		t.Errorf("second expires_at year = %d", entries[1].ExpiresAt.Year())
	}
}

func TestParseAllowlist_RejectsIncompleteEntry(t *testing.T) {
	yaml := `allowlist:
  - matrix_id: DM-Pmissing
    reason: "no expiry"
`
	if _, err := ParseAllowlist(strings.NewReader(yaml)); err == nil {
		t.Fatalf("expected error for missing expires_at")
	}
}

func TestFilterFindings_AppliesAllowlist(t *testing.T) {
	findings := []Finding{
		{MatrixID: "DM-P1"},
		{MatrixID: "DM-P2"},
	}
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	allowed := []AllowlistEntry{
		{MatrixID: "DM-P1", Reason: "x", ExpiresAt: now.AddDate(0, 1, 0)},
	}
	remaining, expired := FilterFindings(findings, allowed, now)
	if len(expired) != 0 {
		t.Fatalf("want 0 expired, got %d", len(expired))
	}
	if len(remaining) != 1 || remaining[0].MatrixID != "DM-P2" {
		t.Fatalf("expected DM-P2 remaining; got %v", remaining)
	}
}

func TestFilterFindings_ExpiredEntryFailsClosed(t *testing.T) {
	findings := []Finding{{MatrixID: "DM-P1"}}
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	allowed := []AllowlistEntry{
		{MatrixID: "DM-P1", Reason: "x", ExpiresAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	remaining, expired := FilterFindings(findings, allowed, now)
	if len(expired) != 1 {
		t.Fatalf("want 1 expired, got %d", len(expired))
	}
	if len(remaining) != 1 {
		t.Fatalf("expired entry must NOT silence finding; got %d remaining", len(remaining))
	}
}
