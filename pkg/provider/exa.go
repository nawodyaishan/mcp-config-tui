package provider

import (
	"fmt"

	"github.com/nawodyaishan/universal-mcp-sync/pkg/exa"
)

type ExaProvider struct{}

func NewExaProvider() *ExaProvider {
	return &ExaProvider{}
}

func (p *ExaProvider) ID() string {
	return "exa"
}

func (p *ExaProvider) Name() string {
	return "Exa AI Search"
}

func (p *ExaProvider) Description() string {
	return "Web search and retrieval for AI agents."
}

func (p *ExaProvider) RequiredCredentials() []CredentialSpec {
	return []CredentialSpec{
		{
			Key:         "EXA_API_KEY",
			Label:       "Exa API Keys",
			Description: "Paste one or more UUID-style keys (one per line or key = \"...\" format)",
			Secret:      true,
			MultiValue:  true,
			Validator: func(s string) error {
				keys, err := exa.ParseKeys(s)
				if err != nil {
					return fmt.Errorf("invalid keys: %w", err)
				}
				if len(keys) == 0 {
					return fmt.Errorf("at least one valid Exa API key is required")
				}
				return nil
			},
		},
	}
}

func (p *ExaProvider) GenerateConfig(credentials map[string]string) (MCPConfig, error) {
	key := credentials["EXA_API_KEY"]
	url, err := exa.BuildURL(key, exa.DefaultTools)
	if err != nil {
		return MCPConfig{}, fmt.Errorf("build Exa URL: %w", err)
	}

	return MCPConfig{
		Type: TransportHTTP,
		URL:  url,
	}, nil
}

// ParseMultiValue implements MultiValueParser.
// It extracts multiple UUID-format Exa API keys from a single pasted text blob.
func (p *ExaProvider) ParseMultiValue(credentialKey string, raw string) ([]CredentialProfile, error) {
    if credentialKey != "EXA_API_KEY" {
        return nil, fmt.Errorf("ExaProvider: unknown multi-value credential key %q", credentialKey)
    }
    keys, err := exa.ParseKeys(raw)
    if err != nil {
        return nil, err
    }
    profiles := make([]CredentialProfile, len(keys))
    for i, k := range keys {
        profiles[i] = CredentialProfile{
            ProviderID: p.ID(),
            Values:     map[string]string{credentialKey: k},
            Label:      exa.RedactKey(k),
        }
    }
    return profiles, nil
}
