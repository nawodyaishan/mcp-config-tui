package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/nawodyaishan/mcp-config-tui/pkg/app"
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
		trimPreview(app.FormatApplyResult(m.ctx.result), 60),
		renderKeyHelp("enter quit", "q quit"),
	)
}
