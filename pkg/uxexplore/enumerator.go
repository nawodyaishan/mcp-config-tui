package uxexplore

import "sort"

func EnumerateFixtures() []FixtureSpec {
	fixtures := []FixtureSpec{
		{Name: "happy-path-exa", Credentials: CredentialsValid, Provider: ProviderRequiresCreds, Conflicts: ConflictsNone, Targets: TargetsOne},
		{Name: "happy-path-no-key", Credentials: CredentialsNone, Provider: ProviderNoKey, Conflicts: ConflictsNone, Targets: TargetsOne},
		{Name: "no-creds-anchor", Credentials: CredentialsNone, Provider: ProviderRequiresCreds, Conflicts: ConflictsNone, Targets: TargetsOne},
		{Name: "conflict-then-resolve", Credentials: CredentialsValid, Provider: ProviderRequiresCreds, Conflicts: ConflictsOne, Targets: TargetsOne},
		{Name: "credential-and-conflict", Credentials: CredentialsNone, Provider: ProviderRequiresCreds, Conflicts: ConflictsOne, Targets: TargetsOne},
		{Name: "many-conflicts", Credentials: CredentialsValid, Provider: ProviderRequiresCreds, Conflicts: ConflictsMany, Targets: TargetsMany},
		{Name: "scan-error", Credentials: CredentialsNone, Provider: ProviderNoKey, Conflicts: ConflictsNone, Targets: TargetsNone, ScanError: true},
		{Name: "apply-error", Credentials: CredentialsValid, Provider: ProviderNoKey, Conflicts: ConflictsNone, Targets: TargetsOne, ApplyError: true},
		{Name: "plan-error", Credentials: CredentialsValid, Provider: ProviderRequiresCreds, Conflicts: ConflictsNone, Targets: TargetsOne, PlanError: true},
		{Name: "network-failure", Credentials: CredentialsValid, Provider: ProviderRequiresCreds, Conflicts: ConflictsNone, Targets: TargetsOne, NetworkFailure: true},
		{Name: "runtime-missing", Credentials: CredentialsNone, Provider: ProviderRuntimeMissing, Conflicts: ConflictsNone, Targets: TargetsOne},
		{Name: "no-targets-deselected", Credentials: CredentialsValid, Provider: ProviderRequiresCreds, Conflicts: ConflictsNone, Targets: TargetsNone},
		{Name: "workspace-on", Credentials: CredentialsValid, Provider: ProviderRequiresCreds, Conflicts: ConflictsNone, Targets: TargetsMixedChecked, Workspace: true},
		{Name: "invalid-credentials", Credentials: CredentialsInvalid, Provider: ProviderRequiresCreds, Conflicts: ConflictsNone, Targets: TargetsOne},
		{Name: "preflight-warning", Credentials: CredentialsValid, Provider: ProviderRequiresCreds, Conflicts: ConflictsNone, Targets: TargetsOne, PreflightWarnings: true},
		{Name: "many-targets", Credentials: CredentialsValid, Provider: ProviderRequiresCreds, Conflicts: ConflictsNone, Targets: TargetsMany},
		{Name: "no-key-many-targets", Credentials: CredentialsNone, Provider: ProviderNoKey, Conflicts: ConflictsNone, Targets: TargetsMany},
		{Name: "no-key-no-targets", Credentials: CredentialsNone, Provider: ProviderNoKey, Conflicts: ConflictsNone, Targets: TargetsNone},
		{Name: "runtime-missing-with-conflict", Credentials: CredentialsNone, Provider: ProviderRuntimeMissing, Conflicts: ConflictsOne, Targets: TargetsOne},
		{Name: "workspace-with-conflict", Credentials: CredentialsValid, Provider: ProviderRequiresCreds, Conflicts: ConflictsOne, Targets: TargetsMixedChecked, Workspace: true},
		{Name: "apply-error-workspace", Credentials: CredentialsValid, Provider: ProviderRequiresCreds, Conflicts: ConflictsNone, Targets: TargetsMixedChecked, Workspace: true, ApplyError: true},
		{Name: "plan-error-no-key", Credentials: CredentialsNone, Provider: ProviderNoKey, Conflicts: ConflictsNone, Targets: TargetsOne, PlanError: true},
		{Name: "network-failure-no-key", Credentials: CredentialsNone, Provider: ProviderNoKey, Conflicts: ConflictsNone, Targets: TargetsOne, NetworkFailure: true},
		{Name: "invalid-credentials-many-targets", Credentials: CredentialsInvalid, Provider: ProviderRequiresCreds, Conflicts: ConflictsNone, Targets: TargetsMany},
		{Name: "credential-workspace", Credentials: CredentialsNone, Provider: ProviderRequiresCreds, Conflicts: ConflictsNone, Targets: TargetsMixedChecked, Workspace: true},
		{Name: "conflict-no-key", Credentials: CredentialsNone, Provider: ProviderNoKey, Conflicts: ConflictsOne, Targets: TargetsOne},
		{Name: "scan-error-with-workspace", Credentials: CredentialsNone, Provider: ProviderNoKey, Conflicts: ConflictsNone, Targets: TargetsMixedChecked, Workspace: true, ScanError: true},
		{Name: "preflight-warning-many-targets", Credentials: CredentialsValid, Provider: ProviderRequiresCreds, Conflicts: ConflictsNone, Targets: TargetsMany, PreflightWarnings: true},
	}
	sort.Slice(fixtures, func(i, j int) bool {
		return fixtures[i].Name < fixtures[j].Name
	})
	return fixtures
}

func FixturePreconditionClasses(spec FixtureSpec) []string {
	classes := []string{PCOK}
	if spec.Credentials == CredentialsNone && spec.Provider == ProviderRequiresCreds {
		classes = append(classes, PCMissingCredentials)
	}
	if spec.Conflicts != ConflictsNone {
		classes = append(classes, PCConflictUnresolved)
	}
	if spec.Targets == TargetsNone {
		classes = append(classes, PCNoTargetsSelected)
	}
	if spec.ScanError {
		classes = append(classes, PCScanError)
	}
	if spec.PlanError {
		classes = append(classes, PCPlanError)
	}
	if spec.ApplyError {
		classes = append(classes, PCApplyError)
	}
	if spec.Provider == ProviderRuntimeMissing {
		classes = append(classes, PCRuntimeMissing)
	}
	if spec.NetworkFailure {
		classes = append(classes, PCNetworkFailure)
	}
	return uniqueStrings(classes)
}

func uniqueStrings(in []string) []string {
	seen := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))
	for _, v := range in {
		if seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}
