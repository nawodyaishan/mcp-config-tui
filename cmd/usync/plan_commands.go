package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/nawodyaishan/universal-mcp-sync/pkg/app"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/config"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/doctor"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/manifest"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/provider"
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
	var workspace string
	var targetsCSV string
	var outPath string
	var allDetected bool
	var includeWorkspace bool
	var detailedExitCode bool

	flags.StringVar(&providerID, "provider", "", "provider id")
	flags.StringVar(&keysFile, "keys-file", "", "path to a credential file")
	flags.StringVar(&keysCSV, "keys", "", "credential input; Exa supports comma-separated keys")
	flags.StringVar(&homeDir, "home-dir", "", "override the target home directory for testing")
	flags.StringVar(&workspace, "workspace", "", "override the workspace directory for project/workspace config detection")
	flags.StringVar(&targetsCSV, "targets", "", "comma-separated target app IDs")
	flags.StringVar(&outPath, "out", "", "output path for the saved plan json")
	flags.BoolVar(&allDetected, "all-detected", false, "select all detected targets")
	flags.BoolVar(&includeWorkspace, "include-workspace", false, "include project/workspace config candidates when detected")
	flags.BoolVar(&detailedExitCode, "detailed-exitcode", false, "return 2 when the saved plan contains pending changes")

	if err := flags.Parse(args); err != nil {
		return 2
	}
	if providerID == "" {
		_, _ = fmt.Fprintln(stderr, "plan requires --provider")
		return 1
	}
	if targetsCSV == "" && !allDetected {
		_, _ = fmt.Fprintln(stderr, "plan requires --targets or --all-detected")
		return 1
	}
	if targetsCSV != "" && allDetected {
		_, _ = fmt.Fprintln(stderr, "plan accepts either --targets or --all-detected")
		return 1
	}

	manager, err := app.NewManager(homeDir, nil, nil)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}

	prov, err := resolveProvider(providerID)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}

	profiles, err := loadValidationProfiles(prov, keysCSV, keysFile)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	validationService, err := validate.NewService(manager.HomeDir)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	validationReport, err := validationService.ValidateProfiles(context.Background(), prov, profiles, false)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	if validationReport.HasFailures() {
		_, _ = fmt.Fprintln(stderr, validate.FormatReport(validationReport))
		return 1
	}

	if len(profiles) == 0 {
		profiles = []provider.CredentialProfile{{
			ProviderID: prov.ID(),
			Values:     map[string]string{},
		}}
	}

	workspaceDir, err := resolveWorkspaceDir(workspace, includeWorkspace)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}

	discovery, err := buildPlanDiscovery(manager, targetsCSV, allDetected, workspaceDir, includeWorkspace)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}

	manager.Apps = discovery.Apps
	executionPlan, err := manager.PrepareProvider(prov, profiles, discovery.Selected, app.DefaultAssignments(discovery.Selected, len(profiles)))
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	if len(discovery.Warnings) > 0 {
		executionPlan.Warnings = append(executionPlan.Warnings, discovery.Warnings...)
	}

	planID, err := app.NewPlanID()
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}

	savedPlan, err := manager.BuildSavedPlan(executionPlan, app.SavedPlanOptions{
		PlanID:       planID,
		CreatedAt:    manager.Now().UTC(),
		UsyncVersion: version.Version,
		ProviderID:   providerID,
		Credentials:  buildCredentialRefs(prov, profiles),
		Doctor:       discovery.Summary,
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
	if detailedExitCode && len(savedPlan.Operations) > 0 {
		return 2
	}
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

type planDiscovery struct {
	Apps     []config.AppConfig
	Selected map[config.AppID]bool
	Summary  app.DoctorSummary
	Warnings []string
}

func buildPlanDiscovery(manager *app.Manager, targetsCSV string, allDetected bool, workspaceDir string, includeWorkspace bool) (planDiscovery, error) {
	report, err := scanDoctorReport(manager.HomeDir, workspaceDir)
	if err != nil && allDetected {
		return planDiscovery{}, err
	}

	discovery := planDiscovery{
		Apps:    append([]config.AppConfig(nil), manager.Apps...),
		Summary: summarizeDoctorReport(report),
	}
	if err != nil {
		discovery.Warnings = append(discovery.Warnings, "doctor summary unavailable: "+err.Error())
	}

	if !allDetected {
		selected, err := parseTargetSelection(manager.Apps, targetsCSV)
		if err != nil {
			return planDiscovery{}, err
		}
		discovery.Selected = selected
		return discovery, nil
	}

	appByID := make(map[config.AppID]config.AppConfig, len(manager.Apps))
	for _, appConfig := range manager.Apps {
		appByID[appConfig.ID] = appConfig
	}

	discovery.Apps = make([]config.AppConfig, 0, len(report.Clients))
	discovery.Selected = make(map[config.AppID]bool)
	for _, finding := range report.Clients {
		switch finding.Confidence {
		case doctor.ConfidenceConflict:
			discovery.Warnings = append(discovery.Warnings, fmt.Sprintf("%s: skipped detected target due to conflict", finding.Name))
			continue
		case doctor.ConfidenceLow:
			discovery.Warnings = append(discovery.Warnings, fmt.Sprintf("%s: skipped detected target due to low confidence", finding.Name))
			continue
		}

		appConfig, warning, ok := buildDetectedAppConfig(appByID[config.AppID(finding.ID)], finding, includeWorkspace)
		if warning != "" {
			discovery.Warnings = append(discovery.Warnings, warning)
		}
		if !ok {
			continue
		}
		discovery.Apps = append(discovery.Apps, appConfig)
		discovery.Selected[appConfig.ID] = true
	}

	slices.SortStableFunc(discovery.Apps, func(a, b config.AppConfig) int {
		return slices.Index(config.AppOrder, a.ID) - slices.Index(config.AppOrder, b.ID)
	})
	return discovery, nil
}

func scanDoctorReport(homeDir, workspaceDir string) (doctor.Report, error) {
	scanner, err := doctor.New(doctor.Options{
		HomeDir:       homeDir,
		WorkspaceDir:  workspaceDir,
		CheckRuntimes: false,
	})
	if err != nil {
		return doctor.Report{}, err
	}
	return scanner.Scan(context.Background())
}

func summarizeDoctorReport(report doctor.Report) app.DoctorSummary {
	summary := app.DoctorSummary{
		Warnings: append([]string(nil), report.Warnings...),
	}
	for _, client := range report.Clients {
		if client.Installed || client.EffectivePath != "" || client.CLIAvailable {
			summary.ClientsDetected++
		}
		switch client.Confidence {
		case doctor.ConfidenceConflict:
			summary.Conflicts++
		case doctor.ConfidenceHigh, doctor.ConfidenceMedium:
			summary.ClientsReady++
		}
	}
	return summary
}

func buildDetectedAppConfig(existing config.AppConfig, finding doctor.ClientFinding, includeWorkspace bool) (config.AppConfig, string, bool) {
	effective, hasEffective := effectiveCandidate(finding)
	if hasEffective && (effective.Scope == manifest.ScopeProject || effective.Scope == manifest.ScopeWorkspace) && !includeWorkspace {
		return config.AppConfig{}, fmt.Sprintf("%s: skipped %s-scoped config; rerun with --include-workspace", finding.Name, effective.Scope), false
	}

	if existing.ID != "" && matchesExistingAppConfig(existing, finding, effective, hasEffective) {
		return existing, "", true
	}

	if finding.ID == manifest.ClientClaudeCode && !hasEffective {
		return existing, "", true
	}
	if !hasEffective {
		return config.AppConfig{}, fmt.Sprintf("%s: skipped detected target because no writable config candidate was found", finding.Name), false
	}

	file := config.TargetFile{
		Label:      effective.Label,
		Path:       effective.Path,
		Kind:       fileKindForDoctorCandidate(finding.ID, effective),
		Exists:     effective.Exists,
		Creatable:  true,
		Scope:      string(effective.Scope),
		GitWarning: effective.Scope == manifest.ScopeProject || effective.Scope == manifest.ScopeWorkspace,
	}
	return config.AppConfig{
		ID:    config.AppID(finding.ID),
		Name:  finding.Name,
		Files: []config.TargetFile{file},
	}, "", true
}

func matchesExistingAppConfig(existing config.AppConfig, finding doctor.ClientFinding, effective doctor.CandidateFinding, hasEffective bool) bool {
	if config.AppID(finding.ID) == config.AppClaudeCode && hasEffective {
		return len(existing.Files) > 0 && existing.Files[0].Scope == string(effective.Scope)
	}
	if !hasEffective {
		return true
	}
	for _, file := range existing.Files {
		if file.Path == effective.Path {
			return true
		}
	}
	return false
}

func effectiveCandidate(finding doctor.ClientFinding) (doctor.CandidateFinding, bool) {
	for _, candidate := range finding.Candidates {
		if candidate.Path == finding.EffectivePath {
			return candidate, true
		}
	}
	return doctor.CandidateFinding{}, false
}

func fileKindForDoctorCandidate(clientID manifest.ClientID, candidate doctor.CandidateFinding) config.FileKind {
	switch clientID {
	case manifest.ClientClaudeCode:
		return config.FileKindClaudeCodeCLI
	case manifest.ClientCodexCLI:
		return config.FileKindCodexTOML
	case manifest.ClientVSCode, manifest.ClientZed, manifest.ClientOpenCode:
		return config.FileKindNamedServer
	case manifest.ClientGeminiCLI, manifest.ClientAntigravityCLI:
		if candidate.RootKey == "" {
			return config.FileKindBareMCPServers
		}
		return config.FileKindMCPServers
	default:
		return config.FileKindMCPServers
	}
}

func resolveWorkspaceDir(workspace string, includeWorkspace bool) (string, error) {
	if strings.TrimSpace(workspace) != "" {
		return workspace, nil
	}
	if !includeWorkspace {
		return "", nil
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve workspace directory: %w", err)
	}
	return wd, nil
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
