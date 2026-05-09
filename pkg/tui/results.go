package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/app"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/verify"
)

type resultsModel struct {
	ctx *wizardContext
}

func (m resultsModel) Init() tea.Cmd {
	return nil
}

func (m resultsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "enter":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m resultsModel) View() string {
	return renderSection(
		"Results",
		renderApplyResults(m.ctx.result),
		renderKeyHelp("enter finish", "q quit", "ctrl+c quit"),
	)
}

func renderApplyResults(result app.ApplyResult) string {
	var builder strings.Builder

	// Summary header
	if len(result.UpdatedTargets) > 0 {
		builder.WriteString(successStyle.Render(fmt.Sprintf("✓ Successfully updated %d targets", len(result.UpdatedTargets))))
	} else {
		builder.WriteString(warningStyle.Render("! No changes were applied"))
	}
	builder.WriteString("\n\n")

	// Backups section
	if len(result.BackupPaths) > 0 {
		builder.WriteString(sectionTitleStyle.Render("Backups Created"))
		builder.WriteString("\n")
		for _, path := range result.BackupPaths {
			fmt.Fprintf(&builder, "  %s\n", dimStyle.Render(path))
		}
		builder.WriteString("\n")
	}

	// Verification section
	if len(result.Verification) > 0 {
		builder.WriteString(sectionTitleStyle.Render("Verification"))
		builder.WriteString("\n")
		for _, item := range result.Verification {
			status := "[?]"
			switch item.Status {
			case verify.StatusOK:
				status = successStyle.Render("[OK]")
			case verify.StatusWarning:
				status = warningStyle.Render("[WARN]")
			case verify.StatusFailed:
				status = errorStyle.Render("[FAIL]")
			case verify.StatusSkipped:
				status = dimStyle.Render("[SKIP]")
			}

			fmt.Fprintf(&builder, "- %s %s\n", status, item.Target)
			for _, detail := range item.Details {
				fmt.Fprintf(&builder, "     %s\n", dimStyle.Render(detail))
			}
		}
		builder.WriteString("\n")
	}

	// Final call to action
	builder.WriteString(accentStyle.Render("Next Steps:"))
	builder.WriteString("\nRestart the affected AI agents to reload the new configuration.\n")

	return builder.String()
}
