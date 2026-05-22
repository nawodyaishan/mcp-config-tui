package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/doctor"
)

// DashboardScanner abstracts the doctor scan so the TUI can be tested without filesystem access.
type DashboardScanner interface {
	Scan(ctx context.Context) (doctor.Report, error)
}

// ProductionScanner wraps the real pkg/doctor.
type ProductionScanner struct {
	Options doctor.Options
}

func NewProductionScanner(homeDir, workspaceDir string) *ProductionScanner {
	return &ProductionScanner{
		Options: doctor.Options{
			HomeDir:       homeDir,
			WorkspaceDir:  workspaceDir,
			CheckRuntimes: true, // "Production scanner enables runtime checks"
		},
	}
}

func (s *ProductionScanner) Scan(ctx context.Context) (doctor.Report, error) {
	doc, err := doctor.New(s.Options)
	if err != nil {
		return doctor.Report{}, err
	}
	return doc.Scan(ctx)
}

// DashboardModel is the read-only TUI state for the doctor dashboard.
type DashboardModel struct {
	scanner        DashboardScanner
	report         doctor.Report
	err            error
	scanning       bool
	RouteToWizard  bool
	placeholderMsg string
	width          int
	showHelp       bool
}

// NewDashboardModel creates a new dashboard.
func NewDashboardModel(scanner DashboardScanner) DashboardModel {
	return DashboardModel{
		scanner:  scanner,
		scanning: true, // starts in scanning state
	}
}

// scanResultMsg is returned when the async scan finishes.
type scanResultMsg struct {
	report doctor.Report
	err    error
}

// Init starts the async scan.
func (m DashboardModel) Init() tea.Cmd {
	return m.scanCmd()
}

func (m DashboardModel) scanCmd() tea.Cmd {
	return func() tea.Msg {
		report, err := m.scanner.Scan(context.Background())
		return scanResultMsg{report: report, err: err}
	}
}

// Update handles messages.
func (m DashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "r":
			if !m.scanning {
				m.scanning = true
				m.err = nil
				m.placeholderMsg = ""
				return m, m.scanCmd()
			}
		case "?":
			m.showHelp = !m.showHelp
		case "w":
			m.RouteToWizard = true
			return m, tea.Quit
		case "c":
			m.placeholderMsg = "Resolving conflicts is planned for Phase 8."
		case "x":
			m.placeholderMsg = "Cleanup of deprecated configs is planned for Phase 10."
		case "m":
			m.placeholderMsg = "Migration features are planned for Phase 10."
		}
	case scanResultMsg:
		m.scanning = false
		m.report = msg.report
		m.err = msg.err
	case tea.WindowSizeMsg:
		m.width = msg.Width
	}
	return m, nil
}


