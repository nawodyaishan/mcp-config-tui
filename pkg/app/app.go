package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/nawodyaishan/universal-mcp-sync/pkg/config"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/exa"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/provider"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/verify"
)

type CommandRunner interface {
	LookPath(name string) (string, error)
	Run(name string, args ...string) (string, error)
}

type WriteConfigFunc func(path string, data []byte, now time.Time) (config.WriteOutcome, error)

type ExecutionPlan struct {
	Operations []Operation
	Warnings   []string
}

type Operation struct {
	AppID           config.AppID
	AppName         string
	FileLabel       string
	Path            string
	Kind            config.FileKind
	CredentialLabel string
	ProviderID      string
	Config          provider.MCPConfig
	BackupPath      string
	WillCreate      bool
	SkipReason      string
	CLIAddArgs      []string
	CLIRemoveArgs   []string
}

type ApplyResult struct {
	Plan           ExecutionPlan
	Warnings       []string
	BackupPaths    []string
	Verification   []verify.Result
	UpdatedTargets []string
	RolledBack     []string
	RollbackFailed []string
}

type Manager struct {
	HomeDir     string
	Apps        []config.AppConfig
	Now         func() time.Time
	Runner      CommandRunner
	Logger      *slog.Logger
	WriteConfig WriteConfigFunc
}

type osRunner struct{}

type preparedWrite struct {
	op      Operation
	content []byte
}

func NewManager(homeDir string, now func() time.Time, runner CommandRunner) (*Manager, error) {
	if homeDir == "" {
		var err error
		homeDir, err = os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("resolve home directory: %w", err)
		}
	}
	if now == nil {
		now = time.Now
	}
	if runner == nil {
		runner = osRunner{}
	}

	apps, err := config.DetectAppConfigs(homeDir)
	if err != nil {
		return nil, err
	}

	return &Manager{
		HomeDir:     homeDir,
		Apps:        apps,
		Now:         now,
		Runner:      runner,
		Logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
		WriteConfig: config.WriteWithBackup,
	}, nil
}

func (m *Manager) PrepareProvider(
	prov provider.MCPProvider,
	profiles []provider.CredentialProfile,
	selected map[config.AppID]bool,
	assignments map[config.AppID]int,
) (ExecutionPlan, error) {
	if len(profiles) == 0 {
		return ExecutionPlan{}, fmt.Errorf("at least one credential profile is required")
	}

	plan := ExecutionPlan{}
	for _, appConfig := range m.Apps {
		if !selected[appConfig.ID] {
			continue
		}

		index, ok := assignments[appConfig.ID]
		if !ok {
			return ExecutionPlan{}, fmt.Errorf("missing credential assignment for %s", appConfig.Name)
		}
		if index < 0 || index >= len(profiles) {
			return ExecutionPlan{}, fmt.Errorf("invalid credential assignment for %s", appConfig.Name)
		}

		profile := profiles[index]
		cfg, err := prov.GenerateConfig(profile.Values)
		if err != nil {
			return ExecutionPlan{}, fmt.Errorf("generate config for %s: %w", prov.ID(), err)
		}

		if appConfig.ID == config.AppClaudeCode {
			op := Operation{
				AppID:           appConfig.ID,
				AppName:         appConfig.Name,
				FileLabel:       "Claude Code CLI",
				Kind:            config.FileKindClaudeCodeCLI,
				CredentialLabel: profile.Label,
				ProviderID:      prov.ID(),
				Config:          cfg,
				CLIRemoveArgs:   []string{"mcp", "remove", prov.ID(), "-s", "user"},
				CLIAddArgs:      []string{"mcp", "add", "--transport", string(cfg.Type), "-s", "user", prov.ID(), cfg.URL},
			}
			if _, err := m.Runner.LookPath("claude"); err != nil {
				op.SkipReason = "claude CLI not found; skipping direct mutation of ~/.claude.json"
				plan.Warnings = append(plan.Warnings, op.SkipReason)
			}
			plan.Operations = append(plan.Operations, op)
			continue
		}

		for _, file := range appConfig.Files {
			fileCfg := configForTarget(appConfig.ID, prov.ID(), cfg)
			plan.Operations = append(plan.Operations, Operation{
				AppID:           appConfig.ID,
				AppName:         appConfig.Name,
				FileLabel:       file.Label,
				Path:            file.Path,
				Kind:            file.Kind,
				CredentialLabel: profile.Label,
				ProviderID:      prov.ID(),
				Config:          fileCfg,
				BackupPath:      backupPathFor(file, m.Now()),
				WillCreate:      !file.Exists,
			})
		}
	}

	return plan, nil
}

