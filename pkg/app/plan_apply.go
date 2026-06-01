package app

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/nawodyaishan/universal-mcp-sync/pkg/audit"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/client"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/config"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/provider"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/redact"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/verify"
)

type Approver interface {
	Confirm(prompt ApprovalPrompt) (bool, error)
}

type ApprovalPrompt struct {
	Reason       string `json:"reason"`
	TargetPath   string `json:"target_path"`
	ResolvedPath string `json:"resolved_path,omitempty"`
	Message      string `json:"message"`
}

type SavedPlanPreflight struct {
	PlanID          string           `json:"plan_id"`
	ProviderID      string           `json:"provider_id"`
	CreatedAt       time.Time        `json:"created_at"`
	ExpiresAt       time.Time        `json:"expires_at"`
	Operations      []PlanOperation  `json:"operations"`
	Warnings        []string         `json:"warnings,omitempty"`
	ApprovalPrompts []ApprovalPrompt `json:"approval_prompts,omitempty"`
}

type SavedPlanApplyOptions struct {
	Credentials map[string]string
	AutoApprove bool
	DryRun      bool
	ForceStale  bool
	Approver    Approver
	Command     string
}

type savedPreparedWrite struct {
	prepared    preparedWrite
	verifyPath  string
	displayPath string
}

type savedPreparedApply struct {
	preflight SavedPlanPreflight
	files     []savedPreparedWrite
	cliOps    []Operation
	seenApps  map[config.AppID]bool
}

func (m *Manager) PreflightSavedPlan(plan SavedPlan, opts SavedPlanApplyOptions) (SavedPlanPreflight, error) {
	if err := verifySavedPlanContentHash(plan); err != nil {
		return SavedPlanPreflight{}, err
	}
	prepared, err := m.prepareSavedPlan(plan, opts)
	if err != nil {
		return SavedPlanPreflight{}, err
	}
	return prepared.preflight, nil
}

// verifySavedPlanContentHash recomputes the SHA-256 of all Redacted strings and
// compares it to plan.ContentHash. Skipped when ContentHash is empty (backward compat).
func verifySavedPlanContentHash(plan SavedPlan) error {
	if plan.ContentHash == "" {
		return nil
	}
	h := sha256.New()
	for _, op := range plan.Operations {
		_, _ = io.WriteString(h, op.Redacted)
	}
	computed := "sha256:" + hex.EncodeToString(h.Sum(nil))
	if computed != plan.ContentHash {
		return fmt.Errorf("saved plan content hash mismatch: plan may have been modified or provider configuration has drifted")
	}
	return nil
}

func (m *Manager) ApplySavedPlan(plan SavedPlan, opts SavedPlanApplyOptions) (result ApplyResult, err error) {
	result = ApplyResult{
		Plan: ExecutionPlan{
			Warnings: append([]string(nil), plan.Warnings...),
		},
		Warnings: append([]string(nil), plan.Warnings...),
	}
	defer func() {
		auditErr := m.writeSavedPlanAudit(plan, opts, result, err)
		if auditErr != nil && err == nil {
			result.Warnings = append(result.Warnings, "audit log: "+redact.Text(auditErr.Error()))
		}
	}()

	if err = verifySavedPlanContentHash(plan); err != nil {
		return result, err
	}

	prepared, err := m.prepareSavedPlan(plan, opts)
	if err != nil {
		return result, err
	}
	if err := confirmApprovalPrompts(prepared.preflight.ApprovalPrompts, opts); err != nil {
		return result, err
	}

	outcomes := make([]config.WriteOutcome, 0, len(prepared.files))
	for _, item := range prepared.files {
		if item.prepared.skipped {
			result.SkippedTargets = append(result.SkippedTargets, item.displayPath)
			continue
		}
		outcome, err := m.WriteConfig(item.prepared.op.Path, item.prepared.content, m.Now())
		if err != nil {
			result.Warnings = append(result.Warnings, rollbackWarnings(m.rollback(outcomes, &result))...)
			return result, fmt.Errorf("%s (%s): %w", item.prepared.op.AppName, item.prepared.op.FileLabel, err)
		}

		outcomes = append(outcomes, outcome)
		if outcome.BackupPath != "" {
			result.BackupPaths = append(result.BackupPaths, outcome.BackupPath)
		}
		result.UpdatedTargets = append(result.UpdatedTargets, item.displayPath)
	}

	fileVerification := verifySavedPreparedFiles(prepared.files)
	result.Verification = append(result.Verification, fileVerification...)
	if verifyErr := firstFileVerificationError(fileVerification); verifyErr != nil {
		result.Warnings = append(result.Warnings, rollbackWarnings(m.rollback(outcomes, &result))...)
		return result, verifyErr
	}

	for _, op := range prepared.cliOps {
		if err := m.applySavedCLIOperation(op, &result); err != nil {
			return result, err
		}
	}

	result.Verification = append(result.Verification, verifySavedCLI(prepared.files, prepared.cliOps, prepared.seenApps, m.Runner)...)
	return result, nil
}

