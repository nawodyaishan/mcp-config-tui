package validate

import (
	"strings"
	"testing"
)

func TestFormatReportIncludesHelpURLAndNoRawToken(t *testing.T) {
	raw := "ghp_" + strings.Repeat("a", 36)
	report := Report{
		ProviderID: "github",
		Results: []Result{{
			ProviderID: "github",
			Key:        "GITHUB_PERSONAL_ACCESS_TOKEN",
			Label:      "ghp_aaaa...aaaa",
			Status:     StatusFailed,
			Mode:       ModeOffline,
			Message:    "missing required credential",
			HelpURL:    "https://github.com/settings/tokens",
		}},
	}

	formatted := FormatReport(report)
	if !strings.Contains(formatted, "get key: https://github.com/settings/tokens") {
		t.Fatalf("expected help url, got:\n%s", formatted)
	}
	if strings.Contains(formatted, raw) {
		t.Fatalf("formatted output leaked raw token:\n%s", formatted)
	}
}
