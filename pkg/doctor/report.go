package doctor

import (
	"encoding/json"
	"fmt"
	"strings"
)

func MarshalReportJSON(report Report) ([]byte, error) {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func FormatReport(report Report) string {
	return formatReport(report, false)
}

func FormatVerboseReport(report Report) string {
	return formatReport(report, true)
}

func formatReport(report Report, verbose bool) string {
	var builder strings.Builder

	_, _ = fmt.Fprintf(&builder, "Doctor report\n")
	_, _ = fmt.Fprintf(&builder, "=============\n")
	_, _ = fmt.Fprintf(&builder, "platform: %s\n", report.Platform)
	_, _ = fmt.Fprintf(&builder, "clients: %d\n", len(report.Clients))

	if len(report.Warnings) > 0 {
		_, _ = fmt.Fprintf(&builder, "\nWarnings\n")
		_, _ = fmt.Fprintf(&builder, "--------\n")
		for _, warning := range report.Warnings {
			_, _ = fmt.Fprintf(&builder, "- %s\n", warning)
		}
	}

	if len(report.Clients) > 0 {
		_, _ = fmt.Fprintf(&builder, "\nClients\n")
		_, _ = fmt.Fprintf(&builder, "-------\n")
		for _, client := range report.Clients {
			status := string(client.Confidence)
			path := client.EffectivePath
			if path == "" && len(client.Candidates) > 0 {
				path = client.Candidates[0].Path
			}
			if path == "" {
				path = "not detected"
			}
			providers := "none"
			if len(client.ConfiguredProviders) > 0 {
				providers = strings.Join(client.ConfiguredProviders, ", ")
			}
			_, _ = fmt.Fprintf(&builder, "- %s [%s] path=%s providers=%s\n", client.Name, status, path, providers)
			if verbose {
				for _, candidate := range client.Candidates {
					_, _ = fmt.Fprintf(&builder, "  candidate: label=%s scope=%s exists=%t writable=%t path=%s\n", candidate.Label, candidate.Scope, candidate.Exists, candidate.Writable, candidate.Path)
					if candidate.IsSymlink {
						_, _ = fmt.Fprintf(&builder, "    symlink: %s\n", candidate.Resolved)
					}
					if candidate.ParseError != "" {
						_, _ = fmt.Fprintf(&builder, "    parse_error: %s\n", candidate.ParseError)
					}
					if len(candidate.Providers) > 0 {
						_, _ = fmt.Fprintf(&builder, "    providers: %s\n", strings.Join(candidate.Providers, ", "))
					}
				}
			}
			for _, issue := range client.Issues {
				_, _ = fmt.Fprintf(&builder, "  issue: %s\n", issue)
			}
			for _, warning := range client.Warnings {
				_, _ = fmt.Fprintf(&builder, "  warning: %s\n", warning)
			}
			if verbose {
				for _, hint := range client.MigrationHints {
					_, _ = fmt.Fprintf(&builder, "  migration: %s -> %s (%s)\n", hint.FromID, hint.ToID, hint.Reason)
				}
			}
		}
	}

	if len(report.Runtimes) > 0 {
		_, _ = fmt.Fprintf(&builder, "\nRuntimes\n")
		_, _ = fmt.Fprintf(&builder, "--------\n")
		for _, runtime := range report.Runtimes {
			state := "ok"
			if !runtime.Available {
				state = "missing"
			}
			if runtime.Error != "" && runtime.Available {
				state = "warning"
			}
			_, _ = fmt.Fprintf(&builder, "- %s [%s]", runtime.Name, state)
			if runtime.Path != "" {
				_, _ = fmt.Fprintf(&builder, " path=%s", runtime.Path)
			}
			if runtime.Version != "" {
				_, _ = fmt.Fprintf(&builder, " version=%s", runtime.Version)
			}
			if runtime.Error != "" {
				_, _ = fmt.Fprintf(&builder, " error=%s", runtime.Error)
			}
			if verbose && len(runtime.RequiredFor) > 0 {
				_, _ = fmt.Fprintf(&builder, " required_for=%s", strings.Join(runtime.RequiredFor, ","))
			}
			_, _ = fmt.Fprintf(&builder, "\n")
		}
	}

	return strings.TrimRight(builder.String(), "\n")
}
