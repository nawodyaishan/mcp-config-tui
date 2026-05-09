package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/config"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/provider"
)

const defaultViewWidth = 88

var (
	shellStyle = lipgloss.NewStyle().
			Padding(1, 2).
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(lipgloss.Color("99"))
	panelStyle = lipgloss.NewStyle().
			PaddingLeft(1).
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(lipgloss.Color("62"))
	markStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("114")).
			Bold(true)
	brandStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("229")).
			Bold(true)
	accentStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("114")).
			Bold(true)
	mutedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244"))
	stepStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Padding(0, 1)
	activeStepStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("16")).
			Background(lipgloss.Color("114")).
			Bold(true).
			Padding(0, 1)
	sectionTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("183")).
				Bold(true)
	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("203")).
			Bold(true)
	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("114")).
			Bold(true)
	warningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
			Bold(true)
	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))
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

func assignmentLabel(profiles []provider.CredentialProfile, index int) string {
	if index < 0 || index >= len(profiles) {
		return "unassigned"
	}
	return profiles[index].Label
}

func renderError(err error) string {
	if err == nil {
		return ""
	}
	return "\n" + errorStyle.Render("Error: "+err.Error()) + "\n"
}

func renderShell(body string, current stage, width int) string {
	if width <= 0 {
		width = defaultViewWidth
	}
	contentWidth := width - 6
	if contentWidth < 64 {
		contentWidth = 64
	}
	if contentWidth > 104 {
		contentWidth = 104
	}

	header := lipgloss.JoinHorizontal(
		lipgloss.Top,
		markStyle.Render("[<>]"),
		mutedStyle.Render(" "),
		brandStyle.Render("MCP Config"),
		mutedStyle.Render("  @"),
		accentStyle.Render("nawodyaishan"),
		mutedStyle.Render("  /  local config sync"),
	)

	return shellStyle.Width(contentWidth).Render(
		header + "\n" +
			renderStageBar(current) + "\n\n" +
			body,
	)
}

func renderStageBar(current stage) string {
	labels := []string{"1 Setup", "2 Assign", "3 Preview", "4 Results"}
	parts := make([]string, len(labels))
	for i, label := range labels {
		if stage(i) == current {
			parts[i] = activeStepStyle.Render(label)
			continue
		}
		parts[i] = stepStyle.Render(label)
	}
	return strings.Join(parts, mutedStyle.Render(" -> "))
}

func renderSection(title, body, help string) string {
	parts := []string{sectionTitleStyle.Render(title), panelStyle.Render(strings.TrimRight(body, "\n"))}
	if help != "" {
		parts = append(parts, mutedStyle.Render(help))
	}
	return strings.Join(parts, "\n\n")
}

func renderKeyHelp(items ...string) string {
	return fmt.Sprintf("[%s]", strings.Join(items, "] ["))
}
