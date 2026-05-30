package uxexplore

import (
	"context"
	"errors"
	"time"

	"github.com/nawodyaishan/universal-mcp-sync/pkg/app"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/config"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/doctor"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/manifest"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/provider"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/validate"
)

type FakeScanner struct {
	Report doctor.Report
	Err    error
}

func (s FakeScanner) Scan(context.Context) (doctor.Report, error) {
	return s.Report, s.Err
}

type FakeDashboardManager struct {
	Home        string
	PlanErr     error
	ApplyErr    error
	ValidateErr error
}

func (m FakeDashboardManager) PrepareProvider(prov provider.MCPProvider, profiles []provider.CredentialProfile, selected map[config.AppID]bool, assign map[config.AppID]int) (app.ExecutionPlan, error) {
	return m.PrepareProviderWithTargetFiles(prov, profiles, selected, assign, nil)
}

func (m FakeDashboardManager) PrepareProviderWithTargetPaths(prov provider.MCPProvider, profiles []provider.CredentialProfile, selected map[config.AppID]bool, assign map[config.AppID]int, targetPaths app.TargetPathOverrides) (app.ExecutionPlan, error) {
	files := make(app.TargetFileOverrides)
	for id, path := range targetPaths {
		files[id] = []config.TargetFile{{Path: path}}
	}
	return m.PrepareProviderWithTargetFiles(prov, profiles, selected, assign, files)
}

func (m FakeDashboardManager) PrepareProviderWithTargetFiles(prov provider.MCPProvider, profiles []provider.CredentialProfile, selected map[config.AppID]bool, assign map[config.AppID]int, targetFiles app.TargetFileOverrides) (app.ExecutionPlan, error) {
	if m.PlanErr != nil {
		return app.ExecutionPlan{}, m.PlanErr
	}
	plan := app.ExecutionPlan{}
	for appID, files := range targetFiles {
		for _, file := range files {
			plan.Operations = append(plan.Operations, app.Operation{
				AppID:      appID,
				AppName:    config.AppName(appID),
				FileLabel:  file.Label,
				Path:       file.Path,
				Kind:       file.Kind,
				Scope:      file.Scope,
				ProviderID: prov.ID(),
			})
		}
	}
	return plan, nil
}

func (m FakeDashboardManager) BuildSavedPlan(plan app.ExecutionPlan, opts app.SavedPlanOptions) (app.SavedPlan, error) {
	created := opts.CreatedAt
	if created.IsZero() {
		created = time.Unix(0, 0).UTC()
	}
	return app.SavedPlan{
		SchemaVersion: 2,
		PlanID:        opts.PlanID,
		ProviderID:    opts.ProviderID,
		CreatedAt:     created,
		ExpiresAt:     created.Add(24 * time.Hour),
	}, nil
}

func (m FakeDashboardManager) PreflightSavedPlan(plan app.SavedPlan, opts app.SavedPlanApplyOptions) (app.SavedPlanPreflight, error) {
	return app.SavedPlanPreflight{PlanID: plan.PlanID, ProviderID: plan.ProviderID, CreatedAt: plan.CreatedAt, ExpiresAt: plan.ExpiresAt}, nil
}

func (m FakeDashboardManager) ApplySavedPlan(plan app.SavedPlan, opts app.SavedPlanApplyOptions) (app.ApplyResult, error) {
	if m.ApplyErr != nil {
		return app.ApplyResult{}, m.ApplyErr
	}
	return app.ApplyResult{UpdatedTargets: []string{"fixture-target"}}, nil
}

func (m FakeDashboardManager) Validate(ctx context.Context, prov provider.MCPProvider, profiles []provider.CredentialProfile, live bool) (validate.Report, error) {
	if m.ValidateErr != nil {
		return validate.Report{}, m.ValidateErr
	}
	return validate.Report{ProviderID: prov.ID(), Results: []validate.Result{{ProviderID: prov.ID(), Status: validate.StatusOK, Mode: validate.ModeOffline}}}, nil
}

func (m FakeDashboardManager) HomeDir() string {
	if m.Home == "" {
		return "."
	}
	return m.Home
}

func BuildScanner(spec FixtureSpec) FakeScanner {
	if spec.ScanError {
		return FakeScanner{Err: errors.New("scan failed")}
	}
	return FakeScanner{Report: fixtureReport(spec)}
}

func BuildManager(spec FixtureSpec, home string) FakeDashboardManager {
	mgr := FakeDashboardManager{Home: home}
	if spec.PlanError {
		mgr.PlanErr = errors.New("plan failed")
	}
	if spec.ApplyError {
		mgr.ApplyErr = errors.New("apply failed")
	}
	if spec.NetworkFailure {
		mgr.ValidateErr = errors.New("network failure")
	}
	return mgr
}

func BuildProfiles(spec FixtureSpec) []provider.CredentialProfile {
	if spec.Credentials != CredentialsValid {
		return nil
	}
	return []provider.CredentialProfile{{
		ProviderID: "exa",
		Values:     map[string]string{"EXA_API_KEY": "11111111-1111-1111-1111-111111111111"},
		Label:      "1111...1111",
	}}
}

func fixtureReport(spec FixtureSpec) doctor.Report {
	report := doctor.Report{Platform: "test"}
	if spec.Provider == ProviderRuntimeMissing {
		report.Runtimes = []doctor.RuntimeFinding{{ID: "docker", Name: "Docker", Available: false}}
	}
	report.Clients = append(report.Clients, doctor.ClientFinding{
		ID:            manifest.ClientAntigravityCLI,
		Name:          "Antigravity CLI",
		Confidence:    doctor.ConfidenceHigh,
		Installed:     true,
		EffectivePath: "/tmp/antigravity/mcp_config.json",
		Candidates: []doctor.CandidateFinding{{
			Label:    "user",
			Path:     "/tmp/antigravity/mcp_config.json",
			Scope:    manifest.ScopeUser,
			Exists:   true,
			ParseOK:  true,
			Writable: true,
		}},
	})
	if spec.Targets == TargetsMany || spec.Targets == TargetsMixedChecked {
		report.Clients = append(report.Clients, doctor.ClientFinding{
			ID:            manifest.ClientCursor,
			Name:          "Cursor",
			Confidence:    doctor.ConfidenceHigh,
			Installed:     true,
			EffectivePath: "/tmp/cursor/mcp.json",
			Candidates: []doctor.CandidateFinding{{
				Label:    "global",
				Path:     "/tmp/cursor/mcp.json",
				Scope:    manifest.ScopeGlobal,
				Exists:   true,
				ParseOK:  true,
				Writable: true,
			}},
		})
	}
	if spec.Conflicts != ConflictsNone {
		report.Clients = append(report.Clients, doctor.ClientFinding{
			ID:         manifest.ClientAntigravity,
			Name:       "Antigravity IDE",
			Confidence: doctor.ConfidenceConflict,
			Candidates: []doctor.CandidateFinding{{
				Label:   "repo-current",
				Path:    "/tmp/antigravity/conflict.json",
				Exists:  true,
				ParseOK: true,
			}},
		})
	}
	return report
}
