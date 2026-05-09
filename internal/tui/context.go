package tui

import (
	"github.com/nawodyaishan/mcp-config-tui/internal/app"
	"github.com/nawodyaishan/mcp-config-tui/internal/config"
	"github.com/nawodyaishan/mcp-config-tui/internal/provider"
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
