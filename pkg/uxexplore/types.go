package uxexplore

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

type CredentialClass string
type ProviderClass string
type ConflictClass string
type TargetClass string

const (
	CredentialsNone    CredentialClass = "none"
	CredentialsValid   CredentialClass = "valid"
	CredentialsInvalid CredentialClass = "invalid"
)

const (
	ProviderRequiresCreds  ProviderClass = "requires-creds"
	ProviderNoKey          ProviderClass = "no-key"
	ProviderRuntimeMissing ProviderClass = "runtime-missing"
)

const (
	ConflictsNone ConflictClass = "none"
	ConflictsOne  ConflictClass = "one"
	ConflictsMany ConflictClass = "many"
)

const (
	TargetsNone         TargetClass = "none"
	TargetsOne          TargetClass = "one"
	TargetsMany         TargetClass = "many"
	TargetsMixedChecked TargetClass = "mixed-checked"
)

const (
	PCOK                 = "ok"
	PCMissingCredentials = "missing-credentials"
	PCConflictUnresolved = "conflict-unresolved"
	PCNoTargetsSelected  = "no-targets-selected"
	PCScanError          = "scan-error"
	PCPlanError          = "plan-error"
	PCApplyError         = "apply-error"
	PCRuntimeMissing     = "runtime-missing"
	PCNetworkFailure     = "network-failure"
)

type FixtureSpec struct {
	Name              string          `json:"name"`
	Credentials       CredentialClass `json:"credentials"`
	Provider          ProviderClass   `json:"provider"`
	Conflicts         ConflictClass   `json:"conflicts"`
	Targets           TargetClass     `json:"targets"`
	Workspace         bool            `json:"workspace"`
	ScanError         bool            `json:"scan_error"`
	ApplyError        bool            `json:"apply_error"`
	PlanError         bool            `json:"plan_error"`
	NetworkFailure    bool            `json:"network_failure"`
	PreflightWarnings bool            `json:"preflight_warnings"`
}

type StateFingerprint struct {
	Screen            string `json:"screen"`
	PreconditionClass string `json:"precondition_class"`
	BlockReason       string `json:"block_reason,omitempty"`
	HasError          bool   `json:"has_error"`
	InFlight          string `json:"in_flight,omitempty"`
}

type ModelSnap struct {
	Screen string `json:"screen"`
	View   string `json:"view,omitempty"`
}

type VisitedState struct {
	Fingerprint StateFingerprint `json:"fingerprint"`
	ViewDigest  string           `json:"view_digest"`
	ModelSnap   ModelSnap        `json:"model_snap"`
}

type Edge struct {
	From           StateFingerprint `json:"from"`
	Key            string           `json:"key"`
	To             StateFingerprint `json:"to"`
	Caused         string           `json:"caused"`
	FromViewDigest string           `json:"from_view_digest,omitempty"`
	ToViewDigest   string           `json:"to_view_digest,omitempty"`
}

type Trace struct {
	Fixture FixtureSpec     `json:"fixture"`
	Origin  string          `json:"origin,omitempty"` // "" = synthetic, "seeded" = seeded with a recorded transcript
	Visited []VisitedState  `json:"visited"`
	Edges   []Edge          `json:"edges"`
	Errors  []ExplorerError `json:"errors"`
}

type ExplorerError struct {
	Kind    string           `json:"kind"`
	Key     string           `json:"key,omitempty"`
	State   StateFingerprint `json:"state"`
	Message string           `json:"message"`
}

type FindingKind string

const (
	FindingDeadEnd               FindingKind = "dead-end"
	FindingSilentNoop            FindingKind = "silent-noop"
	FindingHiddenCursor          FindingKind = "hidden-cursor"
	FindingOrphan                FindingKind = "orphan"
	FindingErrorCycle            FindingKind = "error-cycle"
	FindingUnadvertisedKey       FindingKind = "unadvertised-key"
	FindingAdvertisedUnreachable FindingKind = "advertised-unreachable-key"
	FindingInvariantViolation    FindingKind = "invariant-violation"
)

type Finding struct {
	Kind           FindingKind      `json:"kind"`
	Fixture        string           `json:"fixture"`
	Path           []string         `json:"path"`
	State          StateFingerprint `json:"state"`
	Recommendation string           `json:"recommendation"`
	MatrixID       string           `json:"matrix_id"`
}

func NewFinding(kind FindingKind, fixture string, path []string, state StateFingerprint, recommendation string) Finding {
	f := Finding{
		Kind:           kind,
		Fixture:        fixture,
		Path:           append([]string(nil), path...),
		State:          state,
		Recommendation: recommendation,
	}
	f.MatrixID = MatrixID(f)
	return f
}

func MatrixID(f Finding) string {
	canonical := struct {
		Kind    FindingKind      `json:"kind"`
		Fixture string           `json:"fixture"`
		Path    []string         `json:"path"`
		State   StateFingerprint `json:"state"`
	}{
		Kind:    f.Kind,
		Fixture: f.Fixture,
		Path:    f.Path,
		State:   f.State,
	}
	b, _ := json.Marshal(canonical)
	sum := sha256.Sum256(b)
	return "DM-P" + hex.EncodeToString(sum[:4])
}

func PreconditionClasses() []string {
	return []string{
		PCOK,
		PCMissingCredentials,
		PCConflictUnresolved,
		PCNoTargetsSelected,
		PCScanError,
		PCPlanError,
		PCApplyError,
		PCRuntimeMissing,
		PCNetworkFailure,
	}
}
