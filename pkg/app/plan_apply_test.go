package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nawodyaishan/universal-mcp-sync/pkg/config"
)

type staticApprover struct {
	allow bool
}

func (a staticApprover) Confirm(prompt ApprovalPrompt) (bool, error) {
	return a.allow, nil
}

func TestPreflightSavedPlanRejectsExpiredWithoutForceStale(t *testing.T) {
	manager, saved, key := buildSavedCursorPlan(t)
	saved.ExpiresAt = manager.Now().Add(-time.Minute)

	_, err := manager.PreflightSavedPlan(saved, SavedPlanApplyOptions{
		Credentials: savedPlanCredentialValues(saved, key),
	})
	if err == nil || !strings.Contains(err.Error(), "expired") {
		t.Fatalf("expected expired plan error, got %v", err)
	}
}

func TestPreflightSavedPlanAllowsExpiredWithForceStale(t *testing.T) {
	manager, saved, key := buildSavedCursorPlan(t)
	saved.ExpiresAt = manager.Now().Add(-time.Minute)

	preflight, err := manager.PreflightSavedPlan(saved, SavedPlanApplyOptions{
		Credentials: savedPlanCredentialValues(saved, key),
		ForceStale:  true,
	})
	if err != nil {
		t.Fatalf("PreflightSavedPlan returned error: %v", err)
	}
	if preflight.PlanID != saved.PlanID {
		t.Fatalf("unexpected preflight plan id: got %s want %s", preflight.PlanID, saved.PlanID)
	}
}

func TestPreflightSavedPlanRejectsMissingCredential(t *testing.T) {
	manager, saved, _ := buildSavedCursorPlan(t)

	_, err := manager.PreflightSavedPlan(saved, SavedPlanApplyOptions{})
	if err == nil || !strings.Contains(err.Error(), "missing credential") {
		t.Fatalf("expected missing credential error, got %v", err)
	}
}

func TestPreflightSavedPlanRejectsChecksumMismatch(t *testing.T) {
	manager, saved, key := buildSavedCursorPlan(t)
	if err := os.WriteFile(saved.Operations[0].FilePath, []byte("{\"changed\":true}\n"), 0o600); err != nil {
		t.Fatalf("write changed target: %v", err)
	}

	_, err := manager.PreflightSavedPlan(saved, SavedPlanApplyOptions{
		Credentials: savedPlanCredentialValues(saved, key),
	})
	if err == nil || !strings.Contains(err.Error(), "target changed since plan creation") {
		t.Fatalf("expected checksum mismatch error, got %v", err)
	}
}

