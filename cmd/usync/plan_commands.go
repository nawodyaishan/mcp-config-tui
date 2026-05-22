package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/nawodyaishan/universal-mcp-sync/pkg/app"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/config"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/provider"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/redact"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/validate"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/version"
)

func runPlanCommand(args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 {
		switch args[0] {
		case "list":
			return runPlanList(args[1:], stdout, stderr)
		case "clean":
			return runPlanClean(args[1:], stdout, stderr)
		}
	}
	return runPlan(args, stdout, stderr)
}

func runPlan(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("usync plan", flag.ContinueOnError)
	flags.SetOutput(stderr)

	var providerID string
	var keysFile string
	var keysCSV string
	var homeDir string
	var targetsCSV string
	var outPath string
	var allDetected bool

	flags.StringVar(&providerID, "provider", "", "provider id")
	flags.StringVar(&keysFile, "keys-file", "", "path to a file containing Exa API keys")
	flags.StringVar(&keysCSV, "keys", "", "comma-separated Exa API keys")
	flags.StringVar(&homeDir, "home-dir", "", "override the target home directory for testing")
	flags.StringVar(&targetsCSV, "targets", "", "comma-separated target app IDs")
	flags.StringVar(&outPath, "out", "", "output path for the saved plan json")
	flags.BoolVar(&allDetected, "all-detected", false, "select all detected targets")

	if err := flags.Parse(args); err != nil {
		return 2
	}
	if providerID == "" {
		_, _ = fmt.Fprintln(stderr, "plan requires --provider")
		return 1
	}
	if providerID != "exa" {
		_, _ = fmt.Fprintln(stderr, "saved plan creation currently supports --provider exa only")
		return 1
	}
	if targetsCSV == "" && !allDetected {
		_, _ = fmt.Fprintln(stderr, "plan requires --targets or --all-detected")
		return 1
	}
	if allDetected {
		_, _ = fmt.Fprintln(stderr, "--all-detected requires doctor mode and is not implemented yet")
		return 1
	}

	manager, err := app.NewManager(homeDir, nil, nil)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}

	initialKeys, _, err := loadInitialKeys(keysCSV, keysFile)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	if len(initialKeys) == 0 {
		_, _ = fmt.Fprintln(stderr, "plan requires --keys or --keys-file for provider exa")
		return 1
	}
	validationService, err := validate.NewService(manager.HomeDir)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	validationReport, err := validationService.ValidateProfiles(context.Background(), provider.NewExaProvider(), exaProfilesFromKeys(initialKeys), false)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	if validationReport.HasFailures() {
		_, _ = fmt.Fprintln(stderr, validate.FormatReport(validationReport))
		return 1
	}

	selected, err := parseTargetSelection(manager.Apps, targetsCSV)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}

	legacyPlan, err := manager.Prepare(initialKeys, selected, app.DefaultAssignments(selected, len(initialKeys)))
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}

	planID, err := app.NewPlanID()
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	credentialRefs := make([]app.CredentialRef, len(initialKeys))
	for i, key := range initialKeys {
		credentialRefs[i] = app.CredentialRef{
			Key:    "EXA_API_KEY",
			Label:  redact.Key(key),
			EnvVar: "EXA_API_KEY",
		}
	}

	savedPlan, err := manager.BuildSavedPlan(legacyPlan, app.SavedPlanOptions{
		PlanID:       planID,
		CreatedAt:    manager.Now().UTC(),
		UsyncVersion: version.Version,
		ProviderID:   providerID,
		Credentials:  credentialRefs,
	})
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}

	store, err := app.NewPlanStore(manager.HomeDir)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	store.Now = manager.Now
	path, err := store.Save(savedPlan, outPath)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	_, _ = fmt.Fprintln(stdout, path)
	return 0
}

