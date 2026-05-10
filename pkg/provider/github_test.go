package provider_test

import (
    "strings"
    "testing"

    "github.com/nawodyaishan/universal-mcp-sync/pkg/provider"
)

func TestGitHubProvider_ID(t *testing.T) {
    p := provider.NewGitHubProvider()
    if p.ID() != "github" {
        t.Errorf("ID() = %q, want \"github\"", p.ID())
    }
}

func TestGitHubProvider_GenerateConfig(t *testing.T) {
    validPAT := "ghp_" + strings.Repeat("a", 36)

    tests := []struct {
        name    string
        creds   map[string]string
        wantErr bool
        check   func(*testing.T, provider.MCPConfig)
    }{
        {
            name:  "valid ghp_ PAT produces correct stdio config",
            creds: map[string]string{"GITHUB_PERSONAL_ACCESS_TOKEN": validPAT},
            check: func(t *testing.T, cfg provider.MCPConfig) {
                if cfg.Type != provider.TransportStdio {
                    t.Errorf("Type = %q, want stdio", cfg.Type)
                }
                if cfg.Command != "npx" {
                    t.Errorf("Command = %q, want npx", cfg.Command)
                }
                if len(cfg.Args) < 2 || cfg.Args[1] != "@modelcontextprotocol/server-github" {
                    t.Errorf("Args = %v, expected @modelcontextprotocol/server-github", cfg.Args)
                }
                if cfg.Env["GITHUB_PERSONAL_ACCESS_TOKEN"] != validPAT {
                    t.Errorf("Env does not contain the PAT")
                }
                if cfg.Runtime == nil || cfg.Runtime.Type != "npm" {
                    t.Errorf("Runtime should be npm")
                }
            },
        },
        {
            name:    "missing PAT returns error",
            creds:   map[string]string{},
            wantErr: true,
        },
        {
            name:    "empty PAT returns error",
            creds:   map[string]string{"GITHUB_PERSONAL_ACCESS_TOKEN": ""},
            wantErr: true,
        },
    }

    p := provider.NewGitHubProvider()
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            cfg, err := p.GenerateConfig(tt.creds)
            if (err != nil) != tt.wantErr {
                t.Fatalf("GenerateConfig() error = %v, wantErr %v", err, tt.wantErr)
            }
            if tt.check != nil {
                tt.check(t, cfg)
            }
        })
    }
}

func TestGitHubProvider_CredentialValidator(t *testing.T) {
    p := provider.NewGitHubProvider()
    specs := p.RequiredCredentials()
    if len(specs) != 1 {
        t.Fatalf("expected 1 credential spec, got %d", len(specs))
    }
    validate := specs[0].Validator

    valid := []string{
        "ghp_" + strings.Repeat("a", 36),
        "github_pat_" + strings.Repeat("b", 59),
        strings.Repeat("a", 40), // classic 40-char hex
    }
    for _, v := range valid {
        if err := validate(v); err != nil {
            t.Errorf("validator rejected valid PAT %q: %v", v, err)
        }
    }

    invalid := []string{
        "",
        "not-a-token",
        "ghp_tooshort",
        "ghp_" + strings.Repeat("!", 36), // wrong chars
    }
    for _, v := range invalid {
        if err := validate(v); err == nil {
            t.Errorf("validator accepted invalid PAT %q", v)
        }
    }
}
