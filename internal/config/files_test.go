package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBuildBackupPathFormat(t *testing.T) {
	now := time.Date(2026, time.May, 8, 21, 30, 45, 0, time.UTC)
	got := BuildBackupPath("/tmp/config.json", now)
	want := "/tmp/config.json.bak-exa-20260508-213045"
	if got != want {
		t.Fatalf("unexpected backup path: got %s want %s", got, want)
	}
}

func TestWriteWithBackupUsesPrivatePermissionsAndRollbackRestoresOriginal(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "config.json")
	original := []byte("{\"before\":true}\n")
	updated := []byte("{\"after\":true}\n")

	if err := os.WriteFile(path, original, 0o644); err != nil {
		t.Fatalf("write original file: %v", err)
	}

	now := time.Date(2026, time.May, 8, 21, 30, 45, 0, time.UTC)
	outcome, err := WriteWithBackup(path, updated, now)
	if err != nil {
		t.Fatalf("WriteWithBackup returned error: %v", err)
	}

	perm, err := FilePerm(path)
	if err != nil {
		t.Fatalf("stat updated file: %v", err)
	}
	if perm != 0o600 {
		t.Fatalf("expected updated file perm 0600, got %#o", perm)
	}

	backupPerm, err := FilePerm(outcome.BackupPath)
	if err != nil {
		t.Fatalf("stat backup file: %v", err)
	}
	if backupPerm != 0o600 {
		t.Fatalf("expected backup perm 0600, got %#o", backupPerm)
	}

	if err := RollbackWrite(outcome); err != nil {
		t.Fatalf("RollbackWrite returned error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read restored file: %v", err)
	}
	if string(data) != string(original) {
		t.Fatalf("expected original content after rollback, got %s", string(data))
	}
}
