package validate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestStoreSaveLoadAndPermissions(t *testing.T) {
	homeDir := t.TempDir()
	store, err := NewStore(homeDir)
	if err != nil {
		t.Fatalf("NewStore returned error: %v", err)
	}
	now := time.Date(2026, time.May, 22, 10, 0, 0, 0, time.UTC)
	store.Now = func() time.Time { return now }

	rawKey := "ghp_" + strings.Repeat("a", 36)
	entry := CacheEntry{
		Status:     StatusOK,
		Message:    "live validation succeeded",
		CachedAt:   now,
		ExpiresAt:  now.Add(validationCacheTTL),
		ProviderID: "github",
		Key:        "GITHUB_PERSONAL_ACCESS_TOKEN",
		KeyLabel:   "ghp_aaaa...aaaa",
	}
	if err := store.Put(CacheKey("github", entry.Key, entry.KeyLabel), entry); err != nil {
		t.Fatalf("Put returned error: %v", err)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(loaded.Entries) != 1 {
		t.Fatalf("expected 1 cache entry, got %d", len(loaded.Entries))
	}

	data, err := os.ReadFile(store.Path)
	if err != nil {
		t.Fatalf("read cache file: %v", err)
	}
	if strings.Contains(string(data), rawKey) {
		t.Fatalf("cache file leaked raw key:\n%s", string(data))
	}

	info, err := os.Stat(store.Path)
	if err != nil {
		t.Fatalf("stat cache file: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("expected 0600 cache perms, got %#o", info.Mode().Perm())
	}

	dirInfo, err := os.Stat(filepath.Dir(store.Path))
	if err != nil {
		t.Fatalf("stat cache dir: %v", err)
	}
	if dirInfo.Mode().Perm() != 0o700 {
		t.Fatalf("expected 0700 cache dir perms, got %#o", dirInfo.Mode().Perm())
	}
}

func TestStoreGetReturnsFalseForExpiredEntry(t *testing.T) {
	homeDir := t.TempDir()
	store, err := NewStore(homeDir)
	if err != nil {
		t.Fatalf("NewStore returned error: %v", err)
	}
	now := time.Date(2026, time.May, 22, 10, 0, 0, 0, time.UTC)
	store.Now = func() time.Time { return now }

	cacheKey := CacheKey("github", "GITHUB_PERSONAL_ACCESS_TOKEN", "ghp_aaaa...aaaa")
	if err := store.Put(cacheKey, CacheEntry{
		Status:     StatusOK,
		Message:    "live validation succeeded",
		CachedAt:   now.Add(-48 * time.Hour),
		ExpiresAt:  now.Add(-24 * time.Hour),
		ProviderID: "github",
		Key:        "GITHUB_PERSONAL_ACCESS_TOKEN",
		KeyLabel:   "ghp_aaaa...aaaa",
	}); err != nil {
		t.Fatalf("Put returned error: %v", err)
	}

	_, ok, err := store.Get(cacheKey)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if ok {
		t.Fatal("expected expired cache entry to be ignored")
	}
}

func TestStoreLoadRejectsCorruptCache(t *testing.T) {
	homeDir := t.TempDir()
	store, err := NewStore(homeDir)
	if err != nil {
		t.Fatalf("NewStore returned error: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(store.Path), 0o700); err != nil {
		t.Fatalf("mkdir cache dir: %v", err)
	}
	if err := os.WriteFile(store.Path, []byte("{not-json"), 0o600); err != nil {
		t.Fatalf("write corrupt cache: %v", err)
	}

	_, err = store.Load()
	if err == nil {
		t.Fatal("expected corrupt cache to fail")
	}
	if strings.Contains(err.Error(), "ghp_") || strings.Contains(err.Error(), "tvly-") {
		t.Fatalf("corrupt cache error leaked credential-like content: %v", err)
	}
}
