package provider

import (
	"fmt"

	"github.com/nawodyaishan/universal-mcp-sync/pkg/context7"
)

type Context7Provider struct{}

func NewContext7Provider() *Context7Provider { return &Context7Provider{} }

func (p *Context7Provider) ID() string          { return "context7" }
func (p *Context7Provider) Name() string        { return "Context7" }
func (p *Context7Provider) Description() string {
	return "Up-to-date library documentation and code examples for AI agents."
}

func (p *Context7Provider) RequiredCredentials() []CredentialSpec {
	return []CredentialSpec{
		{
			Key:         "CONTEXT7_API_KEY",
			Label:       "Context7 API Key",
			Description: "Get your key at context7.com/dashboard. Format: ctx7sk_...",
			Secret:      true,
			MultiValue:  false,
			Validator: func(s string) error {
				_, err := context7.ParseKey(s)
				return err
			},
		},
	}
}

func (p *Context7Provider) GenerateConfig(credentials map[string]string) (MCPConfig, error) {
	key := credentials["CONTEXT7_API_KEY"]
	if _, err := context7.ParseKey(key); err != nil {
		return MCPConfig{}, fmt.Errorf("invalid Context7 API key: %w", err)
	}
	return MCPConfig{
		Type:    TransportStreamableHTTP,
		URL:     context7.Endpoint,
		Headers: map[string]string{context7.HeaderName: key},
		BridgeOverride: &BridgeConfig{
			Command: "npx",
			Args:    []string{"-y", "@upstash/context7-mcp", "--api-key", "{header:CONTEXT7_API_KEY}"},
		},
	}, nil
}