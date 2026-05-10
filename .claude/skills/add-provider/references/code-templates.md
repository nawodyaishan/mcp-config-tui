# Code Templates

## Template 1: Key Validation (e.g., `pkg/{{ID}}/keys.go`)
```go
package {{ID}}

import (
    "fmt"
    "strings"
)

func ParseKey(key string) (string, error) {
    if len(key) == 0 {
        return "", fmt.Errorf("key cannot be empty")
    }
    return key, nil
}
```

## Template 2: Provider Implementation (e.g., `pkg/provider/{{ID}}.go`)
```go
package provider

type {{NAME}}Provider struct{}

func New{{NAME}}Provider() *{{NAME}}Provider { return &{{NAME}}Provider{} }

func (p *{{NAME}}Provider) ID() string          { return "{{ID}}" }
func (p *{{NAME}}Provider) Name() string        { return "{{NAME}}" }
func (p *{{NAME}}Provider) Description() string { return "Description here" }

func (p *{{NAME}}Provider) RequiredCredentials() []CredentialSpec {
    return []CredentialSpec{
        {
            Key:         "{{ENV_VAR}}",
            Label:       "{{NAME}} API Key",
            Description: "Get your key...",
            Secret:      true,
            MultiValue:  false,
            Validator: func(s string) error { return nil },
        },
    }
}

func (p *{{NAME}}Provider) GenerateConfig(credentials map[string]string) (MCPConfig, error) {
    return MCPConfig{
        Type:    TransportStdio,
        Command: "npx",
        Args:    []string{"-y", "mcp-server-{{ID}}"},
        Env:     map[string]string{"{{ENV_VAR}}": credentials["{{ENV_VAR}}"]},
    }, nil
}
```

## Template 3: QA Scenario (`pkg/app/qa_scenarios_test.go`)
```go
func TestQA{{NAME}}AllClients(t *testing.T) {
    // Scaffold temp directory and mock paths
    // Initialize provider and credentials
    // Assert plan string does not contain raw keys
    // Run apply
    // Verify file contents match expected shapes
}
```