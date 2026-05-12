package e2e_test

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/nawodyaishan/universal-mcp-sync/pkg/app"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/config"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/provider"
)

func isUpdate() bool {
	f := flag.Lookup("update")
	return f != nil && f.Value.String() == "true"
}

var binaryPath string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "usync-e2e-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp dir: %v\n", err)
		os.Exit(1)
	}

	binaryPath = filepath.Join(dir, "usync")
	cmd := exec.Command("go", "build", "-o", binaryPath, "../../cmd/usync")
	if out, err := cmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to build usync: %v\n%s\n", err, out)
		os.RemoveAll(dir)
		os.Exit(1)
	}

	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

func runBinary(t *testing.T, args []string, homeDir string) ([]byte, error) {
	t.Helper()
	cmd := exec.Command(binaryPath, args...)
	// Set a restricted PATH so real system binaries like `claude` or `docker`
	// aren't executed, ensuring determinism across different environments.
	cmd.Env = append(os.Environ(), "HOME="+homeDir, "PATH=/usr/bin:/bin")
	return cmd.CombinedOutput()
}

func runBinaryWithError(t *testing.T, args []string, homeDir string) ([]byte, error) {
	t.Helper()
	cmd := exec.Command(binaryPath, args...)
	cmd.Env = append(os.Environ(), "HOME="+homeDir, "PATH=/usr/bin:/bin")
	return cmd.CombinedOutput()
}

func scrubPath(content []byte, pathToScrub string) []byte {
	return bytes.ReplaceAll(content, []byte(pathToScrub), []byte("{{HOME}}"))
}

func assertGolden(t *testing.T, actual []byte, goldenFile string) {
	t.Helper()

	if isUpdate() {
		if err := os.MkdirAll(filepath.Dir(goldenFile), 0755); err != nil {
			t.Fatalf("failed to create golden file directory: %v", err)
		}
		err := os.WriteFile(goldenFile, actual, 0644)
		if err != nil {
			t.Fatalf("failed to update golden file %s: %v", goldenFile, err)
		}
	}

	expected, err := os.ReadFile(goldenFile)
	if err != nil {
		if os.IsNotExist(err) {
			t.Fatalf("golden file %s does not exist. run `go test ./tests/e2e -update`", goldenFile)
		}
		t.Fatalf("failed to read golden file %s: %v", goldenFile, err)
	}

	if !bytes.Equal(expected, actual) {
		t.Fatalf("output does not match golden file %s\nExpected:\n%s\nActual:\n%s", goldenFile, string(expected), string(actual))
	}
}

func getFilesToScaffold() []string {
	return []string{
		filepath.Join("Library", "Application Support", "Claude", "claude_desktop_config.json"),
		".claude.json",
		filepath.Join(".cursor", "mcp.json"),
		filepath.Join(".vscode", "mcp.json"),
		filepath.Join(".codeium", "windsurf", "mcp_config.json"),
		filepath.Join(".config", "zed", "settings.json"),
		filepath.Join("Library", "Application Support", "Code", "User", "globalStorage", "saoudrizwan.claude-dev", "settings", "mcp_settings.json"),
		".opencode.json",
		filepath.Join(".kiro", "settings", "mcp.json"),
		filepath.Join(".gemini", "settings.json"),
		filepath.Join(".gemini", "antigravity", "mcp_config.json"),
		filepath.Join(".codex", "config.toml"),
	}
}

func scaffoldHome(t *testing.T, homeDir string) {
	t.Helper()
	for _, file := range getFilesToScaffold() {
		fullPath := filepath.Join(homeDir, file)
		err := os.MkdirAll(filepath.Dir(fullPath), 0755)
		if err != nil {
			t.Fatalf("failed to scaffold dir %s: %v", filepath.Dir(fullPath), err)
		}
		err = os.WriteFile(fullPath, []byte("{}"), 0644)
		if err != nil {
			t.Fatalf("failed to scaffold file %s: %v", fullPath, err)
		}
	}
	_ = os.WriteFile(filepath.Join(homeDir, ".codex", "config.toml"), []byte(""), 0644)
}

func validateGoldenFiles(t *testing.T, tcName, homeDir string) {
	t.Helper()
	for _, file := range getFilesToScaffold() {
		fullPath := filepath.Join(homeDir, file)
		actual, err := os.ReadFile(fullPath)
		if err != nil {
			t.Fatalf("failed to read generated file %s: %v", fullPath, err)
		}

		actualScrubbed := scrubPath(actual, homeDir)

		goldenName := filepath.Base(file)
		if filepath.Base(filepath.Dir(file)) != "." {
			goldenName = filepath.Base(filepath.Dir(file)) + "_" + goldenName
		}
		goldenFile := filepath.Join("testdata", tcName, goldenName+".golden")

		assertGolden(t, actualScrubbed, goldenFile)
	}
}

func TestCLI_ExaProvider(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{
			name: "cli_exa_default",
			args: []string{"--apply", "--keys", "11111111-1111-1111-1111-111111111111,22222222-2222-2222-2222-222222222222,33333333-3333-3333-3333-333333333333"},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			homeDir := t.TempDir()
			scaffoldHome(t, homeDir)

			output, err := runBinary(t, tc.args, homeDir)
			if err != nil {
				t.Fatalf("usync failed: %v\nOutput: %s", err, string(output))
			}

			validateGoldenFiles(t, tc.name, homeDir)
		})
	}
}

