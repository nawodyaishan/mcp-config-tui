package manifest

import (
	"fmt"
	"path/filepath"
	"strings"
)

func AllClients() []ClientManifest {
	return cloneClients(allClients)
}

func ClientByID(id ClientID) (ClientManifest, bool) {
	for _, client := range allClients {
		if client.ID == id {
			return cloneClient(client), true
		}
	}
	return ClientManifest{}, false
}

func ForPlatform(clients []ClientManifest, goos string) []ClientManifest {
	filtered := make([]ClientManifest, 0, len(clients))
	for _, client := range clients {
		if !supportsPlatform(client.Platforms, goos) {
			continue
		}
		next := cloneClient(client)
		next.Candidates = next.Candidates[:0]
		for _, candidate := range client.Candidates {
			if len(candidate.Platforms) == 0 || supportsPlatform(candidate.Platforms, goos) {
				next.Candidates = append(next.Candidates, cloneCandidate(candidate))
			}
		}
		filtered = append(filtered, next)
	}
	return filtered
}

func AllProviders() []ProviderMeta {
	return cloneProviders(allProviders)
}

func ProviderByID(id ProviderID) (ProviderMeta, bool) {
	for _, provider := range allProviders {
		if provider.ID == id {
			return cloneProvider(provider), true
		}
	}
	return ProviderMeta{}, false
}

func AllRuntimeRequirements() []RuntimeRequirement {
	result := make([]RuntimeRequirement, len(allRuntimeRequirements))
	for i, runtime := range allRuntimeRequirements {
		result[i] = cloneRuntime(runtime)
	}
	return result
}

func ExpandPath(template string, vars PathVars) (string, error) {
	if template == "" {
		return "", fmt.Errorf("empty path template")
	}
	if strings.Contains(template, "{{.Home}}") && vars.Home == "" {
		return "", fmt.Errorf("path template requires home")
	}
	if strings.Contains(template, "{{.Workspace}}") && vars.Workspace == "" {
		return "", fmt.Errorf("path template requires workspace")
	}

	result := strings.NewReplacer(
		"{{.Home}}", vars.Home,
		"{{.Workspace}}", vars.Workspace,
	).Replace(template)

	if strings.Contains(result, "{{.") {
		return "", fmt.Errorf("path template contains unsupported token: %s", template)
	}
	return filepath.Clean(result), nil
}

func supportsPlatform(platforms []string, goos string) bool {
	for _, platform := range platforms {
		if platform == goos {
			return true
		}
	}
	return false
}

func cloneClients(clients []ClientManifest) []ClientManifest {
	result := make([]ClientManifest, len(clients))
	for i, client := range clients {
		result[i] = cloneClient(client)
	}
	return result
}

func cloneClient(client ClientManifest) ClientManifest {
	return ClientManifest{
		ID:         client.ID,
		Name:       client.Name,
		Platforms:  append([]string(nil), client.Platforms...),
		Candidates: cloneCandidates(client.Candidates),
		Manager:    client.Manager,
		CLIName:    client.CLIName,
		DocsURL:    client.DocsURL,
		Warnings:   append([]ClientWarning(nil), client.Warnings...),
		Sources:    append([]SourceRef(nil), client.Sources...),
	}
}

func cloneCandidates(candidates []ConfigCandidate) []ConfigCandidate {
	result := make([]ConfigCandidate, len(candidates))
	for i, candidate := range candidates {
		result[i] = cloneCandidate(candidate)
	}
	return result
}

func cloneCandidate(candidate ConfigCandidate) ConfigCandidate {
	candidate.Platforms = append([]string(nil), candidate.Platforms...)
	return candidate
}

func cloneProviders(providers []ProviderMeta) []ProviderMeta {
	result := make([]ProviderMeta, len(providers))
	for i, provider := range providers {
		result[i] = cloneProvider(provider)
	}
	return result
}

func cloneProvider(provider ProviderMeta) ProviderMeta {
	next := ProviderMeta{
		ID:         provider.ID,
		Name:       provider.Name,
		DocsURL:    provider.DocsURL,
		RuntimeIDs: append([]string(nil), provider.RuntimeIDs...),
		Sources:    append([]SourceRef(nil), provider.Sources...),
	}
	if len(provider.Credentials) > 0 {
		next.Credentials = make([]CredentialAcquisition, len(provider.Credentials))
		copy(next.Credentials, provider.Credentials)
		for i := range next.Credentials {
			if provider.Credentials[i].LiveValidation == nil {
				continue
			}
			spec := *provider.Credentials[i].LiveValidation
			next.Credentials[i].LiveValidation = &spec
		}
	}
	return next
}

func cloneRuntime(runtime RuntimeRequirement) RuntimeRequirement {
	return RuntimeRequirement{
		ID:          runtime.ID,
		Name:        runtime.Name,
		Command:     runtime.Command,
		Args:        append([]string(nil), runtime.Args...),
		InstallURL:  runtime.InstallURL,
		RequiredFor: append([]string(nil), runtime.RequiredFor...),
	}
}
