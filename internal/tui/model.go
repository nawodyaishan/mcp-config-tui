package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/nawodyaishan/mcp-config-tui/internal/app"
	"github.com/nawodyaishan/mcp-config-tui/internal/config"
	"github.com/nawodyaishan/mcp-config-tui/internal/exa"
)

type stage int

const (
	stageWelcome stage = iota
	stageKeys
	stageApps
	stageAssignments
	stagePreview
	stageResults
)

type Model struct {
	manager     *app.Manager
	stage       stage
	cursor      int
	keyInput    textarea.Model
	keys        []string
	selected    map[config.AppID]bool
	assignments map[config.AppID]int
	plan        app.ExecutionPlan
	result      app.ApplyResult
	err         error
	loadedKeys  bool
}

func NewModel(manager *app.Manager, initialKeys []string, initialRaw string) Model {
	input := textarea.New()
	input.Placeholder = "Paste Exa keys here, one per line or as key1 = \"...\" entries"
	input.SetWidth(80)
	input.SetHeight(8)
	input.Focus()
	input.ShowLineNumbers = false
	if len(initialKeys) == 0 {
		input.SetValue(initialRaw)
	}

	selected := make(map[config.AppID]bool, len(manager.Apps))
	for _, appConfig := range manager.Apps {
		selected[appConfig.ID] = true
	}

	model := Model{
		manager:     manager,
		stage:       stageWelcome,
		keyInput:    input,
		keys:        initialKeys,
		selected:    selected,
		assignments: app.DefaultAssignments(selected, len(initialKeys)),
		loadedKeys:  len(initialKeys) > 0,
	}
	return model
}

func (m Model) Init() tea.Cmd {
	return textarea.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.stage {
		case stageWelcome:
			return m.updateWelcome(msg)
		case stageKeys:
			return m.updateKeys(msg)
		case stageApps:
			return m.updateApps(msg)
		case stageAssignments:
			return m.updateAssignments(msg)
		case stagePreview:
			return m.updatePreview(msg)
		case stageResults:
			return m.updateResults(msg)
		}
	}

	if m.stage == stageKeys {
		var cmd tea.Cmd
		m.keyInput, cmd = m.keyInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) View() string {
	switch m.stage {
	case stageWelcome:
		return renderWelcome(m)
	case stageKeys:
		return renderKeys(m)
	case stageApps:
		return renderApps(m)
	case stageAssignments:
		return renderAssignments(m)
	case stagePreview:
		return renderPreview(m)
	case stageResults:
		return renderResults(m)
	default:
		return ""
	}
}

func (m Model) Err() error {
	return m.err
}

func (m Model) updateWelcome(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "enter":
		m.stage = stageKeys
		return m, nil
	}
	return m, nil
}

func (m Model) updateKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "tab":
		keys, err := exa.ParseKeys(m.keyInput.Value())
		if err != nil {
			m.err = err
			return m, nil
		}
		m.err = nil
		m.keys = keys
		m.loadedKeys = len(keys) > 0
		m.keyInput.SetValue("")
		m.assignments = app.DefaultAssignments(m.selected, len(m.keys))
		m.stage = stageApps
		return m, nil
	}

	var cmd tea.Cmd
	m.keyInput, cmd = m.keyInput.Update(msg)
	return m, cmd
}

func (m Model) updateApps(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.manager.Apps)-1 {
			m.cursor++
		}
	case " ":
		appConfig := m.manager.Apps[m.cursor]
		m.selected[appConfig.ID] = !m.selected[appConfig.ID]
		m.assignments = app.DefaultAssignments(m.selected, len(m.keys))
	case "enter":
		if selectedCount(m.selected) == 0 {
			m.err = fmt.Errorf("select at least one target app")
			return m, nil
		}
		m.err = nil
		m.cursor = 0
		m.assignments = app.DefaultAssignments(m.selected, len(m.keys))
		m.stage = stageAssignments
	}
	return m, nil
}

func (m Model) updateAssignments(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	selectedApps := selectedAppIDs(m.manager.Apps, m.selected)
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
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
		plan, err := m.manager.Prepare(m.keys, m.selected, m.assignments)
		if err != nil {
			m.err = err
			return m, nil
		}
		m.err = nil
		m.plan = plan
		m.stage = stagePreview
	}
	return m, nil
}

func (m *Model) rotateAssignment(selectedApps []config.AppID, delta int) {
	if len(m.keys) <= 1 || len(selectedApps) == 0 {
		return
	}
	appID := selectedApps[m.cursor]
	current := m.assignments[appID]
	next := current + delta
	if next < 0 {
		next = len(m.keys) - 1
	}
	if next >= len(m.keys) {
		next = 0
	}
	m.assignments[appID] = next
}

func (m Model) updatePreview(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "b":
		m.stage = stageAssignments
	case "enter":
		result, err := m.manager.Apply(m.plan)
		if err != nil {
			m.err = err
			return m, nil
		}
		m.err = nil
		m.result = result
		m.stage = stageResults
	}
	return m, nil
}

func (m Model) updateResults(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c", "enter":
		return m, tea.Quit
	}
	return m, nil
}

func selectedCount(selected map[config.AppID]bool) int {
	count := 0
	for _, isSelected := range selected {
		if isSelected {
			count++
		}
	}
	return count
}

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

func fileStatus(file config.TargetFile) string {
	if file.Exists {
		return "exists"
	}
	if file.Creatable {
		return "missing, creatable"
	}
	return "missing"
}

func trimPreview(text string, lines int) string {
	split := strings.Split(text, "\n")
	if len(split) <= lines {
		return text
	}
	return strings.Join(split[:lines], "\n")
}