func (m *Manager) writeSavedPlanAudit(plan SavedPlan, opts SavedPlanApplyOptions, result ApplyResult, applyErr error) error {
	writer, err := audit.NewWriter(m.HomeDir)
	if err != nil {
		return err
	}

	command := opts.Command
	if strings.TrimSpace(command) == "" {
		command = "usync apply --plan"
	}

	targets := make([]string, 0, len(plan.Operations))
	for _, op := range plan.Operations {
		if op.TargetID != "" {
			targets = append(targets, op.TargetID)
		}
	}

	filesTouched := make([]string, 0, len(result.UpdatedTargets))
	for _, target := range result.UpdatedTargets {
		if strings.Contains(target, string(os.PathSeparator)) {
			filesTouched = append(filesTouched, target)
		}
	}

	entry := audit.Entry{
		Timestamp:    m.Now().UTC(),
		Command:      command,
		PlanID:       plan.PlanID,
		Targets:      targets,
		FilesTouched: filesTouched,
		ExitCode:     0,
	}
	if applyErr != nil {
		entry.ExitCode = 1
		entry.Error = redact.Text(applyErr.Error())
	}
	return writer.Append(entry)
}

func (m *Manager) prepareSavedPlan(plan SavedPlan, opts SavedPlanApplyOptions) (savedPreparedApply, error) {
	if plan.SchemaVersion != SavedPlanSchemaVersion {
		return savedPreparedApply{}, fmt.Errorf("saved-plan apply requires schema version %d, got %d", SavedPlanSchemaVersion, plan.SchemaVersion)
	}
	if !opts.ForceStale && !plan.ExpiresAt.IsZero() && plan.ExpiresAt.Before(m.Now().UTC()) {
		return savedPreparedApply{}, fmt.Errorf("saved plan %s expired at %s", plan.PlanID, plan.ExpiresAt.UTC().Format(time.RFC3339))
	}

	preflight := SavedPlanPreflight{
		PlanID:     plan.PlanID,
		ProviderID: plan.ProviderID,
		CreatedAt:  plan.CreatedAt,
		ExpiresAt:  plan.ExpiresAt,
		Operations: append([]PlanOperation(nil), plan.Operations...),
		Warnings:   append([]string(nil), plan.Warnings...),
	}

	prepared := savedPreparedApply{
		preflight: preflight,
		files:     make([]savedPreparedWrite, 0, len(plan.Operations)),
		cliOps:    make([]Operation, 0, len(plan.Operations)),
		seenApps:  make(map[config.AppID]bool),
	}

	for _, planOp := range plan.Operations {
		if planOp.Action == PlanActionSkip {
			prepared.preflight.Warnings = append(prepared.preflight.Warnings, planOp.Warnings...)
			continue
		}
		if planOp.Action == PlanActionConflict {
			return savedPreparedApply{}, fmt.Errorf("%s: saved plan contains unresolved conflict operation", planOp.TargetName)
		}

		built, err := m.buildOperationFromSavedPlan(plan, planOp, opts.Credentials)
		if err != nil {
			return savedPreparedApply{}, err
		}
		prepared.seenApps[built.AppID] = true

		switch planOp.Manager {
		case PlanManagerFile:
			if err := validateOperationPath(m.HomeDir, planOp.TargetScope, planOp.FilePath); err != nil {
				return savedPreparedApply{}, fmt.Errorf("%s: %w", planOp.TargetName, err)
			}
			appendScopeApprovalPrompt(&prepared.preflight, planOp)

			sha, err := currentSHA(planOp.FilePath)
			if err != nil {
				return savedPreparedApply{}, err
			}
			if sha != planOp.CurrentSHA {
				return savedPreparedApply{}, fmt.Errorf("%s: target changed since plan creation for %s", planOp.TargetName, planOp.FilePath)
			}

			isSymlink, resolvedPath, err := symlinkStatus(planOp.FilePath)
			if err != nil {
				return savedPreparedApply{}, err
			}
			if isSymlink != planOp.IsSymlink {
				return savedPreparedApply{}, fmt.Errorf("%s: symlink status changed for %s", planOp.TargetName, planOp.FilePath)
			}
			if planOp.IsSymlink && filepath.Clean(resolvedPath) != filepath.Clean(planOp.ResolvedPath) {
				return savedPreparedApply{}, fmt.Errorf("%s: symlink target changed for %s", planOp.TargetName, planOp.FilePath)
			}

			writePath := planOp.FilePath
			if planOp.IsSymlink {
				if planOp.ResolvedPath == "" {
					return savedPreparedApply{}, fmt.Errorf("%s: missing resolved symlink target for %s", planOp.TargetName, planOp.FilePath)
				}
				if err := validateOperationPath(m.HomeDir, planOp.TargetScope, planOp.ResolvedPath); err != nil {
					return savedPreparedApply{}, fmt.Errorf("%s: %w", planOp.TargetName, err)
				}
				writePath = planOp.ResolvedPath
				prepared.preflight.ApprovalPrompts = append(prepared.preflight.ApprovalPrompts, ApprovalPrompt{
					Reason:       "symlink",
					TargetPath:   planOp.FilePath,
					ResolvedPath: planOp.ResolvedPath,
					Message:      fmt.Sprintf("Apply changes through symlink %s to %s", planOp.FilePath, planOp.ResolvedPath),
				})
			}
			if planOp.WillCreate {
				prepared.preflight.ApprovalPrompts = append(prepared.preflight.ApprovalPrompts, ApprovalPrompt{
					Reason:     "create",
					TargetPath: planOp.FilePath,
					Message:    fmt.Sprintf("Create new config file %s", planOp.FilePath),
				})
			}

			opForWrite := built
			opForWrite.Path = writePath
			item, err := m.prepareFileOperation(opForWrite, plan.PlanID)
			if err != nil {
				return savedPreparedApply{}, err
			}
			// B2 Phase B: merge VS Code inputs block into the root of the config file.
			if len(planOp.VSCodeInputs) > 0 {
				merged, mergeErr := config.MergeVSCodeInputs(item.content, planOp.VSCodeInputs)
				if mergeErr != nil {
					return savedPreparedApply{}, fmt.Errorf("%s: merge VS Code inputs: %w", planOp.TargetName, mergeErr)
				}
				item.content = merged
			}
			prepared.files = append(prepared.files, savedPreparedWrite{
				prepared:    item,
				verifyPath:  planOp.FilePath,
				displayPath: planOp.FilePath,
			})
		case PlanManagerCLI:
			appendScopeApprovalPrompt(&prepared.preflight, planOp)
			if err := validateSavedCLIOperation(built, planOp, m.Runner); err != nil {
				return savedPreparedApply{}, err
			}
			prepared.cliOps = append(prepared.cliOps, built)
		default:
			return savedPreparedApply{}, fmt.Errorf("%s: unsupported saved plan manager %q", planOp.TargetName, planOp.Manager)
		}
	}

	return prepared, nil
}

