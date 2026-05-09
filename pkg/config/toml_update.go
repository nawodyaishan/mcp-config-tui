package config

import (
	"fmt"
	"strings"

	"github.com/nawodyaishan/mcp-config-tui/pkg/provider"
)

func UpdateCodexTOML(data []byte, providerID string, cfg provider.MCPConfig) ([]byte, error) {
	if cfg.Type == provider.TransportStdio {
		return nil, fmt.Errorf("stdio transport is not supported in Codex TOML")
	}

	block := []string{
		fmt.Sprintf("[mcp_servers.%s]", providerID),
		fmt.Sprintf("url = %q", cfg.URL),
	}

	text := string(data)
	lines := strings.Split(text, "\n")
	output := make([]string, 0, len(lines)+3)
	inProvider := false
	inserted := false

	targetHeader := fmt.Sprintf("[mcp_servers.%s]", providerID)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if isSectionHeader(trimmed) {
			if trimmed == targetHeader {
				if !inserted {
					output = append(output, block...)
					inserted = true
				}
				inProvider = true
				continue
			}
			inProvider = false
		}

		if inProvider {
			continue
		}

		output = append(output, line)
	}

	if !inserted {
		for len(output) > 0 && output[len(output)-1] == "" {
			output = output[:len(output)-1]
		}
		if len(output) > 0 {
			output = append(output, "")
		}
		output = append(output, block...)
	}

	result := strings.Join(output, "\n")
	result = strings.TrimRight(result, "\n") + "\n"
	return []byte(result), nil
}

func isSectionHeader(line string) bool {
	return strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]")
}
