package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/nawodyaishan/universal-mcp-sync/pkg/app"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/provider"
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

	prov, err := resolveProvider(plan.ProviderID)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}

	credentials, err := loadApplyCredentials(plan, prov, keysCSV, keysFile)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}

	validationService, err := validate.NewService(manager.HomeDir)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	validationReport, err := validationService.ValidateProfiles(context.Background(), prov, profilesFromSavedPlan(plan, prov, credentials), false)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	if validationReport.HasFailures() {
		_, _ = fmt.Fprintln(stderr, validate.FormatReport(validationReport))
		return 1
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

func loadApplyCredentials(plan app.SavedPlan, prov provider.MCPProvider, keysCSV, keysFile string) (map[string]string, error) {
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

	profiles, err := loadValidationProfiles(prov, keysCSV, keysFile)
	if err != nil {
		return nil, err
	}
	if len(profiles) == 0 {
		return credentials, nil
	}

	matched, err := assignSuppliedPlanCredentials(plan, prov, profiles, credentials)
	if err != nil {
		return nil, err
	}
	if matched == 0 && len(plan.Credentials) > 0 {
		return nil, fmt.Errorf("supplied credentials did not match any saved plan credential labels")
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

func assignSuppliedPlanCredentials(plan app.SavedPlan, prov provider.MCPProvider, profiles []provider.CredentialProfile, credentials map[string]string) (int, error) {
	if len(plan.Credentials) == 0 {
		return 0, nil
	}

	refsByKeyAndLabel := make(map[string]app.CredentialRef, len(plan.Credentials))
	refsByKeyCount := make(map[string]int)
	for _, ref := range plan.Credentials {
		ref = normalizeApplyCredentialRef(ref)
		key := ref.Key + "\x00" + ref.Label
		if _, exists := refsByKeyAndLabel[key]; exists {
			return 0, fmt.Errorf("saved plan has ambiguous credential label %q", ref.Label)
		}
		refsByKeyAndLabel[key] = ref
		refsByKeyCount[ref.Key]++
	}

	matched := 0
	for _, profile := range profiles {
		for _, spec := range prov.RequiredCredentials() {
			value := strings.TrimSpace(profile.Values[spec.Key])
			if value == "" {
				continue
			}
			label := strings.TrimSpace(profile.Label)
			if label == "" {
				label = validate.RedactedCredentialLabel(prov.ID(), spec.Key, value)
			}

			ref, ok := refsByKeyAndLabel[spec.Key+"\x00"+label]
			if !ok && refsByKeyCount[spec.Key] == 1 {
				for _, candidate := range plan.Credentials {
					candidate = normalizeApplyCredentialRef(candidate)
					if candidate.Key == spec.Key {
						ref = candidate
						ok = true
						break
					}
				}
			}
			if !ok {
				continue
			}
			credentials[ref.ID] = value
			matched++
		}
	}
	return matched, nil
}

func profilesFromSavedPlan(plan app.SavedPlan, prov provider.MCPProvider, credentials map[string]string) []provider.CredentialProfile {
	if len(plan.Credentials) == 0 {
		return nil
	}

	profilesByLabel := make(map[string]provider.CredentialProfile)
	order := make([]string, 0, len(plan.Credentials))
	for _, ref := range plan.Credentials {
		ref = normalizeApplyCredentialRef(ref)
		value := strings.TrimSpace(credentials[ref.ID])
		if value == "" {
			continue
		}

		label := ref.Label
		profile, exists := profilesByLabel[label]
		if !exists {
			profile = provider.CredentialProfile{
				ProviderID: prov.ID(),
				Values:     make(map[string]string),
				Label:      label,
			}
			profilesByLabel[label] = profile
			order = append(order, label)
		}
		profile.Values[ref.Key] = value
		profilesByLabel[label] = profile
	}

	slices.Sort(order)
	profiles := make([]provider.CredentialProfile, 0, len(order))
	for _, label := range order {
		profiles = append(profiles, profilesByLabel[label])
	}
	return profiles
}
