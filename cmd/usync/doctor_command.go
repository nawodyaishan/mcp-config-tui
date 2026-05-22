package main

import (
	"context"
	"flag"
	"fmt"
	"io"

	"github.com/nawodyaishan/universal-mcp-sync/pkg/doctor"
)

func runDoctorCommand(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("usync doctor", flag.ContinueOnError)
	flags.SetOutput(stderr)

	var homeDir string
	var workspace string
	var jsonOutput bool
	var noRuntimes bool

	flags.StringVar(&homeDir, "home-dir", "", "override the target home directory for testing")
	flags.StringVar(&workspace, "workspace", "", "workspace directory to scan for project-local configs")
	flags.BoolVar(&jsonOutput, "json", false, "print machine-readable json output")
	flags.BoolVar(&noRuntimes, "no-runtimes", false, "skip runtime availability checks")
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

	if jsonOutput {
		data, err := doctor.MarshalReportJSON(report)
		if err != nil {
			_, _ = fmt.Fprintln(stderr, err)
			return 1
		}
		_, _ = stdout.Write(data)
	} else {
		_, _ = fmt.Fprintln(stdout, doctor.FormatReport(report))
	}

	if report.HasFindings() {
		return 2
	}
	return 0
}
