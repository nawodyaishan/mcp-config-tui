package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/nawodyaishan/universal-mcp-sync/pkg/exa"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/provider"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/validate"
)

func runValidateCommand(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("usync validate", flag.ContinueOnError)
	flags.SetOutput(stderr)

	var providerID string
	var keysFile string
	var keysCSV string
	var homeDir string
	var live bool
	var jsonOutput bool

	flags.StringVar(&providerID, "provider", "", "provider id")
	flags.StringVar(&keysFile, "keys-file", "", "path to a credential file")
	flags.StringVar(&keysCSV, "keys", "", "credential input; Exa supports comma-separated keys")
	flags.StringVar(&homeDir, "home-dir", "", "override the target home directory for testing")
	flags.BoolVar(&live, "live", false, "run opt-in live validation where supported")
	flags.BoolVar(&jsonOutput, "json", false, "print machine-readable json output")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if strings.TrimSpace(providerID) == "" {
		_, _ = fmt.Fprintln(stderr, "validate requires --provider")
		return 1
	}

	prov, err := resolveProvider(providerID)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}

	service, err := validate.NewService(homeDir)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	profiles, err := loadValidationProfiles(prov, keysCSV, keysFile)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}

	report, err := service.ValidateProfiles(context.Background(), prov, profiles, live)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}

	if jsonOutput {
		data, err := validate.MarshalReportJSON(report)
		if err != nil {
			_, _ = fmt.Fprintln(stderr, err)
			return 1
		}
		_, _ = stdout.Write(data)
	} else {
		_, _ = fmt.Fprintln(stdout, validate.FormatReport(report))
	}

	if report.HasFailures() {
		return 1
	}
	return 0
}

func resolveProvider(providerID string) (provider.MCPProvider, error) {
	registry := provider.DefaultRegistry()
	prov, ok := registry.Get(providerID)
	if !ok {
		return nil, fmt.Errorf("unknown provider %q", providerID)
	}
	return prov, nil
}

func loadValidationProfiles(prov provider.MCPProvider, keysCSV, keysFile string) ([]provider.CredentialProfile, error) {
	specs := prov.RequiredCredentials()
	if keysCSV != "" {
		if parser, ok := prov.(provider.MultiValueParser); ok && len(specs) == 1 {
			return parser.ParseMultiValue(specs[0].Key, keysCSV)
		}
		if len(specs) != 1 {
			return nil, fmt.Errorf("--keys is only supported for providers with exactly one required credential")
		}
		return []provider.CredentialProfile{{
			ProviderID: prov.ID(),
			Values: map[string]string{
				specs[0].Key: strings.TrimSpace(keysCSV),
			},
		}}, nil
	}

	if keysFile == "" {
		return nil, nil
	}

	data, err := os.ReadFile(keysFile)
	if err != nil {
		return nil, fmt.Errorf("read keys file: %w", err)
	}

	if parser, ok := prov.(provider.MultiValueParser); ok && len(specs) == 1 {
		parsed, parseErr := validate.ParseKeyFile(data)
		if parseErr == nil {
			values := parsed.ValuesForKey(specs[0].Key)
			if len(values) > 0 {
				return parser.ParseMultiValue(specs[0].Key, strings.Join(values, "\n"))
			}
		}
		return parser.ParseMultiValue(specs[0].Key, string(data))
	}

	parsed, err := validate.ParseKeyFile(data)
	if err != nil {
		return nil, err
	}
	if len(specs) == 0 {
		return nil, nil
	}

	values := make(map[string]string, len(specs))
	for _, spec := range specs {
		found := parsed.ValuesForKey(spec.Key)
		if len(found) == 0 {
			continue
		}
		values[spec.Key] = found[len(found)-1]
	}

	return []provider.CredentialProfile{{
		ProviderID: prov.ID(),
		Values:     values,
	}}, nil
}

func exaProfilesFromKeys(keys []string) []provider.CredentialProfile {
	profiles := make([]provider.CredentialProfile, 0, len(keys))
	for _, key := range keys {
		profiles = append(profiles, provider.CredentialProfile{
			ProviderID: "exa",
			Values: map[string]string{
				"EXA_API_KEY": key,
			},
			Label: exa.RedactKey(key),
		})
	}
	return profiles
}