func TestApplySavedPlanRollsBackPriorWritesOnLaterFailure(t *testing.T) {
	homeDir := t.TempDir()
	firstPath := filepath.Join(homeDir, ".gemini", "settings.json")
	secondPath := filepath.Join(homeDir, ".gemini", "mcp_config.json")
	firstOriginal := []byte("{\n  \"name\": \"first\"\n}\n")
	secondOriginal := []byte("{\n  \"name\": \"second\"\n}\n")
	mustWriteFile(t, firstPath, firstOriginal)
	mustWriteFile(t, secondPath, secondOriginal)

	manager := newDarwinQAManager(t, homeDir, fakeRunner{})
	key := "11111111-1111-1111-1111-111111111111"
	selected := map[config.AppID]bool{config.AppGeminiCLI: true}
	legacyPlan, err := manager.Prepare([]string{key}, selected, DefaultAssignments(selected, 1))
	if err != nil {
		t.Fatalf("Prepare returned error: %v", err)
	}
	saved, err := manager.BuildSavedPlan(legacyPlan, SavedPlanOptions{
		PlanID:       "saved-plan-rollback",
		CreatedAt:    manager.Now().UTC(),
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

	callCount := 0
	manager.WriteConfig = func(path string, data []byte, now time.Time) (config.WriteOutcome, error) {
		callCount++
		if callCount == 2 {
			return config.WriteOutcome{}, os.ErrPermission
		}
		return config.WriteWithBackup(path, data, now)
	}

	result, err := manager.ApplySavedPlan(saved, SavedPlanApplyOptions{
		Credentials: savedPlanCredentialValues(saved, key),
		AutoApprove: true,
	})
	if err == nil {
		t.Fatal("expected ApplySavedPlan to fail")
	}

	firstData, readErr := os.ReadFile(firstPath)
	if readErr != nil {
		t.Fatalf("read first file after rollback: %v", readErr)
	}
	if string(firstData) != string(firstOriginal) {
		t.Fatalf("expected first file to be restored, got:\n%s", string(firstData))
	}
	if len(result.RolledBack) == 0 || result.RolledBack[0] == "" {
		t.Fatalf("expected rollback to be recorded, got %#v", result.RolledBack)
	}
}

func TestApplySavedPlanWritesThroughSymlinkWithoutReplacingIt(t *testing.T) {
	homeDir := t.TempDir()
	realTarget := filepath.Join(homeDir, ".config", "shared", "cursor.json")
	symlinkPath := filepath.Join(homeDir, ".cursor", "mcp.json")
	mustWriteFile(t, realTarget, []byte("{}\n"))
	if err := os.MkdirAll(filepath.Dir(symlinkPath), 0o755); err != nil {
		t.Fatalf("mkdir symlink parent: %v", err)
	}
	if err := os.Symlink(realTarget, symlinkPath); err != nil {
		t.Fatalf("create symlink: %v", err)
	}

	manager := newDarwinQAManager(t, homeDir, fakeRunner{})
	key := "11111111-1111-1111-1111-111111111111"
	selected := map[config.AppID]bool{config.AppCursor: true}
	legacyPlan, err := manager.Prepare([]string{key}, selected, DefaultAssignments(selected, 1))
	if err != nil {
		t.Fatalf("Prepare returned error: %v", err)
	}
	saved, err := manager.BuildSavedPlan(legacyPlan, SavedPlanOptions{
		PlanID:       "saved-plan-symlink",
		CreatedAt:    manager.Now().UTC(),
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

	result, err := manager.ApplySavedPlan(saved, SavedPlanApplyOptions{
		Credentials: savedPlanCredentialValues(saved, key),
		AutoApprove: true,
	})
	if err != nil {
		t.Fatalf("ApplySavedPlan returned error: %v", err)
	}
	if len(result.UpdatedTargets) != 1 || result.UpdatedTargets[0] != symlinkPath {
		t.Fatalf("unexpected updated targets: %#v", result.UpdatedTargets)
	}

	info, err := os.Lstat(symlinkPath)
	if err != nil {
		t.Fatalf("lstat symlink path: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("expected %s to remain a symlink", symlinkPath)
	}

	data, err := os.ReadFile(realTarget)
	if err != nil {
		t.Fatalf("read resolved target: %v", err)
	}
	if !strings.Contains(string(data), "\"url\":") {
		t.Fatalf("expected resolved target to be updated, got:\n%s", string(data))
	}
}

func TestApplySavedPlanRequiresApprovalForNewFile(t *testing.T) {
	manager, saved, key := buildSavedCursorPlan(t)
	if err := os.Remove(saved.Operations[0].FilePath); err != nil {
		t.Fatalf("remove existing target: %v", err)
	}
	saved.Operations[0].CurrentSHA = PlanCurrentSHAMissing
	saved.Operations[0].WillCreate = true
	saved.Operations[0].Action = PlanActionCreate

	_, err := manager.ApplySavedPlan(saved, SavedPlanApplyOptions{
		Credentials: savedPlanCredentialValues(saved, key),
		Approver:    staticApprover{allow: false},
	})
	if err == nil || !strings.Contains(err.Error(), "apply cancelled") {
		t.Fatalf("expected approval cancellation error, got %v", err)
	}
}

func TestApplySavedPlanWritesAuditEntryWithoutRawKey(t *testing.T) {
	manager, saved, key := buildSavedCursorPlan(t)

	_, err := manager.ApplySavedPlan(saved, SavedPlanApplyOptions{
		Credentials: savedPlanCredentialValues(saved, key),
		AutoApprove: true,
	})
	if err != nil {
		t.Fatalf("ApplySavedPlan returned error: %v", err)
	}

	auditPath := filepath.Join(manager.HomeDir, ".usync", "audit.log")
	data, err := os.ReadFile(auditPath)
	if err != nil {
		t.Fatalf("read audit log: %v", err)
	}
	if !strings.Contains(string(data), saved.PlanID) {
		t.Fatalf("expected audit log to contain plan id, got:\n%s", string(data))
	}
	if strings.Contains(string(data), key) {
		t.Fatalf("audit log leaked raw key:\n%s", string(data))
	}
}

func TestApplySavedPlanWarnsWhenAuditWriteFails(t *testing.T) {
	manager, saved, key := buildSavedCursorPlan(t)
	blockerPath := filepath.Join(manager.HomeDir, ".usync")
	if err := os.WriteFile(blockerPath, []byte("blocker"), 0o600); err != nil {
		t.Fatalf("write audit blocker: %v", err)
	}

	result, err := manager.ApplySavedPlan(saved, SavedPlanApplyOptions{
		Credentials: savedPlanCredentialValues(saved, key),
		AutoApprove: true,
	})
	if err != nil {
		t.Fatalf("ApplySavedPlan returned error: %v", err)
	}

	found := false
	for _, warning := range result.Warnings {
		if strings.Contains(warning, "audit log:") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected audit warning, got %#v", result.Warnings)
	}
}

func buildSavedCursorPlan(t *testing.T) (*Manager, SavedPlan, string) {
	t.Helper()

	homeDir := t.TempDir()
	targetPath := filepath.Join(homeDir, ".cursor", "mcp.json")
	mustWriteFile(t, targetPath, []byte("{}\n"))

	manager := newDarwinQAManager(t, homeDir, fakeRunner{})
	key := "11111111-1111-1111-1111-111111111111"
	selected := map[config.AppID]bool{config.AppCursor: true}
	legacyPlan, err := manager.Prepare([]string{key}, selected, DefaultAssignments(selected, 1))
	if err != nil {
		t.Fatalf("Prepare returned error: %v", err)
	}

	saved, err := manager.BuildSavedPlan(legacyPlan, SavedPlanOptions{
		PlanID:       "saved-plan-cursor",
		CreatedAt:    manager.Now().UTC(),
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

	return manager, saved, key
}

func savedPlanCredentialValues(plan SavedPlan, key string) map[string]string {
	values := make(map[string]string)
	for _, ref := range plan.Credentials {
		ref = normalizedCredentialRef(ref)
		if ref.Key == "EXA_API_KEY" {
			values[ref.ID] = key
		}
	}
	return values
}