func (m *Manager) buildOperationFromSavedPlan(plan SavedPlan, planOp PlanOperation, supplied map[string]string) (Operation, error) {
	appID := config.AppID(planOp.TargetID)
	providerID := strings.TrimSpace(planOp.ProviderID)
	if providerID == "" {
		providerID = plan.ProviderID
	}
	if providerID == "" {
		return Operation{}, fmt.Errorf("%s: missing provider id", planOp.TargetName)
	}

	registry := provider.DefaultRegistry()
	prov, ok := registry.Get(providerID)
	if !ok {
		return Operation{}, fmt.Errorf("%s: unsupported provider %q", planOp.TargetName, providerID)
	}

	credentialLabel := ""
	credentials := make(map[string]string)
	if planOp.CredentialRef != "" || len(plan.Credentials) > 0 {
		ref, value, err := resolveSavedPlanCredential(plan, planOp, supplied)
		if err != nil {
			return Operation{}, err
		}
		if ref.Key != "" {
			credentials[ref.Key] = value
		}
		credentialLabel = ref.Label
	}

	cfg, err := prov.GenerateConfig(credentials)
	if err != nil {
		return Operation{}, fmt.Errorf("%s: generate config for %s: %w", planOp.TargetName, providerID, err)
	}

	fileLabel := savedPlanFileLabel(m.Apps, appID, planOp.FilePath)
	op := Operation{
		AppID:           appID,
		AppName:         planOp.TargetName,
		FileLabel:       fileLabel,
		Path:            planOp.FilePath,
		Kind:            config.FileKind(planOp.FileKind),
		Scope:           planOp.TargetScope,
		GitWarning:      planOp.GitWarning,
		CredentialLabel: credentialLabel,
		ProviderID:      providerID,
		BackupPath:      planOp.BackupPath,
		WillCreate:      planOp.WillCreate,
	}

	switch planOp.Manager {
	case PlanManagerFile:
		op.Config = client.Adapt(appID, cfg)
		if op.Kind == "" {
			return Operation{}, fmt.Errorf("%s: missing file kind for %s", planOp.TargetName, planOp.FilePath)
		}
		if string(op.Config.Type) != planOp.Transport {
			return Operation{}, fmt.Errorf("%s: transport changed since plan creation for %s", planOp.TargetName, planOp.FilePath)
		}
		// B2-A: substitute credential header values with ${input:id} references when
		// the plan records VSCodeInputs. This ensures the real key is never written to the
		// VS Code config file; VS Code resolves ${input:id} securely on first server start.
		if len(planOp.VSCodeInputs) > 0 && op.Config.Type != provider.TransportStdio {
			subHeaders := make(map[string]string, len(op.Config.Headers))
			for k := range op.Config.Headers {
				subHeaders[k] = "${input:" + planOp.VSCodeInputs[0].ID + "}"
			}
			op.Config.Headers = subHeaders
		}
	case PlanManagerCLI:
		op.Kind = config.FileKind(planOp.FileKind)
		op.Config = cfg
		if appID != config.AppClaudeCode || op.Kind != config.FileKindClaudeCodeCLI {
			return Operation{}, fmt.Errorf("%s: unsupported CLI target %s", planOp.TargetName, appID)
		}
		op.CLIRemoveArgs = []string{"mcp", "remove", providerID, "-s", claudeCodeCLIScope(op.Scope)}
		op.CLIAddArgs = buildClaudeCodeAddArgs(providerID, cfg, op.Scope)
		if string(cfg.Type) != planOp.Transport {
			return Operation{}, fmt.Errorf("%s: CLI transport changed since plan creation", planOp.TargetName)
		}
	default:
		return Operation{}, fmt.Errorf("%s: unsupported saved plan manager %q", planOp.TargetName, planOp.Manager)
	}

	return op, nil
}