func configForTarget(appID config.AppID, providerID string, cfg provider.MCPConfig) provider.MCPConfig {
	if providerID == "exa" && appID == config.AppClaudeDesktop && cfg.Type != provider.TransportStdio {
		return provider.MCPConfig{
			Type:    provider.TransportStdio,
			Command: "npx",
			Args:    []string{"-y", "mcp-remote", cfg.URL},
		}
	}
	return cfg
}

func (m *Manager) Prepare(keys []string, selected map[config.AppID]bool, assignments map[config.AppID]int) (ExecutionPlan, error) {
	if len(keys) == 0 {
		return ExecutionPlan{}, fmt.Errorf("at least one Exa API key is required")
	}

	prov := provider.NewExaProvider()
	profiles := make([]provider.CredentialProfile, len(keys))
	for i, key := range keys {
		profiles[i] = provider.CredentialProfile{
			ProviderID: prov.ID(),
			Values: map[string]string{
				"EXA_API_KEY": key,
			},
			Label: exa.RedactKey(key),
		}
	}

	return m.PrepareProvider(prov, profiles, selected, assignments)
}

func (m *Manager) Apply(plan ExecutionPlan) (ApplyResult, error) {
	result := ApplyResult{Plan: plan}
	m.logInfo("apply start", "operations", len(plan.Operations))

	prepared, cliOps, seenApps, err := m.prepareOperations(plan, &result)
	if err != nil {
		m.logError("apply preflight failed", "error", err.Error())
		return result, err
	}

	outcomes := make([]config.WriteOutcome, 0, len(prepared))
	seenFiles := make(map[string]config.FileKind, len(prepared))
	for _, item := range prepared {
		m.logDebug("writing target", "app", item.op.AppName, "target", item.op.FileLabel, "path", item.op.Path)
		outcome, err := m.WriteConfig(item.op.Path, item.content, m.Now())
		if err != nil {
			result.Warnings = append(result.Warnings, rollbackWarnings(m.rollback(outcomes, &result))...)
			m.logError("apply file write failed", "path", item.op.Path, "error", err.Error())
			return result, fmt.Errorf("%s (%s): %w", item.op.AppName, item.op.FileLabel, err)
		}

		outcomes = append(outcomes, outcome)
		seenFiles[item.op.Path] = item.op.Kind
		if outcome.BackupPath != "" {
			result.BackupPaths = append(result.BackupPaths, outcome.BackupPath)
		}
		result.UpdatedTargets = append(result.UpdatedTargets, item.op.Path)
	}

	for _, op := range cliOps {
		m.logInfo("running external cli update", "app", op.AppName)
		if err := m.applyClaudeCode(op, &result); err != nil {
			m.logError("external cli update failed", "app", op.AppName, "error", err.Error())
			return result, err
		}
	}

	result.Verification = append(result.Verification, verifyFiles(prepared)...)
	if seenApps[config.AppCodexCLI] {
		result.Verification = append(result.Verification, verify.VerifyOptionalCLI(m.Runner, "codex", "mcp", "get", "exa"))
	}
	if seenApps[config.AppClaudeCode] {
		result.Verification = append(result.Verification, verify.VerifyOptionalCLI(m.Runner, "claude", "mcp", "get", "exa"))
	}
	if seenApps[config.AppGeminiCLI] {
		result.Verification = append(result.Verification, verify.VerifyOptionalCLI(m.Runner, "gemini", "mcp", "get", "exa"))
	}

	m.logInfo("apply complete", "updated_targets", len(result.UpdatedTargets))
	return result, nil
}

