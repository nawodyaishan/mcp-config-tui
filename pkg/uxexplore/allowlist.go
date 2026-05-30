package uxexplore

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// AllowlistEntry silences a single finding by MatrixID until a given date.
type AllowlistEntry struct {
	MatrixID  string    `json:"matrix_id"`
	Reason    string    `json:"reason"`
	ExpiresAt time.Time `json:"expires_at"`
}

// LoadAllowlist reads the allowlist YAML from disk.
func LoadAllowlist(path string) ([]AllowlistEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer func() { _ = f.Close() }()
	return ParseAllowlist(f)
}

// ParseAllowlist parses the minimal YAML-ish allowlist format. It mirrors the
// hand-rolled exclusions parser so we keep dependencies lean.
func ParseAllowlist(r io.Reader) ([]AllowlistEntry, error) {
	var entries []AllowlistEntry
	var current *AllowlistEntry
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		raw := scanner.Text()
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") || line == "allowlist:" {
			continue
		}
		if strings.HasPrefix(line, "- ") {
			if current != nil {
				entries = append(entries, *current)
			}
			current = &AllowlistEntry{}
			line = strings.TrimSpace(strings.TrimPrefix(line, "- "))
			if line == "" {
				continue
			}
		}
		if current == nil {
			return nil, fmt.Errorf("allowlist field before list item: %q", line)
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			return nil, fmt.Errorf("invalid allowlist line %q", line)
		}
		value = strings.Trim(strings.TrimSpace(value), `"`)
		switch strings.TrimSpace(key) {
		case "matrix_id":
			current.MatrixID = value
		case "reason":
			current.Reason = value
		case "expires_at":
			t, err := time.Parse("2006-01-02", value)
			if err != nil {
				return nil, fmt.Errorf("invalid expires_at %q: %w", value, err)
			}
			current.ExpiresAt = t
		default:
			return nil, fmt.Errorf("unknown allowlist field %q", key)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if current != nil {
		entries = append(entries, *current)
	}
	for _, entry := range entries {
		if entry.MatrixID == "" || entry.Reason == "" || entry.ExpiresAt.IsZero() {
			return nil, fmt.Errorf("incomplete allowlist entry: %#v", entry)
		}
	}
	return entries, nil
}

// FilterFindings returns the findings not covered by an unexpired allowlist
// entry. Expired entries are returned in expired so the caller can surface
// them as a separate failure.
func FilterFindings(findings []Finding, allowlist []AllowlistEntry, now time.Time) (remaining []Finding, expired []AllowlistEntry) {
	allowed := make(map[string]AllowlistEntry, len(allowlist))
	for _, entry := range allowlist {
		if !entry.ExpiresAt.After(now) {
			expired = append(expired, entry)
			continue
		}
		allowed[entry.MatrixID] = entry
	}
	for _, f := range findings {
		if _, ok := allowed[f.MatrixID]; ok {
			continue
		}
		remaining = append(remaining, f)
	}
	return remaining, expired
}
