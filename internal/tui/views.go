package tui

import (
	"fmt"
	"strings"

	"github.com/nawodyaishan/mcp-config-tui/internal/app"
	"github.com/nawodyaishan/mcp-config-tui/internal/config"
	"github.com/nawodyaishan/mcp-config-tui/internal/exa"
)

func renderWelcome(m Model) string {
	var builder strings.Builder
	builder.WriteString("Exa MCP Config Manager\n")
	builder.WriteString("======================\n\n")
	builder.WriteString("Detected targets\n")
	for _, appConfig := range m.manager.Apps {
		builder.WriteString(fmt.Sprintf("- %s\n", appConfig.Name))
		for _, file := range appConfig.Files {
			builder.WriteString(fmt.Sprintf("  %s [%s]\n  %s\n", file.Label, fileStatus(file), file.Path))
		}
	}
	builder.WriteString("\nEnter: continue  q: quit\n")
	return builder.String()
}

func renderKeys(m Model) string {
	var builder strings.Builder
	builder.WriteString("Load Exa Keys\n")
	builder.WriteString("=============\n\n")
	builder.WriteString("Paste one or more UUID-style keys. Labelled lines like key1 = \"...\" also work.\n")
	builder.WriteString("Tab: continue  Ctrl+C: quit\n\n")
	if !m.loadedKeys {
		builder.WriteString(m.keyInput.View())
	}
	if len(m.keys) > 0 {
		builder.WriteString("\nParsed keys: ")
		labels := make([]string, 0, len(m.keys))
		for _, key := range m.keys {
			labels = append(labels, exa.RedactKey(key))
		}
		builder.WriteString(strings.Join(labels, ", "))
	}
	builder.WriteString(renderError(m.err))
	return builder.String()
}

func renderApps(m Model) string {
	var builder strings.Builder
	builder.WriteString("Select Target Apps\n")
	builder.WriteString("==================\n\n")
	for i, appConfig := range m.manager.Apps {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}
		checked := " "
		if m.selected[appConfig.ID] {
			checked = "x"
		}
		builder.WriteString(fmt.Sprintf("%s [%s] %s\n", cursor, checked, appConfig.Name))
	}
	builder.WriteString("\nUp/Down: move  Space: toggle  Enter: continue\n")
	builder.WriteString(renderError(m.err))
	return builder.String()
}

func renderAssignments(m Model) string {
	var builder strings.Builder
	builder.WriteString("Distribute Keys\n")
	builder.WriteString("===============\n\n")
	selectedApps := selectedAppIDs(m.manager.Apps, m.selected)
	for i, appID := range selectedApps {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}
		builder.WriteString(fmt.Sprintf("%s %s -> %s\n", cursor, configName(appID), assignmentLabel(m.keys, m.assignments[appID])))
	}
	builder.WriteString("\nUp/Down: move  Left/Right: change key  Enter: preview\n")
	builder.WriteString(renderError(m.err))
	return builder.String()
}

func renderPreview(m Model) string {
	var builder strings.Builder
	builder.WriteString("Preview\n")
	builder.WriteString("=======\n\n")
	builder.WriteString(trimPreview(app.FormatPlan(m.plan), 40))
	builder.WriteString("\n\nEnter: apply  b: back  q: quit\n")
	builder.WriteString(renderError(m.err))
	return builder.String()
}

func renderResults(m Model) string {
	var builder strings.Builder
	builder.WriteString("Results\n")
	builder.WriteString("=======\n\n")
	builder.WriteString(trimPreview(app.FormatApplyResult(m.result), 60))
	builder.WriteString("\n\nEnter: quit\n")
	builder.WriteString(renderError(m.err))
	return builder.String()
}

func configName(id config.AppID) string {
	return config.AppName(id)
}