func (m *Manager) applyClaudeCode(op Operation, result *ApplyResult) error {
	if _, err := m.Runner.Run("claude", op.CLIRemoveArgs...); err != nil {
		warning := fmt.Sprintf("claude mcp remove exa: %s", exa.RedactText(err.Error()))
		result.Warnings = append(result.Warnings, warning)
		m.logWarn("claude remove failed before add", "error", warning)
	}
	if _, err := m.Runner.Run("claude", op.CLIAddArgs...); err != nil {
		return fmt.Errorf("claude mcp add exa: %s", exa.RedactText(err.Error()))
	}
	result.UpdatedTargets = append(result.UpdatedTargets, "claude mcp add exa")
	return nil
}

func DefaultAssignments(selected map[config.AppID]bool, keyCount int) map[config.AppID]int {
	assignments := make(map[config.AppID]int)
	if keyCount <= 0 {
		return assignments
	}

	if keyCount == 1 {
		for appID, isSelected := range selected {
			if isSelected {
				assignments[appID] = 0
			}
		}
		return assignments
	}

	if keyCount == 2 {
		preferred := map[config.AppID]int{
			config.AppClaudeDesktop: 0,
			config.AppGeminiCLI:     0,
			config.AppCodexCLI:      0,
			config.AppClaudeCode:    1,
			config.AppAntigravity:   1,
		}
		for appID, isSelected := range selected {
			if isSelected {
				assignments[appID] = preferred[appID]
			}
		}
		return assignments
	}

	index := 0
	for _, appID := range config.AppOrder {
		if !selected[appID] {
			continue
		}
		assignments[appID] = index % keyCount
		index++
	}
	return assignments
}

func FormatPlan(plan ExecutionPlan) string {
	var builder strings.Builder
	builder.WriteString("Exa MCP update plan\n")
	builder.WriteString("===================\n")
	for _, warning := range plan.Warnings {
		builder.WriteString("warning: " + warning + "\n")
	}
	for _, op := range plan.Operations {
		fmt.Fprintf(&builder, "- %s: %s\n", op.AppName, op.FileLabel)
		fmt.Fprintf(&builder, "  credential: %s\n", op.CredentialLabel)
		if op.SkipReason != "" {
			fmt.Fprintf(&builder, "  skip: %s\n", op.SkipReason)
			continue
		}
		if op.Path != "" {
			fmt.Fprintf(&builder, "  path: %s\n", op.Path)
		}
		if op.WillCreate {
			builder.WriteString("  backup: not applicable (new file)\n")
		} else if op.BackupPath != "" {
			fmt.Fprintf(&builder, "  backup: %s\n", op.BackupPath)
		}
		fmt.Fprintf(&builder, "  tools: %d\n", len(exa.DefaultTools))
	}
	return strings.TrimRight(builder.String(), "\n")
}

func FormatApplyResult(result ApplyResult) string {
	var builder strings.Builder
	builder.WriteString("Exa MCP apply result\n")
	builder.WriteString("====================\n")

	for _, warning := range result.Warnings {
		builder.WriteString("warning: " + warning + "\n")
	}

	if len(result.BackupPaths) > 0 {
		builder.WriteString("Backups\n")
		for _, backup := range result.BackupPaths {
			builder.WriteString("- " + backup + "\n")
		}
	}

	if len(result.UpdatedTargets) > 0 {
		builder.WriteString("Updated\n")
		for _, target := range result.UpdatedTargets {
			builder.WriteString("- " + target + "\n")
		}
	}

	if len(result.RolledBack) > 0 {
		builder.WriteString("Rolled Back\n")
		for _, target := range result.RolledBack {
			builder.WriteString("- " + target + "\n")
		}
	}

	if len(result.RollbackFailed) > 0 {
		builder.WriteString("Rollback Failed\n")
		for _, target := range result.RollbackFailed {
			builder.WriteString("- " + target + "\n")
		}
	}

	if len(result.Verification) > 0 {
		builder.WriteString("Verification\n")
		for _, item := range result.Verification {
			fmt.Fprintf(&builder, "- [%s] %s\n", item.Status, item.Target)
			for _, detail := range item.Details {
				builder.WriteString("  " + detail + "\n")
			}
		}
	}

	builder.WriteString("Restart the affected apps to reload the updated MCP configuration.\n")
	return strings.TrimRight(builder.String(), "\n")
}

