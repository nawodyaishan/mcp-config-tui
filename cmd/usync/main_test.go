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
		_, _ = fmt.Fprintf(os.Stderr, "failed to create temp dir: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		_ = os.RemoveAll(dir)
	}()

	binaryPath = filepath.Join(dir, "usync")
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	if out, err := cmd.CombinedOutput(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to build usync: %v\n%s\n", err, out)
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

func TestRunValidateRequiresProvider(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"validate", "--keys", "11111111-1111-1111-1111-111111111111"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "validate requires --provider") {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}
}

func TestRunValidateOfflineExaJSON(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{
		"validate",
		"--provider", "exa",
		"--keys", "11111111-1111-1111-1111-111111111111",
		"--json",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"provider_id": "exa"`) {
		t.Fatalf("unexpected json output:\n%s", stdout.String())
	}
	if strings.Contains(stdout.String(), "11111111-1111-1111-1111-111111111111") {
		t.Fatalf("validation json leaked raw key:\n%s", stdout.String())
	}
}

func TestRunValidateInvalidGitHubKeyFailsWithoutLeakingToken(t *testing.T) {
	dir := t.TempDir()
	keyFile := filepath.Join(dir, "keys.env")
	rawToken := "ghp_tooshort"
	if err := os.WriteFile(keyFile, []byte("GITHUB_PERSONAL_ACCESS_TOKEN="+rawToken+"\n"), 0o600); err != nil {
		t.Fatalf("write key file: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{
		"validate",
		"--provider", "github",
		"--keys-file", keyFile,
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if strings.Contains(stdout.String(), rawToken) || strings.Contains(stderr.String(), rawToken) {
		t.Fatalf("validation output leaked raw token\nstdout=%s\nstderr=%s", stdout.String(), stderr.String())
	}
}

func TestRunDoctorJSON(t *testing.T) {
	t.Setenv("PATH", "")

	homeDir := t.TempDir()
	claudePath := filepath.Join(homeDir, "Library", "Application Support", "Claude", "claude_desktop_config.json")
	if err := os.MkdirAll(filepath.Dir(claudePath), 0o700); err != nil {
		t.Fatalf("mkdir claude dir: %v", err)
	}
	if err := os.WriteFile(claudePath, []byte("{\"mcpServers\":{\"context7\":{\"url\":\"https://context7.example/mcp\"}}}\n"), 0o600); err != nil {
		t.Fatalf("write claude config: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{
		"doctor",
		"--home-dir", homeDir,
		"--no-runtimes",
		"--json",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "\"id\": \"claude-desktop\"") {
		t.Fatalf("unexpected doctor json output:\n%s", stdout.String())
	}
	if strings.Contains(stdout.String(), "context7.example") {
		t.Fatalf("doctor json leaked config content:\n%s", stdout.String())
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

func TestRunPlanAllDetectedDetailedExitCode(t *testing.T) {
	homeDir := t.TempDir()
	cursorPath := filepath.Join(homeDir, ".cursor", "mcp.json")
	if err := os.MkdirAll(filepath.Dir(cursorPath), 0o700); err != nil {
		t.Fatalf("mkdir cursor dir: %v", err)
	}
	if err := os.WriteFile(cursorPath, []byte("{\"mcpServers\":{}}\n"), 0o600); err != nil {
		t.Fatalf("write cursor config: %v", err)
	}

	outPath := filepath.Join(homeDir, "plan.json")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{
		"plan",
		"--provider", "playwright",
		"--all-detected",
		"--detailed-exitcode",
		"--home-dir", homeDir,
		"--out", outPath,
	}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d stderr=%s", code, stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != outPath {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read saved plan: %v", err)
	}
	if !strings.Contains(string(data), `"target_id": "cursor"`) {
		t.Fatalf("saved plan missing detected cursor target:\n%s", string(data))
	}
}

func TestRunPlanAllDetectedIncludeWorkspace(t *testing.T) {
	homeDir := t.TempDir()
	workspace := t.TempDir()
	cursorPath := filepath.Join(workspace, ".cursor", "mcp.json")
	if err := os.MkdirAll(filepath.Dir(cursorPath), 0o700); err != nil {
		t.Fatalf("mkdir cursor dir: %v", err)
	}
	if err := os.WriteFile(cursorPath, []byte("{\"mcpServers\":{}}\n"), 0o600); err != nil {
		t.Fatalf("write cursor config: %v", err)
	}

	outPath := filepath.Join(homeDir, "workspace-plan.json")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{
		"plan",
		"--provider", "playwright",
		"--all-detected",
		"--include-workspace",
		"--workspace", workspace,
		"--home-dir", homeDir,
		"--out", outPath,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%s", code, stderr.String())
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read saved plan: %v", err)
	}
	if !strings.Contains(string(data), cursorPath) {
		t.Fatalf("saved plan missing workspace path:\n%s", string(data))
	}
	if !strings.Contains(string(data), `"target_scope": "project"`) {
		t.Fatalf("saved plan missing project scope:\n%s", string(data))
	}

	stdout.Reset()
	stderr.Reset()
	code = run([]string{
		"apply",
		"--plan", outPath,
		"--home-dir", homeDir,
		"--dry-run",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected dry-run exit code 0, got %d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "project-scoped config") {
		t.Fatalf("dry-run output missing project approval prompt:\n%s", stdout.String())
	}
}

func TestRunPlanAndApplySavedPlanForGitHub(t *testing.T) {
	homeDir := t.TempDir()
	keysFile := filepath.Join(homeDir, "github.env")
	rawToken := "ghp_" + strings.Repeat("a", 36)
	if err := os.WriteFile(keysFile, []byte("GITHUB_PERSONAL_ACCESS_TOKEN="+rawToken+"\n"), 0o600); err != nil {
		t.Fatalf("write keys file: %v", err)
	}

	outPath := filepath.Join(homeDir, "github-plan.json")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{
		"plan",
		"--provider", "github",
		"--targets", "cursor",
		"--keys-file", keysFile,
		"--home-dir", homeDir,
		"--out", outPath,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%s", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run([]string{
		"apply",
		"--plan", outPath,
		"--keys-file", keysFile,
		"--home-dir", homeDir,
		"--dry-run",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected apply dry-run exit code 0, got %d stderr=%s", code, stderr.String())
	}
	if strings.Contains(stdout.String(), rawToken) || strings.Contains(stderr.String(), rawToken) {
		t.Fatalf("saved-plan apply output leaked token\nstdout=%s\nstderr=%s", stdout.String(), stderr.String())
	}
}

func TestRunDoctorClientsVerbose(t *testing.T) {
	homeDir := t.TempDir()
	claudePath := filepath.Join(homeDir, "Library", "Application Support", "Claude", "claude_desktop_config.json")
	if err := os.MkdirAll(filepath.Dir(claudePath), 0o700); err != nil {
		t.Fatalf("mkdir claude dir: %v", err)
	}
	if err := os.WriteFile(claudePath, []byte("{\"mcpServers\":{\"context7\":{\"url\":\"https://context7.example/mcp\"}}}\n"), 0o600); err != nil {
		t.Fatalf("write claude config: %v", err)
	}
	codexPath := filepath.Join(homeDir, ".codex", "config.toml")
	if err := os.MkdirAll(filepath.Dir(codexPath), 0o700); err != nil {
		t.Fatalf("mkdir codex dir: %v", err)
	}
	if err := os.WriteFile(codexPath, []byte("[mcp_servers.exa]\nurl = \"https://mcp.exa.ai/mcp\"\n"), 0o600); err != nil {
		t.Fatalf("write codex config: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{
		"doctor",
		"--home-dir", homeDir,
		"--no-runtimes",
		"--clients", "codex-cli",
		"--verbose",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%s", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "Claude Desktop") {
		t.Fatalf("doctor output should be filtered to Codex CLI:\n%s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "Codex CLI") || !strings.Contains(stdout.String(), "candidate:") {
		t.Fatalf("unexpected verbose doctor output:\n%s", stdout.String())
	}
}

func TestRunProvidersJSON(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{
		"providers",
		"--provider", "exa",
		"--json",
		"--home-dir", t.TempDir(),
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"id": "exa"`) || !strings.Contains(stdout.String(), `"credential_status": "missing"`) {
		t.Fatalf("unexpected providers json output:\n%s", stdout.String())
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

func TestRunApplyRejectsMalformedExaKeyBeforeWrite(t *testing.T) {
	homeDir := t.TempDir()
	planPath := filepath.Join(homeDir, "plan.json")
	targetPath := filepath.Join(homeDir, ".cursor", "mcp.json")
	original := []byte("{}\n")
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		t.Fatalf("mkdir target parent: %v", err)
	}
	if err := os.WriteFile(targetPath, original, 0o600); err != nil {
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
		"--keys", "invalid",
		"--auto-approve",
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d stderr=%s", code, stderr.String())
	}

	data, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("read target after failed apply: %v", err)
	}
	if string(data) != string(original) {
		t.Fatalf("expected target to remain unchanged, got:\n%s", string(data))
	}
}
