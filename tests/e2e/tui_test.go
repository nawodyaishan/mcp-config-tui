package e2e_test

import (
	"bytes"
	"io"
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/app"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/config"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/tui"
)

func TestTUI_InteractiveFlow(t *testing.T) {
	homeDir := t.TempDir()
	scaffoldHome(t, homeDir)

	manager, err := app.NewManager(homeDir, func() time.Time { return time.Time{} }, fakeRunner{})
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}
	manager.Apps, err = config.DetectAppConfigsForOS(homeDir, "darwin")
	if err != nil {
		t.Fatalf("failed to detect app configs: %v", err)
	}

	keys := []string{"11111111-1111-1111-1111-111111111111"}
	model := tui.NewModel(manager, keys, "11111111-1111-1111-1111-111111111111")

	// Use teatest to wrap the model
	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(100, 30))

	// Wait for the UI to be ready
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("MCP Config")) || bytes.Contains(bts, []byte("Provider"))
	}, teatest.WithDuration(time.Second*3))

	// Step 1: Setup Form (Provider/Key selection)
	// We provided keys, so we can just press Enter to select Provider
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Wait for Target Apps field
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Target Apps"))
	}, teatest.WithDuration(time.Second*3))

	// Press Enter to submit the Target Apps selection
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Wait for assignments screen
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Assign"))
	}, teatest.WithDuration(time.Second*3))

	// Step 2: Assignments
	// Just press Enter to proceed to preview
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Wait for preview screen
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Preview")) || bytes.Contains(bts, []byte("Plan"))
	}, teatest.WithDuration(time.Second*3))

	// Step 3: Preview (Confirm apply)
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Wait for results screen
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Results")) || bytes.Contains(bts, []byte("Successfully updated"))
	}, teatest.WithDuration(time.Second*3))

	// Step 4: Results (Finish)
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Wait for the program to finish completely
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second*5))

	out, err := io.ReadAll(tm.FinalOutput(t))
	if err != nil {
		t.Fatalf("failed to read final output: %v", err)
	}

	// We scrub paths to ensure the test is robust
	outScrubbed := scrubPath(out, homeDir)

	// Create testdata dir if not exists
	goldenFile := filepath.Join("testdata", "tui_interactive_flow.golden")

	// We assert the final output matches the golden file
	assertGolden(t, outScrubbed, goldenFile)
}
