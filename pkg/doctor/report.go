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
			for _, issue := range client.Issues {
				_, _ = fmt.Fprintf(&builder, "  issue: %s\n", issue)
			}
			for _, warning := range client.Warnings {
				_, _ = fmt.Fprintf(&builder, "  warning: %s\n", warning)
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
			_, _ = fmt.Fprintf(&builder, "\n")
		}
	}

	return strings.TrimRight(builder.String(), "\n")
}
