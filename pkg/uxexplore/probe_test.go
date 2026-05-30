package uxexplore

import (
	"context"
	"testing"
)

// UXE-07: the no-creds-anchor fixture must surface as a credential-blocked
// state during exploration. I-17's per-visit weak form cannot prove the dead-
// end (it only checks for empty action bars); the full I-17 over the edge
// graph runs in the analyzer (PR 14d). What we can assert here is that the
// probe reaches a state with PreconditionClass == missing-credentials and that
// no invariant catastrophically fails.
func TestProbe_NoCredsAnchorReachesMissingCredentialsState(t *testing.T) {
	d, err := NewDriver(FixtureSpec{Name: "no-creds-anchor", Credentials: CredentialsNone, Provider: ProviderRequiresCreds, Conflicts: ConflictsNone, Targets: TargetsOne})
	if err != nil {
		t.Fatalf("NewDriver: %v", err)
	}
	trace, err := d.Explore(context.Background())
	if err != nil {
		t.Fatalf("Explore: %v", err)
	}
	found := false
	for _, v := range trace.Visited {
		if v.Fingerprint.PreconditionClass == PCMissingCredentials {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("probe did not reach a missing-credentials state; visited=%d", len(trace.Visited))
	}
}

func TestProbe_VisitsAllReachableStatesFromHappyPath(t *testing.T) {
	d, err := NewDriver(FixtureSpec{Name: "happy", Credentials: CredentialsValid, Provider: ProviderRequiresCreds, Targets: TargetsOne})
	if err != nil {
		t.Fatalf("NewDriver: %v", err)
	}
	trace, err := d.Explore(context.Background())
	if err != nil {
		t.Fatalf("Explore: %v", err)
	}
	if len(trace.Visited) < 2 {
		t.Fatalf("expected >= 2 reachable states, got %d", len(trace.Visited))
	}
	screens := map[string]bool{}
	for _, v := range trace.Visited {
		screens[v.Fingerprint.Screen] = true
	}
	if !screens["Doctor"] {
		t.Errorf("Doctor screen not visited")
	}
}

func TestProbe_TerminatesOnEveryFixture(t *testing.T) {
	for _, fixture := range EnumerateFixtures() {
		t.Run(fixture.Name, func(t *testing.T) {
			d, err := NewDriver(fixture)
			if err != nil {
				t.Fatalf("NewDriver: %v", err)
			}
			if _, err := d.Explore(context.Background()); err != nil {
				t.Fatalf("Explore: %v", err)
			}
		})
	}
}

func TestProbe_UnmappedKeysDoNotAdvance(t *testing.T) {
	d, err := NewDriver(FixtureSpec{Name: "happy", Credentials: CredentialsValid, Provider: ProviderRequiresCreds, Targets: TargetsOne})
	if err != nil {
		t.Fatalf("NewDriver: %v", err)
	}
	trace, err := d.Explore(context.Background())
	if err != nil {
		t.Fatalf("Explore: %v", err)
	}
	for _, e := range trace.Errors {
		if e.Kind == "unmapped-key-advances" {
			t.Fatalf("unmapped key advanced state: %#v", e)
		}
	}
}

func TestIsTerminal(t *testing.T) {
	if !IsTerminal(StateFingerprint{Screen: "ApplyResult"}) {
		t.Errorf("ApplyResult must be terminal")
	}
	if !IsTerminal(StateFingerprint{Screen: "Doctor", PreconditionClass: PCScanError}) {
		t.Errorf("Doctor+scan-error must be terminal")
	}
	if IsTerminal(StateFingerprint{Screen: "Doctor", PreconditionClass: PCOK}) {
		t.Errorf("Doctor+ok must not be terminal")
	}
	if IsTerminal(StateFingerprint{Screen: "ProviderReady"}) {
		t.Errorf("ProviderReady must not be terminal")
	}
}
