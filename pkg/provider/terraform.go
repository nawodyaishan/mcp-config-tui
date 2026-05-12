package provider

type TerraformProvider struct{}

func NewTerraformProvider() *TerraformProvider { return &TerraformProvider{} }

func (p *TerraformProvider) ID() string   { return "terraform" }
func (p *TerraformProvider) Name() string { return "Terraform" }
func (p *TerraformProvider) Description() string {
	return "Terraform Registry and HCP Terraform context for Infrastructure as Code workflows."
}

func (p *TerraformProvider) RequiredCredentials() []CredentialSpec {
	return nil
}

func (p *TerraformProvider) GenerateConfig(credentials map[string]string) (MCPConfig, error) {
	return MCPConfig{
		Type:    TransportStdio,
		Command: "docker",
		Args: []string{
			"run",
			"-i",
			"--rm",
			"-e",
			"ENABLE_TF_OPERATIONS=false",
			"hashicorp/terraform-mcp-server:0.5.2",
		},
		Runtime: &PackageRuntime{Type: "oci"},
	}, nil
}
