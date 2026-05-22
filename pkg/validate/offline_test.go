package validate

import (
	"strings"
	"testing"

	"github.com/nawodyaishan/universal-mcp-sync/pkg/provider"
)

func TestOfflineExaValidDoesNotLeakRawKey(t *testing.T) {
	key := "11111111-1111-1111-1111-111111111111"
	report := Offline(Request{
		Provider: provider.NewExaProvider(),
		Values: map[string]string{
			"EXA_API_KEY": key,
		},
	})
	if report.HasFailures() {
		t.Fatalf("expected valid Exa key to pass, got %#v", report.Results)
	}

	formatted := FormatReport(report)
	if strings.Contains(formatted, key) {
		t.Fatalf("formatted report leaked raw key:\n%s", formatted)
	}
	data, err := MarshalReportJSON(report)
	if err != nil {
		t.Fatalf("MarshalReportJSON returned error: %v", err)
	}
	if strings.Contains(string(data), key) {
		t.Fatalf("validation json leaked raw key:\n%s", string(data))
	}
}

func TestOfflineGitHubInvalidFails(t *testing.T) {
	report := Offline(Request{
		Provider: provider.NewGitHubProvider(),
		Values: map[string]string{
			"GITHUB_PERSONAL_ACCESS_TOKEN": "ghp_tooshort",
		},
	})
	if !report.HasFailures() {
		t.Fatalf("expected invalid GitHub token to fail, got %#v", report.Results)
	}
}

func TestOfflineContext7ValidAndInvalid(t *testing.T) {
	valid := Offline(Request{
		Provider: provider.NewContext7Provider(),
		Values: map[string]string{
			"CONTEXT7_API_KEY": "ctx7sk-demo1234",
		},
	})
	if valid.HasFailures() {
		t.Fatalf("expected valid Context7 key to pass, got %#v", valid.Results)
	}

	invalid := Offline(Request{
		Provider: provider.NewContext7Provider(),
		Values: map[string]string{
			"CONTEXT7_API_KEY": "ctx7-invalid",
		},
	})
	if !invalid.HasFailures() {
		t.Fatalf("expected invalid Context7 key to fail, got %#v", invalid.Results)
	}
}

func TestOfflineMissingContext7IncludesHelpURL(t *testing.T) {
	report := Offline(Request{
		Provider: provider.NewContext7Provider(),
	})
	if !report.HasFailures() {
		t.Fatalf("expected missing Context7 key to fail, got %#v", report.Results)
	}
	if len(report.Results) != 1 || report.Results[0].HelpURL == "" {
		t.Fatalf("expected help url for missing credential, got %#v", report.Results)
	}
}

func TestOfflineTavilyValidAndInvalid(t *testing.T) {
	rawKey := "tvly-demo12345678"
	valid := Offline(Request{
		Provider: provider.NewTavilyProvider(),
		Values: map[string]string{
			"TAVILY_API_KEY": rawKey,
		},
	})
	if valid.HasFailures() {
		t.Fatalf("expected valid Tavily key to pass, got %#v", valid.Results)
	}

	formatted := FormatReport(valid)
	if strings.Contains(formatted, rawKey) {
		t.Fatalf("formatted report leaked Tavily key:\n%s", formatted)
	}

	invalid := Offline(Request{
		Provider: provider.NewTavilyProvider(),
		Values: map[string]string{
			"TAVILY_API_KEY": "invalid-key",
		},
	})
	if !invalid.HasFailures() {
		t.Fatalf("expected invalid Tavily key to fail, got %#v", invalid.Results)
	}
}

func TestOfflineProfilesWarnOnDuplicateValues(t *testing.T) {
	key := "11111111-1111-1111-1111-111111111111"
	report := OfflineProfiles(provider.NewExaProvider(), []provider.CredentialProfile{
		{
			ProviderID: "exa",
			Values: map[string]string{
				"EXA_API_KEY": key,
			},
		},
		{
			ProviderID: "exa",
			Values: map[string]string{
				"EXA_API_KEY": key,
			},
		},
	})

	foundWarning := false
	for _, result := range report.Results {
		if result.Status == StatusWarning && strings.Contains(result.Message, "duplicate credential value") {
			foundWarning = true
		}
		if strings.Contains(result.Message, key) || strings.Contains(result.Label, key) {
			t.Fatalf("duplicate warning leaked raw key: %#v", result)
		}
	}
	if !foundWarning {
		t.Fatalf("expected duplicate warning, got %#v", report.Results)
	}
}

func TestOfflineNoCredentialsRequiredIsOK(t *testing.T) {
	report := Offline(Request{
		Provider: provider.NewPlaywrightProvider(),
	})
	if report.HasFailures() {
		t.Fatalf("expected no-credential provider to pass, got %#v", report.Results)
	}
	if len(report.Results) != 1 || report.Results[0].Status != StatusOK {
		t.Fatalf("unexpected no-credential report: %#v", report.Results)
	}
}
