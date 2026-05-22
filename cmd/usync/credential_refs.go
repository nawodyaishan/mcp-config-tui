package main

import (
	"strings"

	"github.com/nawodyaishan/universal-mcp-sync/pkg/app"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/manifest"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/provider"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/validate"
)

func buildCredentialRefs(prov provider.MCPProvider, profiles []provider.CredentialProfile) []app.CredentialRef {
	if len(profiles) == 0 {
		return nil
	}

	envByKey := credentialEnvVars(prov.ID())
	refs := make([]app.CredentialRef, 0)
	for _, profile := range profiles {
		for _, spec := range prov.RequiredCredentials() {
			value := strings.TrimSpace(profile.Values[spec.Key])
			if value == "" {
				continue
			}

			label := strings.TrimSpace(profile.Label)
			if label == "" {
				label = validate.RedactedCredentialLabel(prov.ID(), spec.Key, value)
			}

			refs = append(refs, app.CredentialRef{
				Key:    spec.Key,
				Label:  label,
				EnvVar: envByKey[spec.Key],
			})
		}
	}
	return refs
}

func credentialEnvVars(providerID string) map[string]string {
	meta, ok := manifest.ProviderByID(manifest.ProviderID(providerID))
	if !ok {
		return nil
	}

	envByKey := make(map[string]string, len(meta.Credentials))
	for _, credential := range meta.Credentials {
		envByKey[credential.Key] = credential.EnvVar
	}
	return envByKey
}
