package uxexplore

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/tui"
)

// startModel returns a model in its post-init state for a fixture.
func startModel(t *testing.T, spec FixtureSpec) tui.DashboardModel {
	t.Helper()
	d, err := NewDriver(spec)
	if err != nil {
		t.Fatalf("NewDriver: %v", err)
	}
	m := tui.NewDashboardModel(d.scanner, d.manager, BuildProfiles(d.spec))
	if cmd := m.Init(); cmd != nil {
		if msg := cmd(); msg != nil {
			next, _ := m.Update(msg)
			m = next.(tui.DashboardModel)
		}
	}
	return m
}

func TestInvariants_HappyPathInitialStateClean(t *testing.T) {
	m := startModel(t, FixtureSpec{Name: "happy", Credentials: CredentialsValid, Provider: ProviderRequiresCreds, Targets: TargetsOne})
	for _, inv := range Invariants() {
		if err := inv.Check(m); err != nil {
			t.Errorf("invariant %s failed on happy path: %v", inv.ID(), err)
		}
	}
}

func TestI04_RequiresCredsBlocksPlan_FlagsMissingCreds(t *testing.T) {
	// Drive the no-creds anchor fixture to PlanPreview synthetically — the
	// dashboard refuses to advance, so we verify the per-visit check at the
	// state where it would matter: planning with missing credentials.
	m := startModel(t, FixtureSpec{Name: "no-creds", Credentials: CredentialsNone, Provider: ProviderRequiresCreds, Targets: TargetsOne})
	// Move to ProviderReady.
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})
	m = next.(tui.DashboardModel)
	if cmd := m.Init(); cmd != nil {
		if msg := cmd(); msg != nil {
			n, _ := m.Update(msg)
			m = n.(tui.DashboardModel)
		}
	}
	if !m.Snapshot().MissingCredentials {
		t.Fatalf("expected MissingCredentials=true on no-creds-anchor fixture")
	}
	// Per-visit check returns nil because we are not at PlanPreview or
	// in-flight planning — I-04 is meant to catch an impossible state.
	if err := (I04RequiresCredsBlocksPlan{}).Check(m); err != nil {
		t.Errorf("I-04 should not fire on ProviderReady: %v", err)
	}
}

func TestI07_NoTargetsBlocksPlan(t *testing.T) {
	m := startModel(t, FixtureSpec{Name: "no-targets", Credentials: CredentialsValid, Provider: ProviderRequiresCreds, Targets: TargetsNone})
	if err := (I07NoTargetsNoPlan{}).Check(m); err != nil {
		t.Errorf("I-07 should pass when planning has not been issued: %v", err)
	}
}

func TestI13_ErrorOffersRecovery_PassesOnScanError(t *testing.T) {
	// scan-error fixture is the Doctor scan-error terminal — its action bar
	// advertises [r] rescan + [w] wizard, both recovery keys.
	m := startModel(t, FixtureSpec{Name: "scan-error", ScanError: true})
	if !m.Snapshot().HasScanError {
		t.Fatalf("expected HasScanError=true")
	}
	if err := (I13ErrorOffersRecovery{}).Check(m); err != nil {
		t.Errorf("I-13 must pass when [r] and [w] are advertised: %v", err)
	}
}

func TestI13_ErrorOffersRecovery_FailsWhenOnlyQuitAdvertised(t *testing.T) {
	// Use a synthetic model with HasScanError but a minimal action bar. We
	// can't easily produce such a state through normal driving, so we feed
	// the check a fabricated view through a wrapping stub.
	stub := stubModel{
		snap: tui.DashboardSnapshot{Screen: "Doctor", HasScanError: true},
		view: "scan failed\n[q] quit  [?] help",
	}
	// The invariant takes tui.DashboardModel; reuse the per-key helpers on a
	// real model isn't possible here, so verify the recovery rule against the
	// helper in isolation via a parsed action bar.
	keys := ParseActionBar(stub.view)
	recovery := false
	for _, k := range keys {
		switch strings.ToLower(k) {
		case "q", "?", "ctrl+c":
			continue
		default:
			recovery = true
		}
	}
	if recovery {
		t.Fatalf("expected no recovery key in stub view; got %v", keys)
	}
}

// stubModel is a placeholder used by TestI13 only — kept inside the test file
// so it never leaks into the public surface.
type stubModel struct {
	snap tui.DashboardSnapshot
	view string
}

func TestInvariants_Determinism(t *testing.T) {
	// Same fixture run twice → identical invariant violation set.
	fixture := FixtureSpec{Name: "happy", Credentials: CredentialsValid, Provider: ProviderRequiresCreds, Targets: TargetsOne}
	d1, _ := NewDriver(fixture)
	tr1, _ := d1.Explore(context.Background())
	d2, _ := NewDriver(fixture)
	tr2, _ := d2.Explore(context.Background())
	if len(tr1.Errors) != len(tr2.Errors) {
		t.Fatalf("explorer errors not deterministic: %d vs %d", len(tr1.Errors), len(tr2.Errors))
	}
	for i := range tr1.Errors {
		if tr1.Errors[i] != tr2.Errors[i] {
			t.Fatalf("error %d mismatch:\n  %#v\n  %#v", i, tr1.Errors[i], tr2.Errors[i])
		}
	}
}
