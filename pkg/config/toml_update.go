package config

import (
	"fmt"
	"sort"
	"strings"

	"github.com/nawodyaishan/universal-mcp-sync/pkg/provider"
)

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func UpdateCodexTOML(data []byte, providerID string, cfg provider.MCPConfig) ([]byte, error) {
	block := buildCodexBlock(providerID, cfg)

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

func buildCodexBlock(providerID string, cfg provider.MCPConfig) []string {
	block := []string{
		"# managed-by=usync",
		fmt.Sprintf("[mcp_servers.%s]", providerID),
	}
	if cfg.Type == provider.TransportStdio {
		block = append(block, fmt.Sprintf("command = %q", cfg.Command))
		if len(cfg.Args) > 0 {
			block = append(block, fmt.Sprintf("args = [%s]", quoteStringList(cfg.Args)))
		}
		if len(cfg.Env) > 0 {
			block = append(block, fmt.Sprintf("env = { %s }", inlineStringMap(cfg.Env)))
		}
		return block
	}

	block = append(block, fmt.Sprintf("url = %q", cfg.URL))
	if len(cfg.Headers) > 0 {
		block = append(block, fmt.Sprintf("http_headers = { %s }", inlineStringMap(cfg.Headers)))
	}
	return block
}

func quoteStringList(values []string) string {
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, fmt.Sprintf("%q", value))
	}
	return strings.Join(quoted, ", ")
}

func inlineStringMap(values map[string]string) string {
	pairs := make([]string, 0, len(values))
	for _, k := range sortedKeys(values) {
		pairs = append(pairs, fmt.Sprintf("%q = %q", k, values[k]))
	}
	return strings.Join(pairs, ", ")
}

func isSectionHeader(line string) bool {
	return strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]")
}
