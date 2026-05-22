package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nawodyaishan/universal-mcp-sync/pkg/app"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/config"
)

var binaryPath string

func TestMain(m *testing.M) {
	// Build the usync binary into a temporary directory
	dir, err := os.MkdirTemp("", "usync-e2e-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp dir: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		_ = os.RemoveAll(dir)
	}()

	binaryPath = filepath.Join(dir, "usync")
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	if out, err := cmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to build usync: %v\n%s\n", err, out)
		os.Exit(1)
	}

	// Make the binary path available to tests in other packages via env var if needed
	_ = os.Setenv("USYNC_E2E_BINARY", binaryPath)

	os.Exit(m.Run())
}

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

func TestRunPlanRequiresProvider(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"plan", "--targets", "cursor", "--keys", "11111111-1111-1111-1111-111111111111"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "plan requires --provider") {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}
}

func TestRunPlanCreatesSavedPlan(t *testing.T) {
	homeDir := t.TempDir()
	outPath := filepath.Join(homeDir, "plan.json")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{
		"plan",
		"--provider", "exa",
		"--targets", "cursor",
		"--keys", "11111111-1111-1111-1111-111111111111",
		"--home-dir", homeDir,
		"--out", outPath,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%s", code, stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != outPath {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read saved plan: %v", err)
	}
	if strings.Contains(string(data), "11111111-1111-1111-1111-111111111111") {
		t.Fatalf("saved plan leaked raw key:\n%s", string(data))
	}
}

func TestRunShowJSON(t *testing.T) {
	homeDir := t.TempDir()
	store, err := app.NewPlanStore(homeDir)
	if err != nil {
		t.Fatalf("NewPlanStore returned error: %v", err)
	}
	now := time.Date(2026, time.May, 22, 13, 0, 0, 0, time.UTC)
	store.Now = func() time.Time { return now }
	planPath, err := store.Save(app.SavedPlan{
		SchemaVersion: app.SavedPlanSchemaVersion,
		PlanID:        "feedfacecafebeef",
		CreatedAt:     now,
		ExpiresAt:     now.Add(time.Hour),
		UsyncVersion:  "dev",
		ProviderID:    "exa",
		Operations: []app.PlanOperation{{
			TargetID:   "cursor",
			TargetName: "Cursor",
			Action:     app.PlanActionCreate,
			Transport:  "http",
			Manager:    app.PlanManagerFile,
			Redacted:   "Cursor: create exa [http, credential=1111...1111]",
		}},
	}, filepath.Join(homeDir, "saved-plan.json"))
	if err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"show", planPath, "--json", "--home-dir", homeDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"plan_id": "feedfacecafebeef"`) {
		t.Fatalf("unexpected show json output:\n%s", stdout.String())
	}
}

func TestRunPlanListAndClean(t *testing.T) {
	homeDir := t.TempDir()
	store, err := app.NewPlanStore(homeDir)
	if err != nil {
		t.Fatalf("NewPlanStore returned error: %v", err)
	}
	now := time.Date(2026, time.May, 22, 14, 0, 0, 0, time.UTC)
	store.Now = func() time.Time { return now }
	planPath, err := store.Save(app.SavedPlan{
		SchemaVersion: app.SavedPlanSchemaVersion,
		PlanID:        "beadbeadbeadbead",
		CreatedAt:     now.Add(-2 * time.Hour),
		ExpiresAt:     now.Add(-time.Hour),
		UsyncVersion:  "dev",
		ProviderID:    "exa",
	}, "")
	if err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	var listOut bytes.Buffer
	var listErr bytes.Buffer
	code := run([]string{"plan", "list", "--home-dir", homeDir}, &listOut, &listErr)
	if code != 0 {
		t.Fatalf("expected list exit code 0, got %d stderr=%s", code, listErr.String())
	}
	if !strings.Contains(listOut.String(), "beadbeadbeadbead") {
		t.Fatalf("unexpected plan list output:\n%s", listOut.String())
	}

	var cleanOut bytes.Buffer
	var cleanErr bytes.Buffer
	code = run([]string{"plan", "clean", "--expired", "--home-dir", homeDir}, &cleanOut, &cleanErr)
	if code != 0 {
		t.Fatalf("expected clean exit code 0, got %d stderr=%s", code, cleanErr.String())
	}
	if strings.TrimSpace(cleanOut.String()) != planPath {
		t.Fatalf("unexpected clean output: %q", cleanOut.String())
	}
}

func TestRunApplyRequiresPlan(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"apply"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "apply requires --plan") {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}
}

func TestRunApplyDryRun(t *testing.T) {
	homeDir := t.TempDir()
	planPath := filepath.Join(homeDir, "plan.json")

	var planOut bytes.Buffer
	var planErr bytes.Buffer
	code := run([]string{
		"plan",
		"--provider", "exa",
		"--targets", "cursor",
		"--keys", "11111111-1111-1111-1111-111111111111",
		"--home-dir", homeDir,
		"--out", planPath,
	}, &planOut, &planErr)
	if code != 0 {
		t.Fatalf("plan failed: code=%d stderr=%s", code, planErr.String())
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code = run([]string{
		"apply",
		"--plan", planPath,
		"--home-dir", homeDir,
		"--keys", "11111111-1111-1111-1111-111111111111",
		"--dry-run",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Saved MCP apply preview") {
		t.Fatalf("unexpected dry-run output:\n%s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "No config files were written.") {
		t.Fatalf("expected no-write note, got:\n%s", stdout.String())
	}
}

func TestRunApplySavedPlan(t *testing.T) {
	homeDir := t.TempDir()
	planPath := filepath.Join(homeDir, "plan.json")
	targetPath := filepath.Join(homeDir, ".cursor", "mcp.json")
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		t.Fatalf("mkdir target parent: %v", err)
	}
	if err := os.WriteFile(targetPath, []byte("{}\n"), 0o600); err != nil {
		t.Fatalf("write target fixture: %v", err)
	}

	var planOut bytes.Buffer
	var planErr bytes.Buffer
	code := run([]string{
		"plan",
		"--provider", "exa",
		"--targets", "cursor",
		"--keys", "11111111-1111-1111-1111-111111111111",
		"--home-dir", homeDir,
		"--out", planPath,
	}, &planOut, &planErr)
	if code != 0 {
		t.Fatalf("plan failed: code=%d stderr=%s", code, planErr.String())
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code = run([]string{
		"apply",
		"--plan", planPath,
		"--home-dir", homeDir,
		"--keys", "11111111-1111-1111-1111-111111111111",
		"--auto-approve",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "MCP sync result") {
		t.Fatalf("unexpected apply output:\n%s", stdout.String())
	}

	data, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("read updated target: %v", err)
	}
	if !strings.Contains(string(data), "\"url\":") {
		t.Fatalf("expected updated target file, got:\n%s", string(data))
	}
}
