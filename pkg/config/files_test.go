package config

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"
)

func TestBuildBackupPathFormat(t *testing.T) {
	now := time.Date(2026, time.May, 8, 21, 30, 45, 0, time.UTC)
	got := BuildBackupPath("/tmp/config.json", now)
	want := "/tmp/config.json.bak-usync-20260508-213045"
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

func TestReadFileOrEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "exists.txt")
	_ = os.WriteFile(path, []byte("content"), 0600)

	data, exists, err := ReadFileOrEmpty(path)
	if err != nil {
		t.Fatal(err)
	}
	if !exists || string(data) != "content" {
		t.Errorf("expected content, got %v, %s", exists, string(data))
	}

	dataEmpty, existsEmpty, err := ReadFileOrEmpty(filepath.Join(dir, "missing.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if existsEmpty || len(dataEmpty) != 0 {
		t.Errorf("expected empty, got %v, %s", existsEmpty, string(dataEmpty))
	}
}

func TestRollbackWrite_NoBackup(t *testing.T) {
	if err := RollbackWrite(WriteOutcome{}); err != nil {
		t.Errorf("RollbackWrite should succeed even if no backup is present, got %v", err)
	}
}

func TestEnsureParentDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a", "b", "c.txt")
	if err := EnsureParentDir(path); err != nil {
		t.Fatalf("EnsureParentDir failed: %v", err)
	}
	fi, err := os.Stat(filepath.Join(dir, "a", "b"))
	if err != nil || !fi.IsDir() {
		t.Fatal("parent directory not created")
	}
}

func TestWriteWithBackupReturnsLockedErrorWhenLockPersists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	lockPath := path + ".lock"
	if err := os.WriteFile(lockPath, []byte("locked"), 0o600); err != nil {
		t.Fatalf("create lock file: %v", err)
	}

	_, err := WriteWithBackup(path, []byte("{\"after\":true}\n"), time.Now())
	if err == nil {
		t.Fatal("expected WriteWithBackup to fail while lock persists")
	}
	if !errors.Is(err, ErrFileLocked) {
		t.Fatalf("expected ErrFileLocked, got %v", err)
	}
	if _, statErr := os.Stat(lockPath); statErr != nil {
		t.Fatalf("expected lock file to remain for persistent-lock case: %v", statErr)
	}
}

func TestWriteWithBackupRemovesLockAfterWriteError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte("{\"before\":true}\n"), 0o600); err != nil {
		t.Fatalf("write original file: %v", err)
	}

	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatalf("chmod dir read-only: %v", err)
	}
	defer func() {
		_ = os.Chmod(dir, 0o700)
	}()

	_, err := WriteWithBackup(path, []byte("{\"after\":true}\n"), time.Now())
	if err == nil {
		t.Fatal("expected WriteWithBackup to fail when parent directory is not writable")
	}
	if _, statErr := os.Stat(path + ".lock"); !os.IsNotExist(statErr) {
		t.Fatalf("expected lock file to be removed after failure, stat err = %v", statErr)
	}
}

func TestWriteWithBackupConcurrentWritersDoNotCorruptFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	payloads := [][]byte{
		[]byte("{\"writer\":1,\"payload\":\"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\"}\n"),
		[]byte("{\"writer\":2,\"payload\":\"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb\"}\n"),
		[]byte("{\"writer\":3,\"payload\":\"cccccccccccccccccccccccccccccccc\"}\n"),
		[]byte("{\"writer\":4,\"payload\":\"dddddddddddddddddddddddddddddddd\"}\n"),
	}

	var wg sync.WaitGroup
	errs := make(chan error, len(payloads))

	for _, payload := range payloads {
		payload := payload
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := WriteWithBackup(path, payload, time.Now())
			if err != nil && !errors.Is(err, ErrFileLocked) {
				errs <- err
			}
		}()
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Fatalf("unexpected write error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read final config: %v", err)
	}
	if !slices.ContainsFunc(payloads, func(payload []byte) bool {
		return string(payload) == string(data)
	}) {
		t.Fatalf("final content does not match any full payload:\n%s", string(data))
	}
	if _, statErr := os.Stat(path + ".lock"); !os.IsNotExist(statErr) {
		t.Fatalf("expected no stale lock file, stat err = %v", statErr)
	}
}
