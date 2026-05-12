package provider

type KubernetesProvider struct{}

func NewKubernetesProvider() *KubernetesProvider { return &KubernetesProvider{} }

func (p *KubernetesProvider) ID() string   { return "kubernetes" }
func (p *KubernetesProvider) Name() string { return "Kubernetes" }
func (p *KubernetesProvider) Description() string {
	return "Read-only Kubernetes and OpenShift runtime state for AI agents."
}

func (p *KubernetesProvider) RequiredCredentials() []CredentialSpec {
	return nil
}

func (p *KubernetesProvider) GenerateConfig(credentials map[string]string) (MCPConfig, error) {
	return MCPConfig{
		Type:    TransportStdio,
		Command: "npx",
		Args:    []string{"-y", "kubernetes-mcp-server@latest", "--read-only"},
		Runtime: &PackageRuntime{Type: "npm"},
	}, nil
}