func resolveSavedPlanCredential(plan SavedPlan, planOp PlanOperation, supplied map[string]string) (CredentialRef, string, error) {
	ref, ok := savedPlanCredentialRef(plan, planOp.CredentialRef)
	if !ok {
		if planOp.CredentialRef == "" && len(plan.Credentials) == 0 {
			return CredentialRef{}, "", nil
		}
		return CredentialRef{}, "", fmt.Errorf("%s: missing credential reference", planOp.TargetName)
	}

	value, ok := supplied[ref.ID]
	if !ok && ref.ID == "" {
		value, ok = supplied[defaultCredentialRefID(ref.Key, ref.Label)]
	}
	if !ok {
		value, ok = supplied[ref.Key]
	}
	if !ok {
		value, ok = supplied[ref.EnvVar]
	}
	if !ok {
		return CredentialRef{}, "", fmt.Errorf("%s: missing credential for %s", planOp.TargetName, ref.Label)
	}
	return ref, value, nil
}

func savedPlanCredentialRef(plan SavedPlan, id string) (CredentialRef, bool) {
	if id == "" {
		if len(plan.Credentials) == 1 {
			return normalizedCredentialRef(plan.Credentials[0]), true
		}
		return CredentialRef{}, false
	}
	for _, ref := range plan.Credentials {
		ref = normalizedCredentialRef(ref)
		if ref.ID == id {
			return ref, true
		}
	}
	return CredentialRef{}, false
}

func normalizedCredentialRef(ref CredentialRef) CredentialRef {
	if ref.ID == "" {
		ref.ID = defaultCredentialRefID(ref.Key, ref.Label)
	}
	return ref
}

func savedPlanFileLabel(apps []config.AppConfig, appID config.AppID, path string) string {
	for _, appConfig := range apps {
		if appConfig.ID != appID {
			continue
		}
		for _, file := range appConfig.Files {
			if filepath.Clean(file.Path) == filepath.Clean(path) {
				return file.Label
			}
		}
	}
	if path == "" {
		return "Saved plan operation"
	}
	return filepath.Base(path)
}

func appendScopeApprovalPrompt(preflight *SavedPlanPreflight, planOp PlanOperation) {
	scope := planOp.TargetScope
	if scope != "project" && scope != "workspace" && !planOp.GitWarning {
		return
	}

	path := planOp.FilePath
	if path == "" {
		path = planOp.TargetName
	}

	message := fmt.Sprintf("Apply changes to %s-scoped config %s", scopeLabel(scope), path)
	if planOp.GitWarning {
		message = fmt.Sprintf("%s (this file may be shared through source control)", message)
	}

	preflight.ApprovalPrompts = append(preflight.ApprovalPrompts, ApprovalPrompt{
		Reason:     "scope",
		TargetPath: path,
		Message:    message,
	})
}

