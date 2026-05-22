package validate

import (
	"fmt"
	"strings"

	"github.com/nawodyaishan/universal-mcp-sync/pkg/context7"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/exa"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/manifest"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/provider"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/redact"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/tavily"
)

func Offline(req Request) Report {
	if req.Provider == nil {
		return Report{
			Results: []Result{{
				Status:  StatusFailed,
				Mode:    ModeOffline,
				Message: "missing provider",
			}},
		}
	}
	return OfflineProfiles(req.Provider, []provider.CredentialProfile{{
		ProviderID: req.Provider.ID(),
		Values:     cloneValues(req.Values),
	}})
}

func OfflineProfiles(prov provider.MCPProvider, profiles []provider.CredentialProfile) Report {
	report := Report{
		ProviderID: prov.ID(),
	}

	specs := prov.RequiredCredentials()
	if len(specs) == 0 {
		report.Results = []Result{{
			ProviderID: prov.ID(),
			Label:      prov.Name(),
			Status:     StatusOK,
			Mode:       ModeOffline,
			Message:    "no credentials required",
		}}
		return report
	}

	if len(profiles) == 0 {
		profiles = []provider.CredentialProfile{{
			ProviderID: prov.ID(),
			Values:     map[string]string{},
		}}
	}

	for _, profile := range profiles {
		for _, spec := range specs {
			value := strings.TrimSpace(profile.Values[spec.Key])
			label := strings.TrimSpace(profile.Label)
			if label == "" && value != "" {
				label = redactedCredentialLabel(prov.ID(), spec.Key, value)
			}
			if label == "" {
				label = spec.Key
			}

			result := Result{
				ProviderID: prov.ID(),
				Key:        spec.Key,
				Label:      label,
				Status:     StatusOK,
				Mode:       ModeOffline,
				HelpURL:    credentialHelpURL(prov.ID(), spec.Key),
			}

			if value == "" {
				result.Status = StatusFailed
				result.Message = "missing required credential"
				report.Results = append(report.Results, result)
				continue
			}

			if spec.Validator == nil {
				result.Message = "credential present"
				report.Results = append(report.Results, result)
				continue
			}

			if err := spec.Validator(value); err != nil {
				result.Status = StatusFailed
				result.Message = redact.Text(err.Error())
				report.Results = append(report.Results, result)
				continue
			}

			result.Message = "credential format valid"
			report.Results = append(report.Results, result)
		}
	}

	report.Results = append(report.Results, duplicateResults(prov, profiles)...)
	return report
}

func duplicateResults(prov provider.MCPProvider, profiles []provider.CredentialProfile) []Result {
	if len(profiles) < 2 {
		return nil
	}

	type occurrence struct {
		key   string
		label string
	}

	seen := make(map[string][]occurrence)
	for _, profile := range profiles {
		for _, spec := range prov.RequiredCredentials() {
			value := strings.TrimSpace(profile.Values[spec.Key])
			if value == "" {
				continue
			}

			label := strings.TrimSpace(profile.Label)
			if label == "" {
				label = redactedCredentialLabel(prov.ID(), spec.Key, value)
			}

			cacheKey := spec.Key + "\x00" + value
			seen[cacheKey] = append(seen[cacheKey], occurrence{
				key:   spec.Key,
				label: label,
			})
		}
	}

	results := make([]Result, 0)
	for _, occurrences := range seen {
		if len(occurrences) < 2 {
			continue
		}
		for _, occurrence := range occurrences {
			results = append(results, Result{
				ProviderID: prov.ID(),
				Key:        occurrence.key,
				Label:      occurrence.label,
				Status:     StatusWarning,
				Mode:       ModeOffline,
				Message:    "duplicate credential value appears multiple times in this batch",
				HelpURL:    credentialHelpURL(prov.ID(), occurrence.key),
			})
		}
	}

	return results
}

func cloneValues(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func credentialHelpURL(providerID, key string) string {
	meta, ok := manifest.ProviderByID(manifest.ProviderID(providerID))
	if !ok {
		return ""
	}
	for _, credential := range meta.Credentials {
		if credential.Key == key {
			return credential.GetURL
		}
	}
	return ""
}

func redactedCredentialLabel(providerID, key, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return key
	}

	switch providerID {
	case "exa":
		return exa.RedactKey(value)
	case "context7":
		return context7.RedactKey(value)
	case "tavily":
		return tavily.RedactKey(value)
	case "github":
		return redactGitHubToken(value)
	default:
		return redact.Key(value)
	}
}

func redactGitHubToken(value string) string {
	switch {
	case strings.HasPrefix(value, "ghp_") && len(value) > len("ghp_")+8:
		suffix := value[len("ghp_"):]
		return "ghp_" + suffix[:4] + "..." + suffix[len(suffix)-4:]
	case strings.HasPrefix(value, "github_pat_") && len(value) > len("github_pat_")+8:
		suffix := value[len("github_pat_"):]
		return "github_pat_" + suffix[:4] + "..." + suffix[len(suffix)-4:]
	default:
		return redact.Key(value)
	}
}

func ProfilesFromValues(prov provider.MCPProvider, values map[string]string) []provider.CredentialProfile {
	if len(values) == 0 {
		return nil
	}
	return []provider.CredentialProfile{{
		ProviderID: prov.ID(),
		Values:     cloneValues(values),
	}}
}

func MissingRequiredCredentials(prov provider.MCPProvider, values map[string]string) error {
	report := Offline(Request{
		Provider: prov,
		Values:   values,
	})
	for _, result := range report.Results {
		if result.Status == StatusFailed {
			return fmt.Errorf("%s: %s", result.Key, result.Message)
		}
	}
	return nil
}
