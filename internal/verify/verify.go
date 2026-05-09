package verify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/nawodyaishan/mcp-config-tui/internal/config"
	"github.com/nawodyaishan/mcp-config-tui/internal/exa"
	"github.com/nawodyaishan/mcp-config-tui/internal/provider"
)

type Status string

const (
	StatusOK      Status = "ok"
	StatusWarning Status = "warning"
	StatusSkipped Status = "skipped"
	StatusFailed  Status = "failed"
)

type Result struct {
	Target  string
	Status  Status
	Details []string
}

type Runner interface {
	LookPath(name string) (string, error)
	Run(name string, args ...string) (string, error)
}

func VerifyFile(path string, kind config.FileKind, expectedTools int) Result {
	switch kind {
	case config.FileKindMCPServers:
		return verifyMCPServersFile(path, expectedTools)
	case config.FileKindBareMCPServers:
		return verifyBareMCPServersFile(path, expectedTools)
	case config.FileKindNamedServer:
		return verifyNamedServerFile(path, expectedTools)
	case config.FileKindCodexTOML:
		return verifyCodexFile(path, expectedTools)
	default:
		return failure(path, "unsupported verification target")
	}
}

func VerifyProviderFile(path string, kind config.FileKind, providerID string, cfg provider.MCPConfig) Result {
	if providerID == "exa" {
		return VerifyFile(path, kind, len(exa.DefaultTools))
	}
	return failure(path, fmt.Sprintf("verification not implemented for provider %s", providerID))
}

func VerifyOptionalCLI(runner Runner, binary string, args ...string) Result {
	label := binary + " " + strings.Join(args, " ")
	if _, err := runner.LookPath(binary); err != nil {
		return Result{
			Target:  label,
			Status:  StatusSkipped,
			Details: []string{"CLI unavailable"},
		}
	}

	output, err := runner.Run(binary, args...)
	if err != nil {
		return Result{
			Target:  label,
			Status:  StatusWarning,
			Details: []string{exa.RedactText(err.Error())},
		}
	}

	return Result{
		Target:  label,
		Status:  StatusOK,
		Details: summarizeOutput(output),
	}
}

func verifyMCPServersFile(path string, expectedTools int) Result {
	data, err := os.ReadFile(path)
	if err != nil {
		return failure(path, err.Error())
	}

	root := make(map[string]any)
	if err := json.Unmarshal(data, &root); err != nil {
		return failure(path, fmt.Sprintf("parse JSON: %v", err))
	}

	servers, ok := root["mcpServers"].(map[string]any)
	if !ok {
		return failure(path, "missing mcpServers object")
	}

	exaValue, ok := servers["exa"].(map[string]any)
	if !ok {
		return failure(path, "missing mcpServers.exa entry")
	}

	urlValue := getURLField(exaValue)
	return inspectFileURL(path, urlValue, expectedTools)
}

func verifyBareMCPServersFile(path string, expectedTools int) Result {
	data, err := os.ReadFile(path)
	if err != nil {
		return failure(path, err.Error())
	}

	root := make(map[string]any)
	if err := json.Unmarshal(data, &root); err != nil {
		return failure(path, fmt.Sprintf("parse JSON: %v", err))
	}

	exaValue, ok := root["exa"].(map[string]any)
	if !ok {
		return failure(path, "missing exa entry")
	}

	urlValue := getURLField(exaValue)
	return inspectFileURL(path, urlValue, expectedTools)
}

func verifyNamedServerFile(path string, expectedTools int) Result {
	data, err := os.ReadFile(path)
	if err != nil {
		return failure(path, err.Error())
	}

	root := make(map[string]any)
	if err := json.Unmarshal(data, &root); err != nil {
		return failure(path, fmt.Sprintf("parse JSON: %v", err))
	}

	exaValue, ok := root["exa"].(map[string]any)
	if !ok {
		return failure(path, "missing exa entry")
	}

	urlValue := getURLField(exaValue)
	return inspectFileURL(path, urlValue, expectedTools)
}

func getURLField(obj map[string]any) string {
	fields := []string{"url", "httpUrl", "serverUrl"}
	for _, f := range fields {
		if val, ok := obj[f].(string); ok && val != "" {
			return val
		}
	}
	return ""
}

func verifyCodexFile(path string, expectedTools int) Result {
	data, err := os.ReadFile(path)
	if err != nil {
		return failure(path, err.Error())
	}

	text := string(data)
	section := "[mcp_servers.exa]"
	index := strings.Index(text, section)
	if index == -1 {
		return failure(path, "missing [mcp_servers.exa] block")
	}

	block := text[index:]
	lines := strings.Split(block, "\n")
	urlValue := ""
	for _, line := range lines[1:] {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			break
		}
		if strings.HasPrefix(trimmed, "url = ") {
			urlValue = strings.Trim(strings.TrimPrefix(trimmed, "url = "), `"`)
			break
		}
	}

	return inspectFileURL(path, urlValue, expectedTools)
}

func inspectFileURL(path, raw string, expectedTools int) Result {
	details, ok := inspectURL(raw, expectedTools)
	if !ok {
		return Result{Target: path, Status: StatusFailed, Details: details}
	}
	return Result{Target: path, Status: StatusOK, Details: details}
}

func inspectURL(raw string, expectedTools int) ([]string, bool) {
	if raw == "" {
		return []string{"missing Exa URL"}, false
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return []string{fmt.Sprintf("parse URL: %v", err)}, false
	}

	query := parsed.Query()
	key := query.Get("exaApiKey")
	tools := strings.Split(query.Get("tools"), ",")

	details := []string{
		fmt.Sprintf("host=%s", parsed.Host),
		fmt.Sprintf("api key present=%t", key != ""),
		fmt.Sprintf("tool count=%d", len(filterEmpty(tools))),
	}

	if parsed.Scheme != "https" || parsed.Host != "mcp.exa.ai" {
		return append(details, "unexpected Exa MCP endpoint"), false
	}
	if key == "" {
		return append(details, "missing exaApiKey"), false
	}
	if len(filterEmpty(tools)) != expectedTools {
		return append(details, "unexpected tool count"), false
	}
	return details, true
}

func summarizeOutput(output string) []string {
	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return []string{"command completed"}
	}

	lines := bytes.Split([]byte(trimmed), []byte("\n"))
	if len(lines) > 3 {
		lines = lines[:3]
	}

	summary := make([]string, 0, len(lines))
	for _, line := range lines {
		summary = append(summary, exa.RedactText(string(bytes.TrimSpace(line))))
	}
	return summary
}

func filterEmpty(values []string) []string {
	filtered := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		filtered = append(filtered, value)
	}
	return filtered
}

func failure(target, detail string) Result {
	return Result{Target: target, Status: StatusFailed, Details: []string{detail}}
}
