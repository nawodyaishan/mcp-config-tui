package uxexplore

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestEnumerateFixtures_CoversAllPreconditionClasses(t *testing.T) {
	fixtures := EnumerateFixtures()
	if len(fixtures) < 24 {
		t.Fatalf("expected at least 24 canonical fixtures, got %d", len(fixtures))
	}
	covered := map[string]bool{}
	for _, fixture := range fixtures {
		for _, class := range FixturePreconditionClasses(fixture) {
			covered[class] = true
		}
	}
	for _, class := range PreconditionClasses() {
		if !covered[class] {
			t.Fatalf("precondition class %q is not covered by canonical fixtures", class)
		}
	}
}

func TestEnumerateFixtures_Deterministic(t *testing.T) {
	first, err := json.Marshal(EnumerateFixtures())
	if err != nil {
		t.Fatalf("marshal first fixtures: %v", err)
	}
	second, err := json.Marshal(EnumerateFixtures())
	if err != nil {
		t.Fatalf("marshal second fixtures: %v", err)
	}
	if string(first) != string(second) {
		t.Fatalf("fixture enumeration is not deterministic\nfirst:  %s\nsecond: %s", first, second)
	}
}

func TestTypes_JSONRoundTrip(t *testing.T) {
	finding := NewFinding(FindingDeadEnd, "no-creds-anchor", []string{"p", "enter"}, StateFingerprint{
		Screen:            "TargetSelect",
		PreconditionClass: PCMissingCredentials,
		BlockReason:       "credentials required",
		HasError:          true,
	}, "add a recovery edge")
	trace := Trace{
		Fixture: EnumerateFixtures()[0],
		Visited: []VisitedState{{
			Fingerprint: finding.State,
			ViewDigest:  "digest",
			ModelSnap:   ModelSnap{Screen: "TargetSelect", View: "redacted"},
		}},
		Edges: []Edge{{
			From:   finding.State,
			Key:    "k",
			To:     StateFingerprint{Screen: "CredentialEntry", PreconditionClass: PCMissingCredentials},
			Caused: "screen-change",
		}},
		Errors: []ExplorerError{{
			Kind:    "invariant-violation",
			State:   finding.State,
			Message: "example",
		}},
	}

	payload := struct {
		Finding Finding `json:"finding"`
		Trace   Trace   `json:"trace"`
	}{Finding: finding, Trace: trace}

	b, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	var got struct {
		Finding Finding `json:"finding"`
		Trace   Trace   `json:"trace"`
	}
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if !reflect.DeepEqual(payload, got) {
		t.Fatalf("JSON round trip mismatch\nwant: %#v\ngot:  %#v", payload, got)
	}
	if !strings.HasPrefix(got.Finding.MatrixID, "DM-P") {
		t.Fatalf("expected matrix id prefix DM-P, got %q", got.Finding.MatrixID)
	}
}

func TestParseExclusions(t *testing.T) {
	exclusions, err := ParseExclusions(strings.NewReader(`exclusions:
  - screen: CredentialEntry
    precondition_class: scan-error
    reason: "credential entry only reachable after successful scan"
`))
	if err != nil {
		t.Fatalf("ParseExclusions: %v", err)
	}
	if len(exclusions) != 1 {
		t.Fatalf("expected one exclusion, got %#v", exclusions)
	}
	if exclusions[0].Screen != "CredentialEntry" || exclusions[0].PreconditionClass != PCScanError || exclusions[0].Reason == "" {
		t.Fatalf("unexpected exclusion: %#v", exclusions[0])
	}
}