func LoadInitialKeys(keysCSV, keysFile string) ([]string, string, error) {
	if keysCSV != "" {
		keys, err := exa.ParseKeysCSV(keysCSV)
		return keys, keysCSV, err
	}
	if keysFile != "" {
		keys, err := exa.ParseKeysFile(keysFile)
		if err != nil {
			return nil, "", err
		}
		data, err := os.ReadFile(keysFile)
		if err != nil {
			return nil, "", err
		}
		return keys, string(data), nil
	}
	return nil, "", nil
}

func backupPathFor(file config.TargetFile, now time.Time) string {
	if !file.Exists {
		return ""
	}
	return config.BuildBackupPath(file.Path, now)
}

func (osRunner) LookPath(name string) (string, error) {
	return exec.LookPath(name)
}

func (osRunner) Run(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return exa.RedactText(string(output)), errors.New(exa.RedactText(strings.TrimSpace(string(output))))
	}
	return exa.RedactText(string(output)), nil
}

func (m *Manager) prepareOperations(plan ExecutionPlan, result *ApplyResult) ([]preparedWrite, []Operation, map[config.AppID]bool, error) {
	prepared := make([]preparedWrite, 0, len(plan.Operations))
	cliOps := make([]Operation, 0, 1)
	seenApps := make(map[config.AppID]bool)

	for _, op := range plan.Operations {
		if op.SkipReason != "" {
			result.Warnings = append(result.Warnings, op.SkipReason)
			continue
		}

		seenApps[op.AppID] = true
		switch op.Kind {
		case config.FileKindMCPServers, config.FileKindBareMCPServers, config.FileKindNamedServer, config.FileKindCodexTOML:
			item, err := m.prepareFileOperation(op)
			if err != nil {
				return nil, nil, nil, err
			}
			prepared = append(prepared, item)
		case config.FileKindClaudeCodeCLI:
			cliOps = append(cliOps, op)
		default:
			return nil, nil, nil, fmt.Errorf("unsupported operation kind %q", op.Kind)
		}
	}

	return prepared, cliOps, seenApps, nil
}

func (m *Manager) prepareFileOperation(op Operation) (preparedWrite, error) {
	if err := validatePathWithinHome(m.HomeDir, op.Path); err != nil {
		return preparedWrite{}, fmt.Errorf("%s (%s): %w", op.AppName, op.FileLabel, err)
	}

	data, _, err := config.ReadFileOrEmpty(op.Path)
	if err != nil {
		return preparedWrite{}, err
	}

	var updated []byte
	switch op.Kind {
	case config.FileKindMCPServers:
		rootKey := "mcpServers"
		urlFieldName := "url"
		var extra map[string]any

		switch op.AppID {
		case config.AppGeminiCLI:
			urlFieldName = "httpUrl"
		case config.AppAntigravity, config.AppWindsurf:
			urlFieldName = "serverUrl"
		case config.AppRooCode:
			extra = map[string]any{"type": "streamable-http"}
		}

		updated, err = config.UpdateMCPServersJSON(data, op.ProviderID, rootKey, urlFieldName, op.Config, extra)
	case config.FileKindBareMCPServers:
		urlFieldName := "url"
		if op.AppID == config.AppGeminiCLI {
			urlFieldName = "httpUrl"
		}
		updated, err = config.UpdateBareMCPServersJSON(data, op.ProviderID, urlFieldName, op.Config, nil)
	case config.FileKindNamedServer:
		rootKey := ""
		urlFieldName := "url"
		var extra map[string]any

		switch op.AppID {
		case config.AppVSCode:
			rootKey = "servers"
			extra = map[string]any{"type": "http"}
		case config.AppZed:
			rootKey = "context_servers"
		case config.AppOpenCode:
			rootKey = "mcp"
			extra = map[string]any{"type": "remote", "enabled": true}
		case config.AppAntigravity:
			// Backward compatibility: Antigravity used to be a named server but now we use FileKindMCPServers
			// with serverUrl if it's nested. If it's a legacy standalone file, this path still works.
			urlFieldName = "serverUrl"
		}
		updated, err = config.UpdateNamedServerJSON(data, op.ProviderID, rootKey, urlFieldName, op.Config, extra)
	case config.FileKindCodexTOML:
		updated, err = config.UpdateCodexTOML(data, op.ProviderID, op.Config)
	default:
		err = fmt.Errorf("unsupported file operation kind %q", op.Kind)
	}
	if err != nil {
		return preparedWrite{}, fmt.Errorf("%s (%s): %w", op.AppName, op.FileLabel, err)
	}

	return preparedWrite{op: op, content: updated}, nil
}

