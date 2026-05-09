package tui

import (
	"github.com/nawodyaishan/mcp-config-tui/internal/app"
	"github.com/nawodyaishan/mcp-config-tui/internal/config"
)

type wizardContext struct {
	manager     *app.Manager
	keys        []string
	selected    map[config.AppID]bool
	assignments map[config.AppID]int
	plan        app.ExecutionPlan
	result      app.ApplyResult
	err         error
}
