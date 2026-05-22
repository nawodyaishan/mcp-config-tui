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