func (m *Manager) rollback(outcomes []config.WriteOutcome, result *ApplyResult) []string {
	warnings := make([]string, 0)
	for index := len(outcomes) - 1; index >= 0; index-- {
		outcome := outcomes[index]
		if err := config.RollbackWrite(outcome); err != nil {
			message := fmt.Sprintf("%s: %s", outcome.Path, exa.RedactText(err.Error()))
			result.RollbackFailed = append(result.RollbackFailed, message)
			warnings = append(warnings, "rollback failed for "+message)
			m.logError("rollback failed", "path", outcome.Path, "error", err.Error())
			continue
		}
		result.RolledBack = append(result.RolledBack, outcome.Path)
		m.logWarn("rollback restored file", "path", outcome.Path)
	}
	return warnings
}

func verifyFiles(prepared []preparedWrite) []verify.Result {
	// Sort by path for consistent output
	sort.Slice(prepared, func(i, j int) bool {
		return prepared[i].op.Path < prepared[j].op.Path
	})

	results := make([]verify.Result, 0, len(prepared))
	for _, item := range prepared {
		results = append(results, verify.VerifyProviderFile(item.op.Path, item.op.Kind, item.op.ProviderID, item.op.Config))
	}
	return results
}

func validatePathWithinHome(homeDir, target string) error {
	if target == "" {
		return fmt.Errorf("empty target path")
	}

	cleanHome := filepath.Clean(homeDir)
	cleanTarget := filepath.Clean(target)
	rel, err := filepath.Rel(cleanHome, cleanTarget)
	if err != nil {
		return fmt.Errorf("resolve target path: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("target path escapes configured home directory")
	}
	return nil
}

func rollbackWarnings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	return values
}

func (m *Manager) logDebug(msg string, attrs ...any) {
	m.log(slog.LevelDebug, msg, attrs...)
}

func (m *Manager) logInfo(msg string, attrs ...any) {
	m.log(slog.LevelInfo, msg, attrs...)
}

func (m *Manager) logWarn(msg string, attrs ...any) {
	m.log(slog.LevelWarn, msg, attrs...)
}

func (m *Manager) logError(msg string, attrs ...any) {
	m.log(slog.LevelError, msg, attrs...)
}

func (m *Manager) log(level slog.Level, msg string, attrs ...any) {
	if m.Logger == nil {
		return
	}
	m.Logger.Log(context.Background(), level, exa.RedactText(msg), redactAttrs(attrs)...)
}

func redactAttrs(attrs []any) []any {
	redacted := make([]any, 0, len(attrs))
	for _, attr := range attrs {
		if value, ok := attr.(string); ok {
			redacted = append(redacted, exa.RedactText(value))
			continue
		}
		redacted = append(redacted, attr)
	}
	return redacted
}