func runShow(args []string, stdout, stderr io.Writer) int {
	jsonOutput, args := consumeBoolFlag(args, "--json")
	homeDir, args, err := consumeStringFlag(args, "--home-dir")
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 2
	}
	flags := flag.NewFlagSet("usync show", flag.ContinueOnError)
	flags.SetOutput(stderr)
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 1 {
		_, _ = fmt.Fprintln(stderr, "show requires exactly one plan path")
		return 1
	}

	store, err := app.NewPlanStore(homeDir)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	plan, err := store.Load(flags.Arg(0))
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	if jsonOutput {
		data, err := app.MarshalSavedPlanJSON(plan)
		if err != nil {
			_, _ = fmt.Fprintln(stderr, err)
			return 1
		}
		_, _ = stdout.Write(data)
		return 0
	}
	_, _ = fmt.Fprintln(stdout, app.FormatSavedPlan(plan, time.Now().UTC()))
	return 0
}

func runPlanList(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("usync plan list", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var homeDir string
	flags.StringVar(&homeDir, "home-dir", "", "override the target home directory for testing")
	if err := flags.Parse(args); err != nil {
		return 2
	}

	store, err := app.NewPlanStore(homeDir)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	plans, err := store.List()
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	if len(plans) == 0 {
		_, _ = fmt.Fprintln(stdout, "no saved plans")
		return 0
	}
	_, _ = fmt.Fprintln(stdout, "Saved plans")
	_, _ = fmt.Fprintln(stdout, "===========")
	for _, plan := range plans {
		expired := ""
		if plan.Expired {
			expired = " expired"
		}
		_, _ = fmt.Fprintf(stdout, "- %s provider=%s created=%s expires=%s%s path=%s\n",
			plan.PlanID,
			plan.ProviderID,
			plan.CreatedAt.UTC().Format(time.RFC3339),
			plan.ExpiresAt.UTC().Format(time.RFC3339),
			expired,
			plan.Path,
		)
	}
	return 0
}

func runPlanClean(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("usync plan clean", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var homeDir string
	var expiredOnly bool
	var removeAll bool
	flags.StringVar(&homeDir, "home-dir", "", "override the target home directory for testing")
	flags.BoolVar(&expiredOnly, "expired", false, "remove expired plans only")
	flags.BoolVar(&removeAll, "all", false, "remove all saved plans")
	if err := flags.Parse(args); err != nil {
		return 2
	}

	store, err := app.NewPlanStore(homeDir)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	removed, err := store.Clean(app.CleanOptions{ExpiredOnly: expiredOnly, RemoveAll: removeAll})
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	if len(removed) == 0 {
		_, _ = fmt.Fprintln(stdout, "no saved plans removed")
		return 0
	}
	for _, path := range removed {
		_, _ = fmt.Fprintln(stdout, path)
	}
	return 0
}

func parseTargetSelection(apps []config.AppConfig, targetsCSV string) (map[config.AppID]bool, error) {
	selected := make(map[config.AppID]bool)
	known := make(map[string]config.AppID, len(apps))
	for _, appConfig := range apps {
		known[string(appConfig.ID)] = appConfig.ID
	}

	for _, part := range strings.Split(targetsCSV, ",") {
		target := strings.TrimSpace(part)
		if target == "" {
			continue
		}
		appID, ok := known[target]
		if !ok {
			return nil, fmt.Errorf("unknown target %q", target)
		}
		selected[appID] = true
	}
	if len(selected) == 0 {
		return nil, fmt.Errorf("no valid targets selected")
	}
	return selected, nil
}

func consumeBoolFlag(args []string, name string) (bool, []string) {
	remaining := make([]string, 0, len(args))
	found := false
	for _, arg := range args {
		if arg == name {
			found = true
			continue
		}
		remaining = append(remaining, arg)
	}
	return found, remaining
}

func consumeStringFlag(args []string, name string) (string, []string, error) {
	remaining := make([]string, 0, len(args))
	value := ""
	for i := 0; i < len(args); i++ {
		if args[i] != name {
			remaining = append(remaining, args[i])
			continue
		}
		if i+1 >= len(args) {
			return "", nil, fmt.Errorf("%s requires a value", name)
		}
		value = args[i+1]
		i++
	}
	return value, remaining, nil
}