func TestEdgeCase_Idempotency(t *testing.T) {
	t.Parallel()
	homeDir := t.TempDir()
	scaffoldHome(t, homeDir)

	args := []string{"--apply", "--keys", "11111111-1111-1111-1111-111111111111"}
	
	// First run
	if out, err := runBinary(t, args, homeDir); err != nil {
		t.Fatalf("first run failed: %v\nOutput: %s", err, string(out))
	}

	// Second run
	if out, err := runBinary(t, args, homeDir); err != nil {
		t.Fatalf("second run failed: %v\nOutput: %s", err, string(out))
	}

	// Validate against a new golden file to ensure no duplicate keys
	validateGoldenFiles(t, "edge_case_idempotency", homeDir)
}

func TestEdgeCase_Merging(t *testing.T) {
	t.Parallel()
	homeDir := t.TempDir()
	scaffoldHome(t, homeDir)

	// Inject an existing manual server into Claude Desktop
	claudePath := filepath.Join(homeDir, "Library", "Application Support", "Claude", "claude_desktop_config.json")
	existingConfig := `{
		"mcpServers": {
			"manual-db": {
				"command": "node",
				"args": ["postgres-server.js"]
			}
		}
	}`
	if err := os.WriteFile(claudePath, []byte(existingConfig), 0644); err != nil {
		t.Fatalf("failed to seed claude config: %v", err)
	}

	args := []string{"--apply", "--keys", "11111111-1111-1111-1111-111111111111"}
	if out, err := runBinary(t, args, homeDir); err != nil {
		t.Fatalf("run failed: %v\nOutput: %s", err, string(out))
	}

	validateGoldenFiles(t, "edge_case_merging", homeDir)
}

type fakeRunner struct{}
func (f fakeRunner) LookPath(name string) (string, error) { return "/usr/bin/" + name, nil }
func (f fakeRunner) Run(name string, args ...string) (string, error) { return "", nil }

func TestProviders_Golden(t *testing.T) {
	registry := provider.DefaultRegistry()
	providers := registry.All()

	for _, prov := range providers {
		prov := prov
		t.Run(prov.ID(), func(t *testing.T) {
			t.Parallel()
			homeDir := t.TempDir()
			scaffoldHome(t, homeDir)

			manager, err := app.NewManager(homeDir, func() time.Time { return time.Time{} }, fakeRunner{})
			if err != nil {
				t.Fatalf("failed to create manager: %v", err)
			}

			selected := make(map[config.AppID]bool)
			for _, appConfig := range manager.Apps {
				selected[appConfig.ID] = true
			}

			values := make(map[string]string)
			switch prov.ID() {
			case "exa":
				values["EXA_API_KEY"] = "11111111-1111-1111-1111-111111111111"
			case "github":
				values["GITHUB_PERSONAL_ACCESS_TOKEN"] = "ghp_1234567890"
			case "context7":
				values["CONTEXT7_API_KEY"] = "ctx7sk-1234567890"
			case "tavily":
				values["TAVILY_API_KEY"] = "tvly-1234567890"
			}

			profiles := []provider.CredentialProfile{
				{
					ProviderID: prov.ID(),
					Values:     values,
					Label:      "test-profile",
				},
			}

			assignments := make(map[config.AppID]int)
			for appID := range selected {
				assignments[appID] = 0
			}

			plan, err := manager.PrepareProvider(prov, profiles, selected, assignments)
			if err != nil {
				t.Fatalf("PrepareProvider failed: %v", err)
			}

			_, err = manager.Apply(plan)
			if err != nil {
				t.Fatalf("Apply failed: %v", err)
			}

			validateGoldenFiles(t, "provider_"+prov.ID(), homeDir)
		})
	}
}

func TestCLI_FailureModes(t *testing.T) {
	cases := []struct {
		name           string
		args           []string
		expectedExit   int
		expectedStderr string
	}{
		{
			name:           "mutually_exclusive_flags",
			args:           []string{"--apply", "--dry-run"},
			expectedExit:   2,
			expectedStderr: "--dry-run and --apply cannot be used together",
		},
		{
			name:           "missing_keys_in_non_interactive",
			args:           []string{"--apply"}, // no keys provided
			expectedExit:   1,
			expectedStderr: "non-interactive mode requires --keys or --keys-file",
		},
		{
			name:           "invalid_provider_keys",
			args:           []string{"--apply", "--keys", "invalid-key"},
			expectedExit:   1,
			expectedStderr: "no UUID-style Exa API keys found", // Note: The exa provider parses this
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			homeDir := t.TempDir()

			out, err := runBinaryWithError(t, tc.args, homeDir)
			
			if err == nil {
				t.Fatalf("expected command to fail with exit code %d, but it succeeded", tc.expectedExit)
			}

			exitErr, ok := err.(*exec.ExitError)
			if !ok {
				t.Fatalf("expected *exec.ExitError, got %T: %v", err, err)
			}

			if exitErr.ExitCode() != tc.expectedExit {
				t.Errorf("expected exit code %d, got %d", tc.expectedExit, exitErr.ExitCode())
			}

			// We need to check if the error message is present in the combined output
			if !bytes.Contains(out, []byte(tc.expectedStderr)) {
				t.Errorf("expected stderr to contain %q\nActual output: %s", tc.expectedStderr, string(out))
			}
		})
	}
}
