package uxexplore

import (
	"regexp"
	"strings"
)

var actionBarRE = regexp.MustCompile(`\[([^\]]+)\]\s+(\S+)`)

func ParseActionBar(view string) []string {
	line := lastActionLine(view)
	matches := actionBarRE.FindAllStringSubmatch(line, -1)
	keys := make([]string, 0, len(matches))
	for _, match := range matches {
		keys = append(keys, normalizeKey(match[1])...)
	}
	return uniqueStrings(keys)
}

func lastActionLine(view string) string {
	lines := strings.Split(view, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(stripBoxPrefix(lines[i]))
		if len(actionBarRE.FindAllStringSubmatch(line, -1)) >= 2 {
			return line
		}
	}
	return ""
}

func stripBoxPrefix(line string) string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "│")
	return strings.TrimSpace(line)
}

func normalizeKey(token string) []string {
	token = strings.TrimSpace(strings.ToLower(token))
	// A `/`-separated label like `p/enter` advertises that EITHER key is a
	// valid binding. Split and normalize each side independently so the
	// probe can exercise both.
	if strings.Contains(token, "/") {
		var out []string
		for _, part := range strings.Split(token, "/") {
			out = append(out, normalizeKey(part)...)
		}
		return out
	}
	switch token {
	case "↑↓":
		return []string{"up", "down"}
	case "esc":
		return []string{"esc"}
	case "enter":
		return []string{"enter"}
	case "space":
		return []string{" "}
	case "ctrl+c":
		return []string{"ctrl+c"}
	default:
		return []string{token}
	}
}
