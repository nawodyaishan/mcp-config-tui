package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/nawodyaishan/universal-mcp-sync/pkg/doctor"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/manifest"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/provider"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/validate"
)

type providerReadiness struct {
	ID                 string   `json:"id"`
	Name               string   `json:"name"`
	DocsURL            string   `json:"docs_url,omitempty"`
	RuntimeStatus      string   `json:"runtime_status"`
	CredentialStatus   string   `json:"credential_status"`
	RuntimeBlockers    []string `json:"runtime_blockers,omitempty"`
	MissingCredentials []string `json:"missing_credentials,omitempty"`
	GetKeyURLs         []string `json:"get_key_urls,omitempty"`
	Warnings           []string `json:"warnings,omitempty"`
}

func runProvidersCommand(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("usync providers", flag.ContinueOnError)
	flags.SetOutput(stderr)

	var providerID string
	var homeDir string
	var keysFile string
	var keysCSV string
	var live bool
	var jsonOutput bool

	flags.StringVar(&providerID, "provider", "", "provider id")
	flags.StringVar(&homeDir, "home-dir", "", "override the target home directory for testing")
	flags.StringVar(&keysFile, "keys-file", "", "path to a credential file")
	flags.StringVar(&keysCSV, "keys", "", "credential input for a single provider")
	flags.BoolVar(&live, "live", false, "run opt-in live validation where supported")
	flags.BoolVar(&jsonOutput, "json", false, "print machine-readable json output")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if keysCSV != "" && strings.TrimSpace(providerID) == "" {
		_, _ = fmt.Fprintln(stderr, "providers requires --provider when --keys is used")
		return 1
	}

	registry := provider.DefaultRegistry()
	allProviders := registry.All()
	if strings.TrimSpace(providerID) != "" {
		prov, ok := registry.Get(providerID)
		if !ok {
			_, _ = fmt.Fprintf(stderr, "unknown provider %q\n", providerID)
			return 1
		}
		allProviders = []provider.MCPProvider{prov}
	}

	scanner, err := doctor.New(doctor.Options{
		HomeDir:       homeDir,
		CheckRuntimes: true,
	})
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	report, err := scanner.Scan(context.Background())
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}

	service, err := validate.NewService(homeDir)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}

	readiness := make([]providerReadiness, 0, len(allProviders))
	for _, prov := range allProviders {
		item, err := providerStatus(context.Background(), service, report, prov, keysCSV, keysFile, live)
		if err != nil {
			_, _ = fmt.Fprintln(stderr, err)
			return 1
		}
		readiness = append(readiness, item)
	}

	if jsonOutput {
		data, err := json.MarshalIndent(readiness, "", "  ")
		if err != nil {
			_, _ = fmt.Fprintln(stderr, err)
			return 1
		}
		_, _ = stdout.Write(append(data, '\n'))
		return 0
	}

	_, _ = fmt.Fprintln(stdout, "Providers")
	_, _ = fmt.Fprintln(stdout, "=========")
	for _, item := range readiness {
		_, _ = fmt.Fprintf(stdout, "- %s runtime=%s credentials=%s\n", item.Name, item.RuntimeStatus, item.CredentialStatus)
		for _, blocker := range item.RuntimeBlockers {
			_, _ = fmt.Fprintf(stdout, "  runtime: %s\n", blocker)
		}
		for _, missing := range item.MissingCredentials {
			_, _ = fmt.Fprintf(stdout, "  missing: %s\n", missing)
		}
		for _, url := range item.GetKeyURLs {
			_, _ = fmt.Fprintf(stdout, "  get-key: %s\n", url)
		}
		for _, warning := range item.Warnings {
			_, _ = fmt.Fprintf(stdout, "  warning: %s\n", warning)
		}
	}
	return 0
}

func providerStatus(ctx context.Context, service validate.Service, report doctor.Report, prov provider.MCPProvider, keysCSV, keysFile string, live bool) (providerReadiness, error) {
	meta, _ := manifest.ProviderByID(manifest.ProviderID(prov.ID()))
	item := providerReadiness{
		ID:               prov.ID(),
		Name:             prov.Name(),
		DocsURL:          meta.DocsURL,
		RuntimeStatus:    "ready",
		CredentialStatus: "no-credentials-required",
	}

	runtimeByID := make(map[string]doctor.RuntimeFinding, len(report.Runtimes))
	for _, runtime := range report.Runtimes {
		runtimeByID[runtime.ID] = runtime
	}
	for _, runtimeID := range meta.RuntimeIDs {
		runtime, ok := runtimeByID[runtimeID]
		if !ok || !runtime.Available {
			item.RuntimeStatus = "missing"
			item.RuntimeBlockers = append(item.RuntimeBlockers, runtimeID+" is not available")
			continue
		}
		if runtime.Error != "" {
			if item.RuntimeStatus == "ready" {
				item.RuntimeStatus = "warning"
			}
			item.RuntimeBlockers = append(item.RuntimeBlockers, runtimeID+": "+runtime.Error)
		}
	}

	if len(prov.RequiredCredentials()) == 0 {
		return item, nil
	}

	profiles, err := loadValidationProfiles(prov, keysCSV, keysFile)
	if err != nil {
		return providerReadiness{}, err
	}
	validationReport, err := service.ValidateProfiles(ctx, prov, profiles, live)
	if err != nil {
		return providerReadiness{}, err
	}

	item.CredentialStatus = "ready"
	for _, spec := range prov.RequiredCredentials() {
		item.GetKeyURLs = append(item.GetKeyURLs, credentialHelpURL(meta, spec.Key))
	}
	for _, result := range validationReport.Results {
		switch result.Status {
		case validate.StatusFailed:
			if result.Message == "missing required credential" {
				item.CredentialStatus = "missing"
				item.MissingCredentials = append(item.MissingCredentials, result.Key)
			} else {
				item.CredentialStatus = "invalid"
				item.Warnings = append(item.Warnings, result.Key+": "+result.Message)
			}
		case validate.StatusWarning, validate.StatusSkipped:
			if item.CredentialStatus == "ready" {
				item.CredentialStatus = "warning"
			}
			item.Warnings = append(item.Warnings, result.Key+": "+result.Message)
		}
	}
	item.Warnings = append(item.Warnings, validationReport.Warnings...)
	return item, nil
}

func credentialHelpURL(meta manifest.ProviderMeta, key string) string {
	for _, credential := range meta.Credentials {
		if credential.Key == key {
			return credential.GetURL
		}
	}
	return ""
}
