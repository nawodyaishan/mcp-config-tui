package validate

import (
	"encoding/json"
	"testing"
)

func TestReportJSONRoundTrip(t *testing.T) {
	report := Report{
		ProviderID: "github",
		Live:       true,
		Results: []Result{{
			ProviderID: "github",
			Key:        "GITHUB_PERSONAL_ACCESS_TOKEN",
			Label:      "ghp_aaaa...aaaa",
			Status:     StatusOK,
			Mode:       ModeLive,
			Message:    "live validation succeeded",
			Cached:     true,
			QuotaCost:  false,
			HelpURL:    "https://github.com/settings/tokens",
		}},
		Warnings: []string{"cache unavailable"},
	}

	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}

	var decoded Report
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}

	if decoded.ProviderID != report.ProviderID || decoded.Live != report.Live {
		t.Fatalf("unexpected report round-trip: %#v", decoded)
	}
	if len(decoded.Results) != 1 || decoded.Results[0].Status != StatusOK || decoded.Results[0].Mode != ModeLive {
		t.Fatalf("unexpected results after round-trip: %#v", decoded.Results)
	}
	if len(decoded.Warnings) != 1 || decoded.Warnings[0] != "cache unavailable" {
		t.Fatalf("unexpected warnings after round-trip: %#v", decoded.Warnings)
	}
}
