package provider

import (
	"fmt"

	"github.com/nawodyaishan/universal-mcp-sync/pkg/tavily"
)

type TavilyProvider struct{}

func NewTavilyProvider() *TavilyProvider { return &TavilyProvider{} }

func (p *TavilyProvider) ID() string   { return "tavily" }
func (p *TavilyProvider) Name() string { return "Tavily Search" }
func (p *TavilyProvider) Description() string {
	return "Real-time web search and data extraction for AI agents."
}

func (p *TavilyProvider) RequiredCredentials() []CredentialSpec {
	return []CredentialSpec{
		{
			Key:         "TAVILY_API_KEY",
			Label:       "Tavily API Key",
			Description: "Get your key at tavily.com. Format: tvly-...",
			Secret:      true,
			MultiValue:  false,
			Validator: func(s string) error {
				_, err := tavily.ParseKey(s)
				return err
			},
		},
	}
}

func (p *TavilyProvider) GenerateConfig(credentials map[string]string) (MCPConfig, error) {
	key := credentials["TAVILY_API_KEY"]
	if _, err := tavily.ParseKey(key); err != nil {
		return MCPConfig{}, fmt.Errorf("invalid Tavily API key: %w", err)
	}
	return MCPConfig{
		Type:    TransportStdio,
		Command: "npx",
		Args:    []string{"-y", "tavily-mcp@latest"},
		Env:     map[string]string{"TAVILY_API_KEY": key},
		Runtime: &PackageRuntime{Type: "npm"},
	}, nil
}
