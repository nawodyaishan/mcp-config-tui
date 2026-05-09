package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"

	"github.com/nawodyaishan/universal-mcp-sync/pkg/app"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/config"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/exa"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/provider"
)

type stage int

const (
	stageSetup stage = iota
	stageAssignments
	stagePreview
	stageResults
)

type Model struct {
	ctx   *wizardContext
	stage stage
	width int

	// Sub-models
	setupForm   *setupForm
	assignments assignmentModel
	preview     previewModel
	results     resultsModel
}

func NewModel(manager *app.Manager, initialKeys []string, initialRaw string) Model {
	selected := make(map[config.AppID]bool, len(manager.Apps))
	for _, appConfig := range manager.Apps {
		if appConfig.ID == config.AppClaudeCode {
			selected[appConfig.ID] = true
		}
	}

	registry := provider.DefaultRegistry()

	ctx := &wizardContext{
		manager:     manager,
		registry:    registry,
		providerID:  "exa", // Default to Exa, will be updated by setup form
		isPreloaded: len(initialKeys) > 0,
		selected:    selected,
		assignments: app.DefaultAssignments(selected, len(initialKeys)),
	}

	// For backward compatibility, if keys were passed in, we seed them as profiles for Exa
	if ctx.isPreloaded {
		profiles := make([]provider.CredentialProfile, len(initialKeys))
		for i, key := range initialKeys {
			profiles[i] = provider.CredentialProfile{
				ProviderID: "exa",
				Values: map[string]string{
					"EXA_API_KEY": key,
				},
				Label: exa.RedactKey(key),
			}
		}
		ctx.profiles = profiles
	}

	model := Model{
		ctx:         ctx,
		stage:       stageSetup,
		setupForm:   newSetupForm(ctx, initialRaw),
		assignments: assignmentModel{ctx: ctx},
		preview:     previewModel{ctx: ctx},
		results:     resultsModel{ctx: ctx},
	}
	return model
}

func (m Model) Init() tea.Cmd {
	return m.setupForm.form.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
	case nextMsg:
		m.stage++
		return m, nil
	case backMsg:
		if m.stage > 0 {
			m.stage--
			if m.stage == stageSetup {
				if !m.ctx.isPreloaded {
					m.ctx.profiles = nil
				}
				m.setupForm.form.State = huh.StateNormal
				m.setupForm.rebuildForm() // Ensure fields are fresh
				return m, m.setupForm.form.Init()
			}
		}
		return m, nil
	}

	var cmd tea.Cmd
	switch m.stage {
	case stageSetup:
		_, cmd = m.setupForm.update(msg)
		if m.setupForm.form.State == huh.StateCompleted {
			m.setupForm.syncToContext()
			m.ctx.assignments = app.DefaultAssignments(m.ctx.selected, len(m.ctx.profiles))
			m.stage = stageAssignments
		}
	case stageAssignments:
		_, cmd = m.assignments.Update(msg)
	case stagePreview:
		_, cmd = m.preview.Update(msg)
	case stageResults:
		_, cmd = m.results.Update(msg)
	}

	return m, cmd
}

func (m Model) View() string {
	view := ""
	switch m.stage {
	case stageSetup:
		view = m.setupForm.form.View()
	case stageAssignments:
		view = m.assignments.View()
	case stagePreview:
		view = m.preview.View()
	case stageResults:
		view = m.results.View()
	}

	if m.ctx.err != nil {
		view += renderError(m.ctx.err)
	}
	return renderShell(view, m.stage, m.width)
}

func (m Model) Err() error {
	return m.ctx.err
}
