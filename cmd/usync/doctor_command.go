package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/nawodyaishan/universal-mcp-sync/pkg/doctor"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/manifest"
)

func runDoctorCommand(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("usync doctor", flag.ContinueOnError)
	flags.SetOutput(stderr)

	var homeDir string
	var workspace string
	var jsonOutput bool
	var noRuntimes bool
	var clientsCSV string
	var verbose bool

	flags.StringVar(&homeDir, "home-dir", "", "override the target home directory for testing")
	flags.StringVar(&workspace, "workspace", "", "workspace directory to scan for project-local configs")
	flags.BoolVar(&jsonOutput, "json", false, "print machine-readable json output")
	flags.BoolVar(&noRuntimes, "no-runtimes", false, "skip runtime availability checks")
	flags.StringVar(&clientsCSV, "clients", "", "comma-separated client ids to include")
	flags.BoolVar(&verbose, "verbose", false, "show candidate, runtime, and warning detail")
	if err := flags.Parse(args); err != nil {
		return 2
	}

	scanner, err := doctor.New(doctor.Options{
		HomeDir:       homeDir,
		WorkspaceDir:  workspace,
		CheckRuntimes: !noRuntimes,
	})
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}

	report, err := scanner.Scan(context.Background())
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	report = filterDoctorReport(report, parseDoctorClientIDs(clientsCSV))

	if jsonOutput {
		data, err := doctor.MarshalReportJSON(report)
		if err != nil {
			_, _ = fmt.Fprintln(stderr, err)
			return 1
		}
		_, _ = stdout.Write(data)
	} else {
		if verbose {
			_, _ = fmt.Fprintln(stdout, doctor.FormatVerboseReport(report))
		} else {
			_, _ = fmt.Fprintln(stdout, doctor.FormatReport(report))
		}
	}

	if report.HasFindings() {
		return 2
	}
	return 0
}

func parseDoctorClientIDs(csv string) map[manifest.ClientID]bool {
	selected := make(map[manifest.ClientID]bool)
	for _, part := range strings.Split(csv, ",") {
		id := strings.TrimSpace(part)
		if id == "" {
			continue
		}
		selected[manifest.ClientID(id)] = true
	}
	return selected
}

func filterDoctorReport(report doctor.Report, selected map[manifest.ClientID]bool) doctor.Report {
	if len(selected) == 0 {
		return report
	}

	filtered := report
	filtered.Clients = make([]doctor.ClientFinding, 0, len(report.Clients))
	filtered.Warnings = filtered.Warnings[:0]
	requiredRuntimeIDs := make(map[string]bool)

	for _, client := range report.Clients {
		if !selected[client.ID] {
			continue
		}
		filtered.Clients = append(filtered.Clients, client)
		for _, warning := range client.Warnings {
			filtered.Warnings = append(filtered.Warnings, fmt.Sprintf("%s: %s", client.Name, warning))
		}
		for _, runtime := range manifest.AllRuntimeRequirements() {
			for _, requiredFor := range runtime.RequiredFor {
				if requiredFor == string(client.ID) {
					requiredRuntimeIDs[runtime.ID] = true
				}
			}
		}
	}

	filtered.Runtimes = make([]doctor.RuntimeFinding, 0, len(report.Runtimes))
	for _, runtime := range report.Runtimes {
		if requiredRuntimeIDs[runtime.ID] {
			filtered.Runtimes = append(filtered.Runtimes, runtime)
		}
	}
	return filtered
}
