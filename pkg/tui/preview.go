package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/app"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/provider"
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
	return renderSection(
		"Preview",
		renderPreviewPlan(m.ctx.plan, m.ctx.manager.HomeDir),
		renderKeyHelp("enter apply", "b back", "ctrl+c quit"),
	)
}

func signalNext() tea.Msg { return nextMsg{} }
func signalBack() tea.Msg { return backMsg{} }

type nextMsg struct{}
type backMsg struct{}

func renderPreviewPlan(plan app.ExecutionPlan, homeDir string) string {
	if len(plan.Operations) == 0 {
		return mutedStyle.Render("No targets selected.")
	}

	var builder strings.Builder
	targetLabel := "targets"
	if len(plan.Operations) == 1 {
		targetLabel = "target"
	}

	first := plan.Operations[0]
	builder.WriteString(accentStyle.Render("Ready to apply MCP configuration"))
	builder.WriteString("\n\n")
	fmt.Fprintf(&builder, "Targets   %d %s\n", len(plan.Operations), targetLabel)
	fmt.Fprintf(&builder, "Provider  %s\n", first.ProviderID)
	fmt.Fprintf(&builder, "Mode      %s transport\n", first.Config.Type)
	builder.WriteString("Safety    backups before file writes; credentials stay redacted\n")

	if len(plan.Warnings) > 0 {
		builder.WriteString("\n")
		builder.WriteString(sectionTitleStyle.Render("Warnings"))
		builder.WriteString("\n")
		for _, warning := range plan.Warnings {
			fmt.Fprintf(&builder, "- %s\n", warning)
		}
	}

	builder.WriteString("\n")
	builder.WriteString(sectionTitleStyle.Render("Targets"))
	builder.WriteString("\n")
	for index, op := range plan.Operations {
		fmt.Fprintf(&builder, "\n%s %s\n", accentStyle.Render(fmt.Sprintf("%d.", index+1)), op.AppName)
		fmt.Fprintf(&builder, "   Config   %s\n", op.FileLabel)

		if op.Config.Type == provider.TransportStdio {
			fmt.Fprintf(&builder, "   Transport stdio (%s %s)\n",
				op.Config.Command, strings.Join(op.Config.Args, " "))
		} else {
			fmt.Fprintf(&builder, "   Transport %s\n", op.Config.Type)
		}

		fmt.Fprintf(&builder, "   Key      %s\n", op.CredentialLabel)

		if op.SkipReason != "" {
			builder.WriteString("   Status   skipped\n")
			fmt.Fprintf(&builder, "   Reason   %s\n", op.SkipReason)
			continue
		}

		if op.Path == "" {
			fmt.Fprintf(&builder, "   Action   update through %s command\n", op.AppName)
			continue
		}

		action := "update existing file"
		if op.WillCreate {
			action = "create new file"
		}
		fmt.Fprintf(&builder, "   Action   %s\n", action)
		fmt.Fprintf(&builder, "   Path     %s\n", shortenHomePath(op.Path, homeDir))
		if op.WillCreate {
			builder.WriteString("   Backup   not needed for new file\n")
		} else if op.BackupPath != "" {
			fmt.Fprintf(&builder, "   Backup   %s\n", shortenHomePath(op.BackupPath, homeDir))
		}
	}

	return strings.TrimSpace(builder.String())
}

func shortenHomePath(path, homeDir string) string {
	if path == "" || homeDir == "" {
		return path
	}
	cleanPath := filepath.Clean(path)
	cleanHome := filepath.Clean(homeDir)
	if cleanPath == cleanHome {
		return "~"
	}
	prefix := cleanHome + string(filepath.Separator)
	if strings.HasPrefix(cleanPath, prefix) {
		return "~" + strings.TrimPrefix(cleanPath, cleanHome)
	}
	return path
}
