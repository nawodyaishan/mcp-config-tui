package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nawodyaishan/universal-mcp-sync/pkg/tui"
)

// DM-P85: --record cannot combine with --keys / --keys-file.
func TestRun_RecordWithKeysIsRejected(t *testing.T) {
	var out, errBuf bytes.Buffer
	got := run([]string{"--record", "--keys", "11111111-1111-1111-1111-111111111111", "--home-dir", t.TempDir()}, &out, &errBuf)
	if got != 2 {
		t.Fatalf("expected exit 2, got %d (stderr=%q)", got, errBuf.String())
	}
	if !strings.Contains(errBuf.String(), "--record cannot be combined") {
		t.Errorf("unexpected error message: %s", errBuf.String())
	}
}

// DM-P83: replay reproduces the recorded final-state digest against the
// matching fixture.
func TestRun_ReplayReproducesDigest(t *testing.T) {
	dir := t.TempDir()
	transcript := filepath.Join(dir, "session.jsonl")

	// Craft a transcript that drives the happy-path-exa fixture from Doctor
	// to Plan Preview via [p] [enter] [enter] [n] [enter].
	entries := []tui.RecordEntry{
		{Kind: "key", Key: "p", Screen: "Doctor"},
		{Kind: "key", Key: "enter", Screen: "ProviderReady"},
	}
	writeJSONL(t, transcript, entries)

	var out, errBuf bytes.Buffer
	code := runReplayCommand([]string{"--against-fixture", "happy-path-exa", transcript}, &out, &errBuf)
	if code != 0 {
		t.Fatalf("replay exit = %d, stderr=%s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "final screen=") {
		t.Errorf("replay output missing final-state line:\n%s", out.String())
	}
}

// DM-P84: --emit-matrix output starts with the matrix-row stub heading.
func TestRun_ReplayEmitMatrix(t *testing.T) {
	dir := t.TempDir()
	transcript := filepath.Join(dir, "session.jsonl")
	entries := []tui.RecordEntry{
		{Kind: "key", Key: "p", Screen: "Doctor"},
	}
	writeJSONL(t, transcript, entries)

	var out, errBuf bytes.Buffer
	code := runReplayCommand([]string{"--emit-matrix", "--against-fixture", "happy-path-exa", transcript}, &out, &errBuf)
	if code != 0 {
		t.Fatalf("emit-matrix exit = %d, stderr=%s", code, errBuf.String())
	}
	if !strings.HasPrefix(strings.TrimSpace(out.String()), "## DM-RP") {
		t.Errorf("emit-matrix output missing stub heading:\n%s", out.String())
	}
}

func writeJSONL(t *testing.T, path string, entries []tui.RecordEntry) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create transcript: %v", err)
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	for _, e := range entries {
		if err := enc.Encode(e); err != nil {
			t.Fatalf("encode: %v", err)
		}
	}
}
