package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nawodyaishan/universal-mcp-sync/pkg/config"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/exa"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/provider"
)

func TestBuildSavedPlanDoesNotContainRawKeys(t *testing.T) {
	homeDir := t.TempDir()
	targetPath := filepath.Join(homeDir, ".cursor", "mcp.json")
	mustWriteFile(t, targetPath, []byte("{}"))

	manager, err := NewManager(homeDir, fixedNow(), fakeRunner{})
	if err != nil {
		t.Fatalf("NewManager returned error: %v", err)
	}

	key := "11111111-1111-1111-1111-111111111111"
	plan, err := manager.Prepare([]string{key}, map[config.AppID]bool{config.AppCursor: true}, DefaultAssignments(map[config.AppID]bool{config.AppCursor: true}, 1))
	if err != nil {
		t.Fatalf("Prepare returned error: %v", err)
	}

	saved, err := manager.BuildSavedPlan(plan, SavedPlanOptions{
		PlanID:       "deadbeefcafebabe",
		CreatedAt:    fixedNow()(),
		UsyncVersion: "dev",
		ProviderID:   "exa",
		Credentials: []CredentialRef{{
			Key:    "EXA_API_KEY",
			Label:  "1111...1111",
			EnvVar: "EXA_API_KEY",
		}},
	})
	if err != nil {
		t.Fatalf("BuildSavedPlan returned error: %v", err)
	}

	data, err := MarshalSavedPlanJSON(saved)
	if err != nil {
		t.Fatalf("MarshalSavedPlanJSON returned error: %v", err)
	}
	if strings.Contains(string(data), key) {
		t.Fatalf("saved plan json leaked raw key:\n%s", string(data))
	}
}

func TestBuildSavedPlanTracksCLICommandRedacted(t *testing.T) {
	homeDir := t.TempDir()
	manager, err := NewManager(homeDir, fixedNow(), fakeRunner{available: map[string]bool{"claude": true}})
	if err != nil {
		t.Fatalf("NewManager returned error: %v", err)
	}

	key := "11111111-1111-1111-1111-111111111111"
	urlValue, err := exa.BuildURL(key, exa.DefaultTools)
	if err != nil {
		t.Fatalf("BuildURL returned error: %v", err)
	}
	plan := ExecutionPlan{
		Operations: []Operation{
			{
				AppID:           config.AppClaudeCode,
				AppName:         "Claude Code",
				FileLabel:       "Claude Code CLI",
				Kind:            config.FileKindClaudeCodeCLI,
				CredentialLabel: "1111...1111",
				ProviderID:      "exa",
				Config:          provider.MCPConfig{Type: provider.TransportHTTP, URL: urlValue},
				CLIAddArgs:      []string{"mcp", "add", "--transport", "http", "-s", "user", "exa", urlValue},
			},
		},
	}

	saved, err := manager.BuildSavedPlan(plan, SavedPlanOptions{
		PlanID:       "facefeedcafefeed",
		CreatedAt:    fixedNow()(),
		UsyncVersion: "dev",
		ProviderID:   "exa",
	})
	if err != nil {
		t.Fatalf("BuildSavedPlan returned error: %v", err)
	}
	if len(saved.Operations) != 1 {
		t.Fatalf("expected 1 operation, got %d", len(saved.Operations))
	}
	if strings.Contains(strings.Join(saved.Operations[0].CLICommand, " "), key) {
		t.Fatalf("CLICommand leaked raw key: %v", saved.Operations[0].CLICommand)
	}
}

func TestPlanStoreSaveLoadListAndClean(t *testing.T) {
	homeDir := t.TempDir()
	cacheDir := filepath.Join(homeDir, ".cache-root")
	t.Setenv("XDG_CACHE_HOME", cacheDir)
	t.Setenv("USYNC_PLAN_DIR", "")

	store, err := NewPlanStore(homeDir)
	if err != nil {
		t.Fatalf("NewPlanStore returned error: %v", err)
	}
	now := time.Date(2026, time.May, 22, 11, 0, 0, 0, time.UTC)
	store.Now = func() time.Time { return now }

	saved := SavedPlan{
		SchemaVersion: SavedPlanSchemaVersion,
		PlanID:        "12345678deadbeef",
		CreatedAt:     now,
		ExpiresAt:     now.Add(time.Hour),
		UsyncVersion:  "dev",
		ProviderID:    "exa",
		Operations: []PlanOperation{{
			TargetID:   "cursor",
			TargetName: "Cursor",
			Action:     PlanActionCreate,
			FilePath:   filepath.Join(homeDir, ".cursor", "mcp.json"),
			CurrentSHA: PlanCurrentSHAMissing,
			Transport:  "http",
			Manager:    PlanManagerFile,
			Redacted:   "Cursor: create exa [http, credential=1111...1111]",
		}},
	}

	path, err := store.Save(saved, "")
	if err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	if filepath.Dir(path) != store.PlanDir {
		t.Fatalf("expected plan in %s, got %s", store.PlanDir, path)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat saved plan: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("expected 0600 plan perms, got %#o", info.Mode().Perm())
	}
	dirInfo, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatalf("stat plan dir: %v", err)
	}
	if dirInfo.Mode().Perm() != 0o700 {
		t.Fatalf("expected 0700 plan dir perms, got %#o", dirInfo.Mode().Perm())
	}

	loaded, err := store.Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if loaded.PlanID != saved.PlanID {
		t.Fatalf("unexpected plan id: got %s want %s", loaded.PlanID, saved.PlanID)
	}

	listed, err := store.List()
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(listed) != 1 || listed[0].Path != path {
		t.Fatalf("unexpected listed plans: %#v", listed)
	}

	removed, err := store.Clean(CleanOptions{RemoveAll: true})
	if err != nil {
		t.Fatalf("Clean returned error: %v", err)
	}
	if len(removed) != 1 || removed[0] != path {
		t.Fatalf("unexpected removed plans: %#v", removed)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected plan to be removed, stat err=%v", err)
	}
}

func TestFormatSavedPlanNotesNoWritesAndExpiry(t *testing.T) {
	now := time.Date(2026, time.May, 22, 12, 0, 0, 0, time.UTC)
	plan := SavedPlan{
		PlanID:        "planid",
		ProviderID:    "exa",
		CreatedAt:     now.Add(-2 * time.Hour),
		ExpiresAt:     now.Add(-time.Hour),
		Operations:    []PlanOperation{{TargetName: "Cursor", Action: PlanActionCreate, Redacted: "Cursor: create exa [http, credential=1111...1111]"}},
		SchemaVersion: SavedPlanSchemaVersion,
	}

	formatted := FormatSavedPlan(plan, now)
	if !strings.Contains(formatted, "warning: plan is expired") {
		t.Fatalf("expected expiry warning, got:\n%s", formatted)
	}
	if !strings.Contains(formatted, "No config files were written.") {
		t.Fatalf("expected no-write note, got:\n%s", formatted)
	}
}
