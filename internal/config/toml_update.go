package config

import (
	"fmt"
	"strings"
)

func UpdateCodexTOML(data []byte, exaURL string) ([]byte, error) {
	block := []string{
		"[mcp_servers.exa]",
		fmt.Sprintf("url = %q", exaURL),
	}

	text := string(data)
	lines := strings.Split(text, "\n")
	output := make([]string, 0, len(lines)+3)
	inExa := false
	inserted := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if isSectionHeader(trimmed) {
			if trimmed == "[mcp_servers.exa]" {
				if !inserted {
					output = append(output, block...)
					inserted = true
				}
				inExa = true
				continue
			}
			inExa = false
		}

		if inExa {
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
