package tui

import (
	"strings"

	"github.com/nawodyaishan/mcp-config-tui/internal/config"
	"github.com/nawodyaishan/mcp-config-tui/internal/exa"
)

func selectedAppIDs(apps []config.AppConfig, selected map[config.AppID]bool) []config.AppID {
	ids := make([]config.AppID, 0, len(apps))
	for _, appConfig := range apps {
		if selected[appConfig.ID] {
			ids = append(ids, appConfig.ID)
		}
	}
	return ids
}

func assignmentLabel(keys []string, index int) string {
	if index < 0 || index >= len(keys) {
		return "unassigned"
	}
	return exa.RedactKey(keys[index])
}

func renderError(err error) string {
	if err == nil {
		return ""
	}
	return "\nError: " + err.Error() + "\n"
}

func trimPreview(text string, lines int) string {
	split := strings.Split(text, "\n")
	if len(split) <= lines {
		return text
	}
	return strings.Join(split[:lines], "\n")
}
