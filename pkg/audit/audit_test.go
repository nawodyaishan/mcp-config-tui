package audit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWriterAppendCreatesPrivateLog(t *testing.T) {
	homeDir := t.TempDir()
	writer, err := NewWriter(homeDir)
	if err != nil {
		t.Fatalf("NewWriter returned error: %v", err)
	}

	entry := Entry{
		Timestamp: time.Date(2026, time.May, 22, 12, 0, 0, 0, time.UTC),
		Command:   "usync apply --plan",
		PlanID:    "feedfacecafebeef",
		Targets:   []string{"cursor"},
		ExitCode:  0,
	}
	if err := writer.Append(entry); err != nil {
		t.Fatalf("Append returned error: %v", err)
	}

	data, err := os.ReadFile(writer.Path)
	if err != nil {
		t.Fatalf("read audit log: %v", err)
	}
	if !strings.Contains(string(data), `"plan_id":"feedfacecafebeef"`) {
		t.Fatalf("unexpected audit log contents:\n%s", string(data))
	}

	info, err := os.Stat(writer.Path)
	if err != nil {
		t.Fatalf("stat audit log: %v", err)
	}
	if info.Mode().Perm() != privateFilePerm {
		t.Fatalf("expected %#o file perms, got %#o", privateFilePerm, info.Mode().Perm())
	}

	dirInfo, err := os.Stat(filepath.Dir(writer.Path))
	if err != nil {
		t.Fatalf("stat audit dir: %v", err)
	}
	if dirInfo.Mode().Perm() != privateDirPerm {
		t.Fatalf("expected %#o dir perms, got %#o", privateDirPerm, dirInfo.Mode().Perm())
	}
}

func TestAuditLogRotatesAt5MB(t *testing.T) {
	homeDir := t.TempDir()
	w, err := NewWriter(homeDir)
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}

	// Write entries until log exceeds 5 MB.
	large := strings.Repeat("x", 1024)
	for i := 0; i < 6000; i++ {
		_ = w.Append(Entry{
			Timestamp: time.Now(),
			Command:   large,
			ExitCode:  0,
		})
	}

	// Rotation must have occurred.
	if _, err := os.Stat(w.Path + ".1"); err != nil {
		t.Fatalf("expected audit.log.1 to exist after rotation: %v", err)
	}
	info, err := os.Stat(w.Path)
	if err != nil {
		t.Fatalf("expected audit.log to exist: %v", err)
	}
	if info.Size() >= maxAuditLogBytes {
		t.Errorf("expected audit.log to be fresh and small after rotation, got %d bytes", info.Size())
	}
}

func TestAuditLogRotationFailureDoesNotPreventAppend(t *testing.T) {
	homeDir := t.TempDir()
	w, err := NewWriter(homeDir)
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}

	// Place a directory where the .1 file would go to make rename fail.
	if err := os.MkdirAll(w.Path+".1", 0o700); err != nil {
		t.Fatalf("setup: %v", err)
	}

	// Fill log past 5 MB so maybeRotate fires.
	large := strings.Repeat("x", 1024)
	for i := 0; i < 6000; i++ {
		_ = w.Append(Entry{Timestamp: time.Now(), Command: large, ExitCode: 0})
	}

	// Append should still succeed despite rename failure.
	err = w.Append(Entry{Timestamp: time.Now(), Command: "after-rotation-failure", ExitCode: 0})
	if err != nil {
		t.Errorf("Append failed after rename failure: %v", err)
	}
}
