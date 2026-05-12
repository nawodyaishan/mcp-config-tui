package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/config"
)

type assignmentModel struct {
	ctx    *wizardContext
	cursor int
}

func (m assignmentModel) Init() tea.Cmd {
	return nil
}

func (m assignmentModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	selectedApps := selectedAppIDs(m.ctx.manager.Apps, m.ctx.selected)
	switch km.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(selectedApps)-1 {
			m.cursor++
		}
	case "left", "h":
		m.rotateAssignment(selectedApps, -1)
	case "right", "l":
		m.rotateAssignment(selectedApps, 1)
	case "enter":
		if len(selectedApps) == 0 {
			m.ctx.err = fmt.Errorf("select at least one target app")
			return m, signalBack
		}
		plan, err := m.ctx.manager.PrepareProvider(m.ctx.provider, m.ctx.profiles, m.ctx.selected, m.ctx.assignments)
		if err != nil {
			m.ctx.err = err
			return m, nil
		}
		m.ctx.err = nil
		m.ctx.plan = plan
		return m, signalNext
	case "b", "esc":
		return m, signalBack
	}
	return m, nil
}

func (m assignmentModel) View() string {
	var builder strings.Builder
	selectedApps := selectedAppIDs(m.ctx.manager.Apps, m.ctx.selected)
	for i, appID := range selectedApps {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}
		fmt.Fprintf(&builder, "%s %s -> %s\n", cursor, config.AppName(appID), assignmentLabel(m.ctx.profiles, m.ctx.assignments[appID]))
	}
	hints := []string{"up/down move"}
	if len(m.ctx.profiles) > 1 {
		hints = append(hints, "left/right change")
	}
	hints = append(hints, "enter preview", "esc back")
	return renderSection(
		"Distribute Credentials",
		builder.String(),
		renderKeyHelp(hints...),
	)
}

func (m *assignmentModel) rotateAssignment(selectedApps []config.AppID, delta int) {
	if len(m.ctx.profiles) <= 1 || len(selectedApps) == 0 {
		return
	}
	appID := selectedApps[m.cursor]
	current := m.ctx.assignments[appID]
	next := current + delta
	if next < 0 {
		next = len(m.ctx.profiles) - 1
	}
	if next >= len(m.ctx.profiles) {
		next = 0
	}
	m.ctx.assignments[appID] = next
}