func scopeLabel(scope string) string {
	if scope == "" {
		return "user"
	}
	return scope
}

func confirmApprovalPrompts(prompts []ApprovalPrompt, opts SavedPlanApplyOptions) error {
	if len(prompts) == 0 || opts.DryRun || opts.AutoApprove {
		return nil
	}
	if opts.Approver == nil {
		return fmt.Errorf("apply requires approval for %d operation(s); rerun with --auto-approve or an interactive approver", len(prompts))
	}
	for _, prompt := range prompts {
		ok, err := opts.Approver.Confirm(prompt)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("apply cancelled: %s", prompt.Message)
		}
	}
	return nil
}

func validateSavedCLIOperation(op Operation, planOp PlanOperation, runner CommandRunner) error {
	if _, err := runner.LookPath("claude"); err != nil {
		return fmt.Errorf("%s: claude CLI not found", planOp.TargetName)
	}
	if len(planOp.CLICommand) == 0 {
		return nil
	}
	want := redactStrings(append([]string{"claude"}, op.CLIAddArgs...))
	if len(want) != len(planOp.CLICommand) {
		return fmt.Errorf("%s: CLI command changed since plan creation", planOp.TargetName)
	}
	for i := range want {
		if want[i] != planOp.CLICommand[i] {
			return fmt.Errorf("%s: CLI command changed since plan creation", planOp.TargetName)
		}
	}
	return nil
}

func verifySavedPreparedFiles(prepared []savedPreparedWrite) []verify.Result {
	sort.Slice(prepared, func(i, j int) bool {
		return prepared[i].verifyPath < prepared[j].verifyPath
	})

	results := make([]verify.Result, 0, len(prepared))
	for _, item := range prepared {
		results = append(results, verify.VerifyProviderFile(
			item.verifyPath,
			item.prepared.op.Kind,
			item.prepared.op.ProviderID,
			item.prepared.op.Config,
		))
	}
	return results
}

func firstFileVerificationError(results []verify.Result) error {
	for _, item := range results {
		if item.Status != verify.StatusOK {
			return fmt.Errorf("%s verification failed: %s", item.Target, strings.Join(item.Details, "; "))
		}
	}
	return nil
}

func (m *Manager) applySavedCLIOperation(op Operation, result *ApplyResult) error {
	switch op.AppID {
	case config.AppClaudeCode:
		return m.applyClaudeCode(op, result)
	default:
		return fmt.Errorf("%s: unsupported saved CLI operation", op.AppName)
	}
}

func verifySavedCLI(fileWrites []savedPreparedWrite, cliOps []Operation, seenApps map[config.AppID]bool, runner CommandRunner) []verify.Result {
	results := make([]verify.Result, 0, len(cliOps)+len(seenApps))
	type cliVerifyKey struct {
		appID      config.AppID
		providerID string
	}
	seen := make(map[cliVerifyKey]bool)

	for _, op := range cliOps {
		key := cliVerifyKey{appID: op.AppID, providerID: op.ProviderID}
		if seen[key] {
			continue
		}
		seen[key] = true
		switch op.AppID {
		case config.AppClaudeCode:
			results = append(results, verify.VerifyOptionalCLI(runner, "claude", "mcp", "get", op.ProviderID))
		}
	}

	providerByAppID := make(map[config.AppID]string, len(fileWrites)+len(cliOps))
	for _, item := range fileWrites {
		if item.prepared.op.ProviderID != "" {
			providerByAppID[item.prepared.op.AppID] = item.prepared.op.ProviderID
		}
	}
	for _, op := range cliOps {
		if op.ProviderID != "" {
			providerByAppID[op.AppID] = op.ProviderID
		}
	}

	for appID := range seenApps {
		providerID := providerByAppID[appID]
		if providerID == "" {
			continue
		}

		key := cliVerifyKey{appID: appID, providerID: providerID}
		if seen[key] {
			continue
		}
		seen[key] = true

		switch appID {
		case config.AppCodexCLI:
			results = append(results, verify.VerifyOptionalCLI(runner, "codex", "mcp", "get", providerID))
		case config.AppAntigravityCLI:
			results = append(results, verify.VerifyOptionalCLI(runner, "antigravity", "mcp", "get", providerID))
		}
	}

	return results
}
