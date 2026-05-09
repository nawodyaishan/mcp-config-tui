package tui

import (
	"github.com/nawodyaishan/mcp-config-tui/pkg/app"
	"github.com/nawodyaishan/mcp-config-tui/pkg/config"
	"github.com/nawodyaishan/mcp-config-tui/pkg/provider"
)

type wizardContext struct {
	manager     *app.Manager
	registry    provider.Registry
	providerID  string
	provider    provider.MCPProvider
	profiles    []provider.CredentialProfile
	selected    map[config.AppID]bool
	assignments map[config.AppID]int
	plan        app.ExecutionPlan
	result      app.ApplyResult
	err         error
}
