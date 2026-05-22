package validate

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const validationCacheTTL = 24 * time.Hour

type CredentialCache struct {
	Entries map[string]CacheEntry `json:"entries"`
}

type CacheEntry struct {
	Status     Status    `json:"status"`
	Message    string    `json:"message"`
	CachedAt   time.Time `json:"cached_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	ProviderID string    `json:"provider_id"`
	Key        string    `json:"key"`
	KeyLabel   string    `json:"key_label"`
}

type Store struct {
	HomeDir string
	Path    string
	Now     func() time.Time
}

func DefaultCachePath(homeDir string) (string, error) {
	if homeDir == "" {
		return "", fmt.Errorf("missing home directory")
	}
	return filepath.Join(homeDir, ".usync", "cache", "credentials.json"), nil
}

func NewStore(homeDir string) (Store, error) {
	if homeDir == "" {
		var err error
		homeDir, err = os.UserHomeDir()
		if err != nil {
			return Store{}, fmt.Errorf("resolve home directory: %w", err)
		}
	}

	path, err := DefaultCachePath(homeDir)
	if err != nil {
		return Store{}, err
	}

	return Store{
		HomeDir: homeDir,
		Path:    path,
		Now:     time.Now,
	}, nil
}

func CacheKey(providerID, key, label string) string {
	sum := sha256.Sum256([]byte(providerID + ":" + key + ":" + label))
	return hex.EncodeToString(sum[:])
}

func (s Store) Load() (CredentialCache, error) {
	info, err := os.Stat(s.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return CredentialCache{Entries: map[string]CacheEntry{}}, nil
		}
		return CredentialCache{}, fmt.Errorf("stat credential cache %s: %w", s.Path, err)
	}
	if info.Mode().Perm() != 0o600 {
		return CredentialCache{}, fmt.Errorf("credential cache %s must have 0600 permissions", s.Path)
	}

	data, err := os.ReadFile(s.Path)
	if err != nil {
		return CredentialCache{}, fmt.Errorf("read credential cache %s: %w", s.Path, err)
	}

	var cache CredentialCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return CredentialCache{}, fmt.Errorf("parse credential cache %s: %w", s.Path, err)
	}
	if cache.Entries == nil {
		cache.Entries = map[string]CacheEntry{}
	}
	return cache, nil
}

func (s Store) Save(cache CredentialCache) error {
	if cache.Entries == nil {
		cache.Entries = map[string]CacheEntry{}
	}

	if err := os.MkdirAll(filepath.Dir(s.Path), 0o700); err != nil {
		return fmt.Errorf("create credential cache directory for %s: %w", s.Path, err)
	}

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal credential cache: %w", err)
	}
	if err := writeCacheAtomic(s.Path, append(data, '\n')); err != nil {
		return err
	}
	return nil
}

func (s Store) Get(cacheKey string) (CacheEntry, bool, error) {
	cache, err := s.Load()
	if err != nil {
		return CacheEntry{}, false, err
	}

	entry, ok := cache.Entries[cacheKey]
	if !ok {
		return CacheEntry{}, false, nil
	}

	now := time.Now()
	if s.Now != nil {
		now = s.Now()
	}
	if !entry.ExpiresAt.IsZero() && entry.ExpiresAt.Before(now) {
		return CacheEntry{}, false, nil
	}
	return entry, true, nil
}

func (s Store) Put(cacheKey string, entry CacheEntry) error {
	cache, err := s.Load()
	if err != nil {
		return err
	}
	if cache.Entries == nil {
		cache.Entries = map[string]CacheEntry{}
	}
	cache.Entries[cacheKey] = entry
	return s.Save(cache)
}

func writeCacheAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	temp, err := os.CreateTemp(dir, ".usync-credential-cache-*")
	if err != nil {
		return fmt.Errorf("create temp credential cache for %s: %w", path, err)
	}

	tempPath := temp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tempPath)
		}
	}()

	if err := temp.Chmod(0o600); err != nil {
		_ = temp.Close()
		return fmt.Errorf("chmod temp credential cache for %s: %w", path, err)
	}
	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		return fmt.Errorf("write temp credential cache for %s: %w", path, err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close temp credential cache for %s: %w", path, err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("rename temp credential cache for %s: %w", path, err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Errorf("chmod credential cache %s: %w", path, err)
	}

	cleanup = false
	return nil
}
