package provider

import (
	"fmt"
	"regexp"
)

// githubPATRE matches the three known GitHub PAT formats:
// - Classic: 40-char hex (legacy)
// - Fine-grained: ghp_ prefix + 36 alphanumeric chars
// - Fine-grained v2: github_pat_ prefix + long alphanumeric+underscore string
var githubPATRE = regexp.MustCompile(
	`^(?:[0-9a-f]{40}|ghp_[A-Za-z0-9]{36}|github_pat_[A-Za-z0-9_]{59,})$`,
)

// GitHubProvider installs the official MCP GitHub server via npx.
// Reference: https://github.com/modelcontextprotocol/servers/tree/main/src/github
type GitHubProvider struct{}

func NewGitHubProvider() *GitHubProvider { return &GitHubProvider{} }

func (p *GitHubProvider) ID() string   { return "github" }
func (p *GitHubProvider) Name() string { return "GitHub" }
func (p *GitHubProvider) Description() string {
	return "GitHub repository, issue, and PR management via the official MCP server."
}

func (p *GitHubProvider) RequiredCredentials() []CredentialSpec {
	return []CredentialSpec{
		{
			Key:         "GITHUB_PERSONAL_ACCESS_TOKEN",
			Label:       "GitHub Personal Access Token",
			Description: "Create at github.com/settings/tokens. Requires 'repo' scope. Format: ghp_... or github_pat_...",
			Secret:      true,
			MultiValue:  false,
			Validator: func(s string) error {
				if s == "" {
					return fmt.Errorf("token is required")
				}
				if !githubPATRE.MatchString(s) {
					return fmt.Errorf("unrecognised GitHub PAT format (expected ghp_..., github_pat_..., or 40-char hex)")
				}
				return nil
			},
		},
	}
}

func (p *GitHubProvider) GenerateConfig(credentials map[string]string) (MCPConfig, error) {
	pat := credentials["GITHUB_PERSONAL_ACCESS_TOKEN"]
	if pat == "" {
		return MCPConfig{}, fmt.Errorf("missing GITHUB_PERSONAL_ACCESS_TOKEN")
	}
	return MCPConfig{
		Type:    TransportStdio,
		Command: "npx",
		Args:    []string{"-y", "@modelcontextprotocol/server-github"},
		Env:     map[string]string{"GITHUB_PERSONAL_ACCESS_TOKEN": pat},
		Runtime: &PackageRuntime{Type: "npm"},
	}, nil
}
