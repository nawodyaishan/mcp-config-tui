package provider

import (
	"fmt"

	"github.com/nawodyaishan/mcp-config-tui/internal/exa"
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

func (p *ExaProvider) RequiredCredentials() map[string]string {
	return map[string]string{
		"EXA_API_KEY": "Exa API Key (UUID format)",
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
