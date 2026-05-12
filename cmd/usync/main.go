package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nawodyaishan/universal-mcp-sync/pkg/app"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/config"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/exa"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/tui"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/version"
)

func loadInitialKeys(keysCSV, keysFile string) ([]string, string, error) {
	if keysCSV != "" {
		keys, err := exa.ParseKeysCSV(keysCSV)
		return keys, keysCSV, err
	}
	if keysFile != "" {
		keys, err := exa.ParseKeysFile(keysFile)
		if err != nil {
			return nil, "", err
		}
		data, err := os.ReadFile(keysFile)
		if err != nil {
			return nil, "", err
		}
		return keys, string(data), nil
	}
	return nil, "", nil
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "sync" {
		// Strip "sync" and continue as if it were the main command
		os.Args = append(os.Args[:1], os.Args[2:]...)
	}

	var keysFile string
	var keysCSV string
	var homeDir string
	var dryRun bool
	var apply bool
	var showVersion bool

	flag.StringVar(&keysFile, "keys-file", "", "path to a file containing Exa API keys")
	flag.StringVar(&keysCSV, "keys", "", "comma-separated Exa API keys")
	flag.StringVar(&homeDir, "home-dir", "", "override the target home directory for testing")
	flag.BoolVar(&dryRun, "dry-run", false, "print the redacted plan without writing files")
	flag.BoolVar(&apply, "apply", false, "apply updates without launching the TUI")
	flag.BoolVar(&showVersion, "version", false, "print version information and exit")
	flag.Parse()

	if showVersion {
		_, _ = fmt.Fprintln(os.Stdout, version.String())
		return
	}

	if dryRun && apply {
		fmt.Fprintln(os.Stderr, "--dry-run and --apply cannot be used together")
		os.Exit(2)
	}

	manager, err := app.NewManager(homeDir, nil, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if os.Getenv("USYNC_DEBUG") == "true" {
		fmt.Fprintf(os.Stderr, "DEBUG: detected %d apps\n", len(manager.Apps))
		for _, a := range manager.Apps {
			fmt.Fprintf(os.Stderr, "DEBUG: app %s id=%s files=%d\n", a.Name, a.ID, len(a.Files))
		}
	}

	initialKeys, initialRaw, err := loadInitialKeys(keysCSV, keysFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if !apply && !dryRun {
		model := tui.NewModel(manager, initialKeys, initialRaw)
		program := tea.NewProgram(model, tea.WithAltScreen())
		finalModel, err := program.Run()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if finalTyped, ok := finalModel.(tui.Model); ok && finalTyped.Err() != nil {
			fmt.Fprintln(os.Stderr, finalTyped.Err())
			os.Exit(1)
		}
		return
	}

	if len(initialKeys) == 0 {
		fmt.Fprintln(os.Stderr, "non-interactive mode requires --keys or --keys-file")
		os.Exit(1)
	}

	selected := mapAllSelected(manager.Apps)
	if os.Getenv("USYNC_DEBUG") == "true" {
		fmt.Fprintf(os.Stderr, "DEBUG: selected %d apps\n", len(selected))
	}
	assignments := app.DefaultAssignments(selected, len(initialKeys))

	plan, err := manager.Prepare(initialKeys, selected, assignments)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if os.Getenv("USYNC_DEBUG") == "true" {
		fmt.Fprintf(os.Stderr, "DEBUG: plan has %d operations\n", len(plan.Operations))
	}

	if dryRun {
		_, _ = fmt.Fprintln(os.Stdout, app.FormatPlan(plan))
		return
	}

	result, err := manager.Apply(plan)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	_, _ = fmt.Fprintln(os.Stdout, app.FormatApplyResult(result))
}

func mapAllSelected(apps []config.AppConfig) map[config.AppID]bool {
	selected := make(map[config.AppID]bool, len(apps))
	for _, appConfig := range apps {
		selected[appConfig.ID] = true
	}
	return selected
}
