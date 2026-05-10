package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nawodyaishan/universal-mcp-sync/pkg/config"
)

func TestLoadInitialKeys(t *testing.T) {
	// Test CSV
	keys, raw, err := loadInitialKeys("11111111-1111-1111-1111-111111111111,22222222-2222-2222-2222-222222222222", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 2 || raw != "11111111-1111-1111-1111-111111111111,22222222-2222-2222-2222-222222222222" {
		t.Errorf("unexpected results for CSV: %v, %s", keys, raw)
	}

	// Test File
	dir := t.TempDir()
	path := filepath.Join(dir, "keys.txt")
	content := "11111111-1111-1111-1111-111111111111"
	_ = os.WriteFile(path, []byte(content), 0600)
	keys, raw, err = loadInitialKeys("", path)
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 1 || keys[0] != content || raw != content {
		t.Errorf("unexpected results for file: %v, %s", keys, raw)
	}

	// Test empty
	keys, raw, err = loadInitialKeys("", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 0 || raw != "" {
		t.Errorf("expected empty results, got %v, %s", keys, raw)
	}
}

func TestMapAllSelected(t *testing.T) {
	apps := []config.AppConfig{
		{ID: config.AppCursor},
		{ID: config.AppVSCode},
	}
	selected := mapAllSelected(apps)
	if len(selected) != 2 || !selected[config.AppCursor] || !selected[config.AppVSCode] {
		t.Errorf("expected all apps to be selected, got %v", selected)
	}
}
