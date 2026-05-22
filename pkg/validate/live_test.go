package validate

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/nawodyaishan/universal-mcp-sync/pkg/manifest"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/provider"
)

type countingDoer struct {
	calls int
	do    func(req *http.Request) (*http.Response, error)
}

func (d *countingDoer) Do(req *http.Request) (*http.Response, error) {
	d.calls++
	return d.do(req)
}

func TestRunLiveValidationRequestHandlesStatusesAndTimeout(t *testing.T) {
	okServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer okServer.Close()

	credential := manifest.CredentialAcquisition{
		Key:    "GITHUB_PERSONAL_ACCESS_TOKEN",
		GetURL: "https://github.com/settings/tokens",
		LiveValidation: &manifest.LiveValidationSpec{
			Method:     http.MethodGet,
			URL:        okServer.URL,
			AuthHeader: "Authorization: Bearer {key}",
			QuotaSafe:  true,
		},
	}
	token := "ghp_" + strings.Repeat("a", 36)
	okResult := runLiveValidationRequest(context.Background(), okServer.Client(), "github", token, "ghp_aaaa...aaaa", credential)
	if okResult.Status != StatusOK {
		t.Fatalf("expected OK live result, got %#v", okResult)
	}

	unauthorizedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer unauthorizedServer.Close()
	credential.LiveValidation.URL = unauthorizedServer.URL
	failedResult := runLiveValidationRequest(context.Background(), unauthorizedServer.Client(), "github", token, "ghp_aaaa...aaaa", credential)
	if failedResult.Status != StatusFailed {
		t.Fatalf("expected failed auth result, got %#v", failedResult)
	}

	timeoutResult := runLiveValidationRequest(context.Background(), &countingDoer{
		do: func(req *http.Request) (*http.Response, error) {
			return nil, context.DeadlineExceeded
		},
	}, "github", token, "ghp_aaaa...aaaa", credential)
	if timeoutResult.Status != StatusSkipped || !strings.Contains(timeoutResult.Message, "timed out") {
		t.Fatalf("expected timeout skip result, got %#v", timeoutResult)
	}

	networkResult := runLiveValidationRequest(context.Background(), &countingDoer{
		do: func(req *http.Request) (*http.Response, error) {
			return nil, errors.New("dial tcp 127.0.0.1:1: connect refused")
		},
	}, "github", token, "ghp_aaaa...aaaa", credential)
	if networkResult.Status != StatusSkipped {
		t.Fatalf("expected network error to skip, got %#v", networkResult)
	}
}

func TestRunLiveValidationRequestTavilyUsesBearerAuthAndHandlesRateLimit(t *testing.T) {
	rawKey := "tvly-demo1234"
	var authHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	credential := manifest.CredentialAcquisition{
		Key:    "TAVILY_API_KEY",
		GetURL: "https://app.tavily.com/home",
		LiveValidation: &manifest.LiveValidationSpec{
			Method:     http.MethodGet,
			URL:        server.URL,
			AuthHeader: "Authorization: Bearer {key}",
			QuotaSafe:  true,
		},
	}

	result := runLiveValidationRequest(context.Background(), server.Client(), "tavily", rawKey, "tvly-demo...1234", credential)
	if authHeader != "Bearer "+rawKey {
		t.Fatalf("expected bearer auth header, got %q", authHeader)
	}
	if result.Status != StatusSkipped || !strings.Contains(result.Message, "rate limited") {
		t.Fatalf("expected rate-limit skip result, got %#v", result)
	}
	if strings.Contains(result.Message, rawKey) || strings.Contains(result.Label, rawKey) {
		t.Fatalf("live result leaked raw Tavily key: %#v", result)
	}
}

