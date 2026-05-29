package client_test

import (
	"testing"

	"github.com/nawodyaishan/universal-mcp-sync/pkg/client"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/config"
)

func TestMatrixCoversAllAppIDs(t *testing.T) {
	for _, id := range config.AppOrder {
		if _, ok := client.Matrix[id]; !ok {
			t.Errorf("Matrix missing entry for AppID %q — add it to pkg/client/capabilities.go", id)
		}
	}
}

func TestClaudeDesktopCapabilities(t *testing.T) {
	cap := client.Matrix[config.AppClaudeDesktop]
	if !cap.Supports.Stdio {
		t.Error("ClaudeDesktop must support Stdio natively")
	}
	if cap.Supports.StreamableHTTP {
		t.Error("ClaudeDesktop does not natively support StreamableHTTP; it uses a bridge")
	}
	if len(cap.Bridge) == 0 {
		t.Error("ClaudeDesktop must declare bridges for HTTP transports")
	}
}

func TestAntigravityCLICapabilities(t *testing.T) {
	cap := client.Matrix[config.AppAntigravityCLI]
	if cap.Supports.Stdio {
		t.Error("AntigravityCLI does not support local stdio subprocess servers")
	}
	if !cap.Supports.StreamableHTTP {
		t.Error("AntigravityCLI must support StreamableHTTP")
	}
}
