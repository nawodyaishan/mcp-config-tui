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
	if saved.SchemaVersion != SavedPlanSchemaVersion {
		t.Fatalf("unexpected schema version: got %d want %d", saved.SchemaVersion, SavedPlanSchemaVersion)
	}
	if len(saved.Credentials) != 1 || saved.Credentials[0].ID == "" {
		t.Fatalf("expected credential ref id to be populated: %#v", saved.Credentials)
	}
	if len(saved.Operations) != 1 {
		t.Fatalf("expected 1 saved operation, got %d", len(saved.Operations))
	}
	if saved.Operations[0].ProviderID != "exa" {
		t.Fatalf("unexpected provider id: %s", saved.Operations[0].ProviderID)
	}
	if saved.Operations[0].CredentialRef != saved.Credentials[0].ID {
		t.Fatalf("unexpected credential ref mapping: %#v", saved.Operations[0])
	}
	if saved.Operations[0].FileKind != string(config.FileKindMCPServers) {
		t.Fatalf("unexpected file kind: %s", saved.Operations[0].FileKind)
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
	if saved.Operations[0].ProviderID != "exa" {
		t.Fatalf("unexpected provider id: %s", saved.Operations[0].ProviderID)
	}
	if saved.Operations[0].FileKind != string(config.FileKindClaudeCodeCLI) {
		t.Fatalf("unexpected file kind: %s", saved.Operations[0].FileKind)
	}
	if strings.Contains(strings.Join(saved.Operations[0].CLICommand, " "), key) {
		t.Fatalf("CLICommand leaked raw key: %v", saved.Operations[0].CLICommand)
	}
}