func TestServiceUsesCachedLiveResultWithoutHTTPCall(t *testing.T) {
	homeDir := t.TempDir()
	service, err := NewService(homeDir)
	if err != nil {
		t.Fatalf("NewService returned error: %v", err)
	}
	now := time.Date(2026, time.May, 22, 10, 0, 0, 0, time.UTC)
	service.Now = func() time.Time { return now }
	service.Cache.Now = service.Now
	service.MetaResolver = func(providerID string) (manifest.ProviderMeta, bool) {
		return manifest.ProviderMeta{
			ID:   manifest.ProviderGitHub,
			Name: "GitHub",
			Credentials: []manifest.CredentialAcquisition{{
				Key:    "GITHUB_PERSONAL_ACCESS_TOKEN",
				GetURL: "https://github.com/settings/tokens",
				LiveValidation: &manifest.LiveValidationSpec{
					Method:     http.MethodGet,
					URL:        "https://example.invalid/live",
					AuthHeader: "Authorization: Bearer {key}",
					QuotaSafe:  true,
				},
			}},
		}, true
	}

	doer := &countingDoer{
		do: func(req *http.Request) (*http.Response, error) {
			t.Fatal("HTTP client should not be called when cache is fresh")
			return nil, nil
		},
	}
	service.HTTPClient = doer

	label := "ghp_aaaa...aaaa"
	cacheKey := CacheKey("github", "GITHUB_PERSONAL_ACCESS_TOKEN", label)
	if err := service.Cache.Put(cacheKey, CacheEntry{
		Status:     StatusOK,
		Message:    "live validation succeeded",
		CachedAt:   now,
		ExpiresAt:  now.Add(validationCacheTTL),
		ProviderID: "github",
		Key:        "GITHUB_PERSONAL_ACCESS_TOKEN",
		KeyLabel:   label,
	}); err != nil {
		t.Fatalf("Put returned error: %v", err)
	}

	report, err := service.ValidateProfiles(context.Background(), provider.NewGitHubProvider(), []provider.CredentialProfile{{
		ProviderID: "github",
		Values: map[string]string{
			"GITHUB_PERSONAL_ACCESS_TOKEN": "ghp_" + strings.Repeat("a", 36),
		},
		Label: label,
	}}, true)
	if err != nil {
		t.Fatalf("ValidateProfiles returned error: %v", err)
	}
	if doer.calls != 0 {
		t.Fatalf("expected no live HTTP call, got %d", doer.calls)
	}

	foundCached := false
	for _, result := range report.Results {
		if result.Mode == ModeLive && result.Cached {
			foundCached = true
		}
	}
	if !foundCached {
		t.Fatalf("expected cached live result, got %#v", report.Results)
	}
}

func TestServiceRefetchesWhenCacheExpired(t *testing.T) {
	homeDir := t.TempDir()
	service, err := NewService(homeDir)
	if err != nil {
		t.Fatalf("NewService returned error: %v", err)
	}
	now := time.Date(2026, time.May, 22, 10, 0, 0, 0, time.UTC)
	service.Now = func() time.Time { return now }
	service.Cache.Now = service.Now

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	service.MetaResolver = func(providerID string) (manifest.ProviderMeta, bool) {
		return manifest.ProviderMeta{
			ID:   manifest.ProviderGitHub,
			Name: "GitHub",
			Credentials: []manifest.CredentialAcquisition{{
				Key:    "GITHUB_PERSONAL_ACCESS_TOKEN",
				GetURL: "https://github.com/settings/tokens",
				LiveValidation: &manifest.LiveValidationSpec{
					Method:     http.MethodGet,
					URL:        server.URL,
					AuthHeader: "Authorization: Bearer {key}",
					QuotaSafe:  true,
				},
			}},
		}, true
	}
	service.HTTPClient = server.Client()

	label := "ghp_aaaa...aaaa"
	cacheKey := CacheKey("github", "GITHUB_PERSONAL_ACCESS_TOKEN", label)
	if err := service.Cache.Put(cacheKey, CacheEntry{
		Status:     StatusOK,
		Message:    "live validation succeeded",
		CachedAt:   now.Add(-48 * time.Hour),
		ExpiresAt:  now.Add(-24 * time.Hour),
		ProviderID: "github",
		Key:        "GITHUB_PERSONAL_ACCESS_TOKEN",
		KeyLabel:   label,
	}); err != nil {
		t.Fatalf("Put returned error: %v", err)
	}

	report, err := service.ValidateProfiles(context.Background(), provider.NewGitHubProvider(), []provider.CredentialProfile{{
		ProviderID: "github",
		Values: map[string]string{
			"GITHUB_PERSONAL_ACCESS_TOKEN": "ghp_" + strings.Repeat("a", 36),
		},
		Label: label,
	}}, true)
	if err != nil {
		t.Fatalf("ValidateProfiles returned error: %v", err)
	}
	if callCount != 1 {
		t.Fatalf("expected 1 live request after expired cache, got %d", callCount)
	}
	if report.HasFailures() {
		t.Fatalf("expected successful refetch, got %#v", report.Results)
	}
}
