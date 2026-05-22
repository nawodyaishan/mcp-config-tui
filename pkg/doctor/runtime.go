package doctor

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/nawodyaishan/universal-mcp-sync/pkg/manifest"
)

func defaultLookPath(name string) (string, error) {
	return exec.LookPath(name)
}

func defaultRunCmd(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		return "", fmt.Errorf("%s", message)
	}
	return strings.TrimSpace(stdout.String()), nil
}

func (d *Doctor) scanRuntimes(ctx context.Context) []RuntimeFinding {
	runtimes := manifest.AllRuntimeRequirements()
	results := make([]RuntimeFinding, 0, len(runtimes))

	for _, runtimeReq := range runtimes {
		finding := RuntimeFinding{
			ID:          runtimeReq.ID,
			Name:        runtimeReq.Name,
			RequiredFor: append([]string(nil), runtimeReq.RequiredFor...),
		}
		path, err := d.lookPath(runtimeReq.Command)
		if err != nil {
			finding.Error = "not found"
			results = append(results, finding)
			continue
		}

		finding.Available = true
		finding.Path = path
		cmdCtx, cancel := context.WithTimeout(ctx, d.options.CommandTimeout)
		version, err := d.runCmd(cmdCtx, runtimeReq.Command, runtimeReq.Args...)
		cancel()
		if err != nil {
			finding.Error = err.Error()
			results = append(results, finding)
			continue
		}
		finding.Version = version
		results = append(results, finding)
	}

	return results
}