func TestPlanStoreLoadAcceptsLegacySchemaV1ForDisplay(t *testing.T) {
	homeDir := t.TempDir()
	store, err := NewPlanStore(homeDir)
	if err != nil {
		t.Fatalf("NewPlanStore returned error: %v", err)
	}
	now := fixedNow()()
	path, err := store.Save(SavedPlan{
		SchemaVersion: 1,
		PlanID:        "legacyplan",
		CreatedAt:     now,
		ExpiresAt:     now.Add(time.Hour),
		ProviderID:    "exa",
	}, filepath.Join(homeDir, "legacy-plan.json"))
	if err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	loaded, err := store.Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if loaded.SchemaVersion != 1 {
		t.Fatalf("unexpected schema version: %d", loaded.SchemaVersion)
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

func TestFormatSavedPlanPreflightIncludesApprovals(t *testing.T) {
	now := time.Date(2026, time.May, 22, 12, 0, 0, 0, time.UTC)
	preflight := SavedPlanPreflight{
		PlanID:     "planid",
		ProviderID: "exa",
		CreatedAt:  now.Add(-2 * time.Hour),
		ExpiresAt:  now.Add(time.Hour),
		Operations: []PlanOperation{{
			TargetName: "Cursor",
			Action:     PlanActionCreate,
			FilePath:   "/tmp/cursor.json",
			Redacted:   "Cursor: create exa [http, credential=1111...1111]",
		}},
		ApprovalPrompts: []ApprovalPrompt{{
			Reason:     "create",
			TargetPath: "/tmp/cursor.json",
			Message:    "Create new config file /tmp/cursor.json",
		}},
	}

	formatted := FormatSavedPlanPreflight(preflight, now)
	if !strings.Contains(formatted, "Approvals") {
		t.Fatalf("expected approvals section, got:\n%s", formatted)
	}
	if !strings.Contains(formatted, "Create new config file /tmp/cursor.json") {
		t.Fatalf("expected approval message, got:\n%s", formatted)
	}
}

// --- Secret indirection tests ---

func TestBuildSavedPlan_VSCodeInputVariables(t *testing.T) {
	homeDir := t.TempDir()
	vscodePath := filepath.Join(homeDir, ".vscode", "mcp.json")
	mustWriteFile(t, vscodePath, []byte(`{"servers":{}}`))

	manager, err := NewManager(homeDir, fixedNow(), fakeRunner{})
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	manager.Apps, err = config.DetectAppConfigsForOS(homeDir, "darwin")
	if err != nil {
		t.Fatalf("DetectAppConfigsForOS: %v", err)
	}

	key := "11111111-1111-1111-1111-111111111111"
	selected := map[config.AppID]bool{config.AppVSCode: true}
	prov := provider.NewExaProvider()
	profiles := []provider.CredentialProfile{{
		ProviderID: "exa",
		Values:     map[string]string{"EXA_API_KEY": key},
		Label:      "1111...1111",
	}}
	plan, err := manager.PrepareProvider(prov, profiles, selected, DefaultAssignments(selected, 1))
	if err != nil {
		t.Fatalf("PrepareProvider: %v", err)
	}

	planID, _ := NewPlanID()
	saved, err := manager.BuildSavedPlan(plan, SavedPlanOptions{
		PlanID:            planID,
		CreatedAt:         fixedNow()(),
		UsyncVersion:      "test",
		ProviderID:        "exa",
		Credentials:       buildCredentialRefsFromProfiles(prov, profiles),
		UseInputVariables: true,
	})
	if err != nil {
		t.Fatalf("BuildSavedPlan: %v", err)
	}

	found := false
	for _, op := range saved.Operations {
		if op.TargetID == string(config.AppVSCode) {
			found = true
			if len(op.VSCodeInputs) == 0 {
				t.Error("expected VSCodeInputs non-empty for VS Code target")
			}
			if strings.Contains(op.Redacted, key) {
				t.Errorf("raw key must not appear in Redacted: %s", op.Redacted)
			}
			if !strings.Contains(op.Redacted, "${input:") {
				t.Errorf("expected ${input:…} in Redacted, got: %s", op.Redacted)
			}
		}
	}
	if !found {
		t.Error("expected VS Code operation in plan")
	}
}

func TestBuildSavedPlan_DefaultFlagsNoChange(t *testing.T) {
	homeDir := t.TempDir()
	vscodePath := filepath.Join(homeDir, ".vscode", "mcp.json")
	mustWriteFile(t, vscodePath, []byte(`{"servers":{}}`))

	manager, err := NewManager(homeDir, fixedNow(), fakeRunner{})
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	manager.Apps, err = config.DetectAppConfigsForOS(homeDir, "darwin")
	if err != nil {
		t.Fatalf("DetectAppConfigsForOS: %v", err)
	}

	key := "11111111-1111-1111-1111-111111111111"
	selected := map[config.AppID]bool{config.AppVSCode: true}
	prov := provider.NewExaProvider()
	profiles := []provider.CredentialProfile{{
		ProviderID: "exa",
		Values:     map[string]string{"EXA_API_KEY": key},
		Label:      "1111...1111",
	}}
	plan, err := manager.PrepareProvider(prov, profiles, selected, DefaultAssignments(selected, 1))
	if err != nil {
		t.Fatalf("PrepareProvider: %v", err)
	}

	planID, _ := NewPlanID()
	saved, err := manager.BuildSavedPlan(plan, SavedPlanOptions{
		PlanID:       planID,
		CreatedAt:    fixedNow()(),
		UsyncVersion: "test",
		ProviderID:   "exa",
		Credentials:  buildCredentialRefsFromProfiles(prov, profiles),
		// UseInputVariables: false (default)
	})
	if err != nil {
		t.Fatalf("BuildSavedPlan: %v", err)
	}

	for _, op := range saved.Operations {
		if op.TargetID == string(config.AppVSCode) && len(op.VSCodeInputs) != 0 {
			t.Errorf("expected no VSCodeInputs with default flags, got %+v", op.VSCodeInputs)
		}
	}
}

// buildCredentialRefsFromProfiles is a test helper mirroring cmd/usync logic.
func buildCredentialRefsFromProfiles(prov provider.MCPProvider, profiles []provider.CredentialProfile) []CredentialRef {
	refs := make([]CredentialRef, 0, len(profiles))
	for _, p := range profiles {
		for _, cred := range prov.RequiredCredentials() {
			if _, ok := p.Values[cred.Key]; ok {
				refs = append(refs, CredentialRef{
					Key:   cred.Key,
					Label: p.Label,
				})
			}
		}
	}
	return refs
}
