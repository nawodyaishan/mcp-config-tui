package provider

type PlaywrightProvider struct{}

func NewPlaywrightProvider() *PlaywrightProvider { return &PlaywrightProvider{} }

func (p *PlaywrightProvider) ID() string   { return "playwright" }
func (p *PlaywrightProvider) Name() string { return "Playwright" }
func (p *PlaywrightProvider) Description() string {
	return "Browser automation for AI agents through structured accessibility snapshots."
}

func (p *PlaywrightProvider) RequiredCredentials() []CredentialSpec {
	return nil
}

func (p *PlaywrightProvider) GenerateConfig(credentials map[string]string) (MCPConfig, error) {
	return MCPConfig{
		Type:    TransportStdio,
		Command: "npx",
		Args:    []string{"@playwright/mcp@latest"},
		Runtime: &PackageRuntime{Type: "npm"},
	}, nil
}
