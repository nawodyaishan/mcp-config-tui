package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/nawodyaishan/universal-mcp-sync/pkg/app"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/provider"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/redact"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/validate"
)

func runApplyCommand(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("usync apply", flag.ContinueOnError)
	flags.SetOutput(stderr)

	var planPath string
	var keysFile string
	var keysCSV string
	var homeDir string
	var dryRun bool
	var autoApprove bool
	var forceStale bool

	flags.StringVar(&planPath, "plan", "", "path to the saved plan json")
	flags.StringVar(&keysFile, "keys-file", "", "path to a file containing Exa API keys")
	flags.StringVar(&keysCSV, "keys", "", "comma-separated Exa API keys")
	flags.StringVar(&homeDir, "home-dir", "", "override the target home directory for testing")
	flags.BoolVar(&dryRun, "dry-run", false, "validate and preview the saved plan without writing files")
	flags.BoolVar(&autoApprove, "auto-approve", false, "apply without interactive approval prompts")
	flags.BoolVar(&forceStale, "force-stale", false, "allow expired saved plans")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if strings.TrimSpace(planPath) == "" {
		_, _ = fmt.Fprintln(stderr, "apply requires --plan")
		return 1
	}

	manager, err := app.NewManager(homeDir, nil, nil)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	store, err := app.NewPlanStore(manager.HomeDir)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	plan, err := store.Load(planPath)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}

	credentials, err := loadApplyCredentials(plan, keysCSV, keysFile)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	if plan.ProviderID == "exa" {
		validationService, err := validate.NewService(manager.HomeDir)
		if err != nil {
			_, _ = fmt.Fprintln(stderr, err)
			return 1
		}
		validationReport, err := validationService.ValidateProfiles(context.Background(), provider.NewExaProvider(), exaProfilesFromSavedPlan(plan, credentials), false)
		if err != nil {
			_, _ = fmt.Fprintln(stderr, err)
			return 1
		}
		if validationReport.HasFailures() {
			_, _ = fmt.Fprintln(stderr, validate.FormatReport(validationReport))
			return 1
		}
	}

	opts := app.SavedPlanApplyOptions{
		Credentials: credentials,
		AutoApprove: autoApprove,
		DryRun:      dryRun,
		ForceStale:  forceStale,
		Command:     "usync apply --plan",
	}
	if !dryRun && !autoApprove {
		opts.Approver = terminalApprover{
			in:  os.Stdin,
			out: stderr,
		}
	}

	if dryRun {
		preflight, err := manager.PreflightSavedPlan(plan, opts)
		if err != nil {
			_, _ = fmt.Fprintln(stderr, err)
			return 1
		}
		_, _ = fmt.Fprintln(stdout, app.FormatSavedPlanPreflight(preflight, manager.Now().UTC()))
		return 0
	}

	result, err := manager.ApplySavedPlan(plan, opts)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	_, _ = fmt.Fprintln(stdout, app.FormatApplyResult(result))
	return 0
}

type terminalApprover struct {
	in  io.Reader
	out io.Writer
}

func (a terminalApprover) Confirm(prompt app.ApprovalPrompt) (bool, error) {
	if _, err := fmt.Fprintf(a.out, "%s [y/N]: ", prompt.Message); err != nil {
		return false, err
	}
	reader := bufio.NewReader(a.in)
	line, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true, nil
	default:
		return false, nil
	}
}

func loadApplyCredentials(plan app.SavedPlan, keysCSV, keysFile string) (map[string]string, error) {
	credentials := make(map[string]string)
	envCounts := make(map[string]int)
	for _, ref := range plan.Credentials {
		if ref.EnvVar != "" {
			envCounts[ref.EnvVar]++
		}
	}
	for _, ref := range plan.Credentials {
		ref = normalizeApplyCredentialRef(ref)
		if ref.EnvVar == "" || envCounts[ref.EnvVar] != 1 {
			continue
		}
		if value := strings.TrimSpace(os.Getenv(ref.EnvVar)); value != "" {
			credentials[ref.ID] = value
		}
	}

	if keysCSV == "" && keysFile == "" {
		return credentials, nil
	}
	if plan.ProviderID != "exa" {
		return nil, fmt.Errorf("--keys and --keys-file are currently supported for provider exa plans only")
	}

	keys, _, err := loadInitialKeys(keysCSV, keysFile)
	if err != nil {
		return nil, err
	}
	refsByLabel := make(map[string]app.CredentialRef)
	for _, ref := range plan.Credentials {
		ref = normalizeApplyCredentialRef(ref)
		if ref.Key != "EXA_API_KEY" {
			continue
		}
		if _, exists := refsByLabel[ref.Label]; exists {
			return nil, fmt.Errorf("saved plan has ambiguous credential label %q", ref.Label)
		}
		refsByLabel[ref.Label] = ref
	}
	if len(refsByLabel) == 0 {
		return nil, fmt.Errorf("saved plan does not declare Exa credential references")
	}

	matched := 0
	for _, key := range keys {
		label := redact.Key(key)
		ref, ok := refsByLabel[label]
		if !ok {
			continue
		}
		credentials[ref.ID] = key
		matched++
	}
	if matched == 0 {
		return nil, fmt.Errorf("supplied Exa keys did not match any saved plan credential labels")
	}

	return credentials, nil
}

func normalizeApplyCredentialRef(ref app.CredentialRef) app.CredentialRef {
	if ref.ID == "" {
		ref.ID = defaultApplyCredentialRefID(ref.Key, ref.Label)
	}
	return ref
}

func defaultApplyCredentialRefID(key, label string) string {
	if key == "" {
		return label
	}
	if label == "" {
		return key
	}
	return key + ":" + label
}

func exaProfilesFromSavedPlan(plan app.SavedPlan, credentials map[string]string) []provider.CredentialProfile {
	profiles := make([]provider.CredentialProfile, 0, len(plan.Credentials))
	for _, ref := range plan.Credentials {
		ref = normalizeApplyCredentialRef(ref)
		if ref.Key != "EXA_API_KEY" {
			continue
		}
		value := strings.TrimSpace(credentials[ref.ID])
		if value == "" {
			continue
		}
		profiles = append(profiles, provider.CredentialProfile{
			ProviderID: "exa",
			Values: map[string]string{
				"EXA_API_KEY": value,
			},
			Label: ref.Label,
		})
	}
	return profiles
}
