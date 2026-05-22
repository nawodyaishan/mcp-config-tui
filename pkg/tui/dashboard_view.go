package tui

import (
	"fmt"
	"strings"

	"github.com/nawodyaishan/universal-mcp-sync/pkg/doctor"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/redact"
)

// View renders the current state of the dashboard.
func (m DashboardModel) View() string {
	if m.showHelp {
		return renderShell(renderHelpOverlay(), stageSetup, m.width) // using stageSetup as placeholder
	}

	var content string
	if m.scanning {
		content = "Scanning for AI clients and runtimes...\n"
	} else if m.err != nil {
		content = fmt.Sprintf("Error scanning clients: %v\n", m.err)
	} else {
		content = m.renderReport()
	}

	if m.placeholderMsg != "" {
		content += "\n[Status] " + m.placeholderMsg + "\n"
	}

	// Action bar
	content += "\n[r] rescan  [?] help  [w] wizard  [c] conflict  [x] clean  [m] migrate  [q] quit"

	return renderShell(content, stageSetup, m.width)
}

func (m DashboardModel) renderReport() string {
	var b strings.Builder

	b.WriteString("System Status\n\n")

	// Runtimes
	if len(m.report.Runtimes) > 0 {
		b.WriteString("Runtimes:\n")
		for _, rt := range m.report.Runtimes {
			status := "OK"
			if !rt.Available {
				status = "MISSING"
			} else if rt.Error != "" {
				status = "WARNING"
			}
			fmt.Fprintf(&b, "  - %s: %s\n", rt.Name, status)
			if rt.Error != "" {
				fmt.Fprintf(&b, "    %s\n", redact.Key(rt.Error))
			}
		}
		b.WriteString("\n")
	}

	// Warnings
	if len(m.report.Warnings) > 0 {
		b.WriteString("Global Warnings:\n")
		for _, w := range m.report.Warnings {
			fmt.Fprintf(&b, "  ! %s\n", redact.Key(w))
		}
		b.WriteString("\n")
	}

	// Clients
	clients := m.report.Clients
	if len(clients) == 0 {
		b.WriteString("No AI clients detected.\n")
	} else {
		fmt.Fprintf(&b, "AI Clients Detected (%d):\n", len(clients))
		
		// Render conflicts first
		for _, client := range clients {
			if client.Confidence == doctor.ConfidenceConflict {
				m.renderClient(&b, client)
			}
		}
		// Then the rest
		for _, client := range clients {
			if client.Confidence != doctor.ConfidenceConflict {
				m.renderClient(&b, client)
			}
		}
	}

	return b.String()
}

func (m DashboardModel) renderClient(b *strings.Builder, client doctor.ClientFinding) {
	status := string(client.Confidence)
	if !client.Installed {
		status = "not installed"
	}

	fmt.Fprintf(b, "  * %s [%s]\n", client.Name, status)
	if client.EffectivePath != "" {
		fmt.Fprintf(b, "    Config: %s\n", redact.Key(client.EffectivePath))
	}

	if len(client.ConfiguredProviders) > 0 {
		fmt.Fprintf(b, "    Providers: %s\n", strings.Join(client.ConfiguredProviders, ", "))
	}

	for _, issue := range client.Issues {
		fmt.Fprintf(b, "    ! Issue: %s\n", redact.Key(issue))
	}
	for _, warning := range client.Warnings {
		fmt.Fprintf(b, "    ! Warning: %s\n", redact.Key(warning))
	}
}
