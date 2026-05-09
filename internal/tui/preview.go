package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/nawodyaishan/mcp-config-tui/internal/app"
)

type previewModel struct {
	ctx *wizardContext
}

func (m previewModel) Init() tea.Cmd {
	return nil
}

func (m previewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "b":
			return m, signalBack
		case "enter":
			result, err := m.ctx.manager.Apply(m.ctx.plan)
			if err != nil {
				m.ctx.err = err
				return m, nil
			}
			m.ctx.result = result
			return m, signalNext
		}
	}
	return m, nil
}

func (m previewModel) View() string {
	return "Preview\n=======\n\n" +
		trimPreview(app.FormatPlan(m.ctx.plan), 40) +
		"\n\nEnter: apply  b: back  q: quit\n"
}

func signalNext() tea.Msg { return nextMsg{} }
func signalBack() tea.Msg { return backMsg{} }

type nextMsg struct{}
type backMsg struct{}
