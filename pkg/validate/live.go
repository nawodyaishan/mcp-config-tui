package validate

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/nawodyaishan/universal-mcp-sync/pkg/manifest"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/provider"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/redact"
)

type Service struct {
	HomeDir      string
	Now          func() time.Time
	HTTPClient   HTTPDoer
	Cache        Store
	MetaResolver func(providerID string) (manifest.ProviderMeta, bool)
}

func NewService(homeDir string) (Service, error) {
	cache, err := NewStore(homeDir)
	if err != nil {
		return Service{}, err
	}

	now := time.Now
	cache.Now = now

	return Service{
		HomeDir:    cache.HomeDir,
		Now:        now,
		HTTPClient: http.DefaultClient,
		Cache:      cache,
		MetaResolver: func(providerID string) (manifest.ProviderMeta, bool) {
			return manifest.ProviderByID(manifest.ProviderID(providerID))
		},
	}, nil
}

func (s Service) Validate(ctx context.Context, req Request) (Report, error) {
	if req.Provider == nil {
		return Report{}, fmt.Errorf("missing provider")
	}

	profiles := ProfilesFromValues(req.Provider, req.Values)
	if len(profiles) == 0 {
		profiles = []provider.CredentialProfile{{
			ProviderID: req.Provider.ID(),
			Values:     cloneValues(req.Values),
		}}
	}
	return s.ValidateProfiles(ctx, req.Provider, profiles, req.Live)
}

func (s Service) ValidateProfiles(ctx context.Context, prov provider.MCPProvider, profiles []provider.CredentialProfile, live bool) (Report, error) {
	report := OfflineProfiles(prov, profiles)
	report.Live = live
	if !live || report.HasFailures() {
		return report, nil
	}

	liveResults, warnings := s.liveResults(ctx, prov, profiles)
	report.Results = append(report.Results, liveResults...)
	report.Warnings = append(report.Warnings, warnings...)
	return report, nil
}

func (s Service) liveResults(ctx context.Context, prov provider.MCPProvider, profiles []provider.CredentialProfile) ([]Result, []string) {
	meta, ok := s.resolveProviderMeta(prov.ID())
	if !ok {
		return []Result{{
			ProviderID: prov.ID(),
			Label:      prov.Name(),
			Status:     StatusSkipped,
			Mode:       ModeLive,
			Message:    "live validation metadata is unavailable",
		}}, nil
	}

	liveCredential, ok := firstLiveCredential(meta)
	if !ok {
		return []Result{{
			ProviderID: prov.ID(),
			Label:      meta.Name,
			Status:     StatusSkipped,
			Mode:       ModeLive,
			Message:    "live validation is not supported for this provider",
		}}, nil
	}

	client := s.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}

	now := time.Now
	if s.Now != nil {
		now = s.Now
	}
	s.Cache.Now = now

	results := make([]Result, 0, len(profiles))
	warnings := make([]string, 0)

	for _, profile := range profiles {
		value := strings.TrimSpace(profile.Values[liveCredential.Key])
		if value == "" {
			continue
		}

		label := strings.TrimSpace(profile.Label)
		if label == "" {
			label = redactedCredentialLabel(prov.ID(), liveCredential.Key, value)
		}

		cacheKey := CacheKey(prov.ID(), liveCredential.Key, label)
		if entry, ok, err := s.Cache.Get(cacheKey); err != nil {
			warnings = append(warnings, "credential cache: "+redact.Text(err.Error()))
		} else if ok {
			results = append(results, Result{
				ProviderID: prov.ID(),
				Key:        liveCredential.Key,
				Label:      label,
				Status:     entry.Status,
				Mode:       ModeLive,
				Message:    entry.Message,
				Cached:     true,
				QuotaCost:  !liveCredential.LiveValidation.QuotaSafe,
				HelpURL:    liveCredential.GetURL,
			})
			continue
		}

		result := runLiveValidationRequest(ctx, client, prov.ID(), value, label, liveCredential)
		results = append(results, result)

		if shouldCacheResult(result.Status) {
			entry := CacheEntry{
				Status:     result.Status,
				Message:    result.Message,
				CachedAt:   now().UTC(),
				ExpiresAt:  now().UTC().Add(validationCacheTTL),
				ProviderID: prov.ID(),
				Key:        liveCredential.Key,
				KeyLabel:   label,
			}
			if err := s.Cache.Put(cacheKey, entry); err != nil {
				warnings = append(warnings, "credential cache: "+redact.Text(err.Error()))
			}
		}
	}

	return results, warnings
}

func runLiveValidationRequest(ctx context.Context, client HTTPDoer, providerID, value, label string, credential manifest.CredentialAcquisition) Result {
	result := Result{
		ProviderID: providerID,
		Key:        credential.Key,
		Label:      label,
		Status:     StatusSkipped,
		Mode:       ModeLive,
		QuotaCost:  !credential.LiveValidation.QuotaSafe,
		HelpURL:    credential.GetURL,
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	request, err := http.NewRequestWithContext(timeoutCtx, credential.LiveValidation.Method, credential.LiveValidation.URL, nil)
	if err != nil {
		result.Message = "live validation request could not be created"
		return result
	}

	authHeader := strings.ReplaceAll(credential.LiveValidation.AuthHeader, "{key}", value)
	headerParts := strings.SplitN(authHeader, ":", 2)
	if len(headerParts) == 2 {
		request.Header.Set(strings.TrimSpace(headerParts[0]), strings.TrimSpace(headerParts[1]))
	}
	request.Header.Set("Accept", "application/json")

	response, err := client.Do(request)
	if err != nil {
		if timeoutCtx.Err() == context.DeadlineExceeded || errors.Is(err, context.DeadlineExceeded) {
			result.Message = "validation timed out"
			return result
		}
		result.Message = "live validation unavailable: " + redact.Text(err.Error())
		return result
	}
	defer func() {
		_ = response.Body.Close()
	}()

	switch response.StatusCode {
	case http.StatusOK:
		result.Status = StatusOK
		result.Message = "live validation succeeded"
	case http.StatusUnauthorized, http.StatusForbidden:
		result.Status = StatusFailed
		result.Message = "authentication rejected by remote service"
	case http.StatusTooManyRequests:
		result.Status = StatusSkipped
		result.Message = "validation rate limited"
	default:
		if response.StatusCode >= 500 {
			result.Status = StatusSkipped
			result.Message = fmt.Sprintf("live validation unavailable (HTTP %d)", response.StatusCode)
			return result
		}
		result.Status = StatusWarning
		result.Message = fmt.Sprintf("unexpected live validation status: HTTP %d", response.StatusCode)
	}

	return result
}

func firstLiveCredential(meta manifest.ProviderMeta) (manifest.CredentialAcquisition, bool) {
	for _, credential := range meta.Credentials {
		if credential.LiveValidation != nil {
			return credential, true
		}
	}
	return manifest.CredentialAcquisition{}, false
}

func shouldCacheResult(status Status) bool {
	switch status {
	case StatusOK, StatusFailed:
		return true
	default:
		return false
	}
}

func (s Service) resolveProviderMeta(providerID string) (manifest.ProviderMeta, bool) {
	if s.MetaResolver != nil {
		return s.MetaResolver(providerID)
	}
	return manifest.ProviderByID(manifest.ProviderID(providerID))
}
