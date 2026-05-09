package tui

import (
	"github.com/nawodyaishan/universal-mcp-sync/pkg/app"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/config"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/provider"
)

type wizardContext struct {
	manager     *app.Manager
	registry    provider.Registry
	providerID  string
	provider    provider.MCPProvider
	profiles    []provider.CredentialProfile
	isPreloaded bool
	selected    map[config.AppID]bool
	assignments map[config.AppID]int
	plan        app.ExecutionPlan
	result      app.ApplyResult
	err         error
}
