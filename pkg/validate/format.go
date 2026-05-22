package validate

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nawodyaishan/universal-mcp-sync/pkg/redact"
)

func FormatReport(report Report) string {
	var builder strings.Builder
	builder.WriteString("Credential validation\n")
	builder.WriteString("=====================\n")
	fmt.Fprintf(&builder, "provider: %s\n", report.ProviderID)

	for _, warning := range report.Warnings {
		builder.WriteString("warning: " + redact.Text(warning) + "\n")
	}

	for _, result := range report.Results {
		label := result.Key
		if result.Label != "" && result.Label != result.Key {
			label = result.Key + " (" + result.Label + ")"
		}

		message := redact.Text(result.Message)
		if result.Cached {
			message += " (cached)"
		}
		fmt.Fprintf(&builder, "- %s [%s] %s\n", label, result.Status, message)
		if result.HelpURL != "" && result.Status == StatusFailed {
			fmt.Fprintf(&builder, "  get key: %s\n", result.HelpURL)
		}
	}

	return strings.TrimRight(builder.String(), "\n")
}

func MarshalReportJSON(report Report) ([]byte, error) {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal validation json: %w", err)
	}
	return append(data, '\n'), nil
}
