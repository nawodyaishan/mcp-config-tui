package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/nawodyaishan/universal-mcp-sync/pkg/migrate"
)

func runMigrateCommand(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		_, _ = fmt.Fprintln(stderr, "migrate requires a subcommand; available: gemini-to-antigravity")
		return 1
	}
	switch args[0] {
	case "gemini-to-antigravity":
		return runMigrateGeminiToAntigravity(args[1:], stdout, stderr)
	default:
		_, _ = fmt.Fprintf(stderr, "unknown migrate subcommand %q; available: gemini-to-antigravity\n", args[0])
		return 1
	}
}

func runMigrateGeminiToAntigravity(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("usync migrate gemini-to-antigravity", flag.ContinueOnError)
	flags.SetOutput(stderr)

	var targetStr string
	var homeDir string
	var dryRun bool
	var apply bool

	flags.StringVar(&targetStr, "target", "", "migration target: antigravity-cli or antigravity-ide")
	flags.StringVar(&homeDir, "home-dir", "", "override home directory")
	flags.BoolVar(&dryRun, "dry-run", false, "show migration preview without writing files")
	flags.BoolVar(&apply, "apply", false, "write the migration to disk")

	if err := flags.Parse(args); err != nil {
		return 2
	}
	if dryRun && apply {
		_, _ = fmt.Fprintln(stderr, "--dry-run and --apply cannot be used together")
		return 2
	}
	if !dryRun && !apply {
		dryRun = true
	}

	if homeDir == "" {
		var err error
		homeDir, err = os.UserHomeDir()
		if err != nil {
			_, _ = fmt.Fprintln(stderr, "resolve home directory:", err)
			return 1
		}
	}

	target, err := pickMigrateTarget(homeDir, migrate.TargetID(targetStr))
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}

	opts := migrate.Options{
		HomeDir: homeDir,
		Target:  target,
	}

	preview, err := migrate.Plan(opts)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}

	if dryRun {
		_, _ = fmt.Fprint(stdout, migrate.Format(preview))
		return 0
	}

	result, err := migrate.Apply(opts, preview)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}

	_, _ = fmt.Fprint(stdout, migrate.FormatResult(result))
	return 0
}

// pickMigrateTarget resolves the migration target.
// If targetStr is empty, it auto-selects when only one Antigravity target config exists.
// When both exist, an explicit --target is required.
func pickMigrateTarget(homeDir string, target migrate.TargetID) (migrate.TargetID, error) {
	if target != "" {
		if _, err := migrate.TargetPath(homeDir, target); err != nil {
			return "", err
		}
		return target, nil
	}

	existing := migrate.ExistingTargets(homeDir)
	if len(existing) == 0 {
		return migrate.TargetAntigravityCLI, nil
	}
	if len(existing) == 1 {
		return existing[0], nil
	}
	return "", fmt.Errorf(
		"both antigravity-cli and antigravity-ide configs exist; specify --target antigravity-cli or --target antigravity-ide",
	)
}
