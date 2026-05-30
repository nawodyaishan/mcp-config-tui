package uxexplore

import "testing"

func TestDetectDeadEnds_FindsStuckState(t *testing.T) {
	fp := StateFingerprint{Screen: "TargetSelect", PreconditionClass: PCMissingCredentials, BlockReason: "missing credentials"}
	trace := &Trace{
		Fixture: FixtureSpec{Name: "stuck"},
		Visited: []VisitedState{{Fingerprint: fp}},
		Edges: []Edge{
			{From: fp, Key: "enter", To: fp},
			{From: fp, Key: "esc", To: fp},
		},
	}
	got := detectDeadEnds(trace)
	if len(got) != 1 {
		t.Fatalf("want 1 dead-end finding, got %d", len(got))
	}
	if got[0].Kind != FindingDeadEnd {
		t.Fatalf("unexpected kind: %v", got[0].Kind)
	}
}

func TestDetectDeadEnds_ExcludesTerminal(t *testing.T) {
	fp := StateFingerprint{Screen: "ApplyResult", PreconditionClass: PCOK}
	trace := &Trace{
		Fixture: FixtureSpec{Name: "apply"},
		Visited: []VisitedState{{Fingerprint: fp}},
		Edges:   []Edge{{From: fp, Key: "r", To: fp}},
	}
	if got := detectDeadEnds(trace); len(got) != 0 {
		t.Fatalf("ApplyResult is terminal; want 0 findings, got %d", len(got))
	}
}

func TestDetectSilentNoops_QuietKeysIgnored(t *testing.T) {
	fp := StateFingerprint{Screen: "ProviderReady"}
	trace := &Trace{
		Edges: []Edge{
			{From: fp, Key: "up", To: fp},   // nav — quiet
			{From: fp, Key: "down", To: fp}, // nav — quiet
			{From: fp, Key: "tab", To: fp},  // quiet
			{From: fp, Key: "r", To: fp},    // refresh — quiet
		},
	}
	if got := detectSilentNoops(trace); len(got) != 0 {
		t.Fatalf("quiet keys must not produce silent-noop findings; got %d", len(got))
	}
}

func TestDetectSilentNoops_ViewChangeIgnored(t *testing.T) {
	fp := StateFingerprint{Screen: "TargetSelect", PreconditionClass: PCOK}
	trace := &Trace{
		Edges: []Edge{
			{From: fp, Key: "i", To: fp, FromViewDigest: "AAA", ToViewDigest: "BBB"},
		},
	}
	if got := detectSilentNoops(trace); len(got) != 0 {
		t.Fatalf("view-changing edge must not be silent-noop; got %d", len(got))
	}
}

func TestDetectSilentNoops_FlagsRealNoop(t *testing.T) {
	fp := StateFingerprint{Screen: "PlanPreview"}
	trace := &Trace{
		Edges: []Edge{
			{From: fp, Key: "enter", To: fp, FromViewDigest: "AAA", ToViewDigest: "AAA"},
		},
	}
	got := detectSilentNoops(trace)
	if len(got) != 1 {
		t.Fatalf("want 1 silent-noop, got %d", len(got))
	}
}

func TestDetectOrphans_FlagsZeroOutbound(t *testing.T) {
	fp := StateFingerprint{Screen: "ProviderReady"}
	trace := &Trace{
		Visited: []VisitedState{{Fingerprint: fp}},
	}
	got := detectOrphans(trace)
	if len(got) != 1 {
		t.Fatalf("want 1 orphan, got %d", len(got))
	}
}

func TestDetectErrorCycles_FindsSCC(t *testing.T) {
	a := StateFingerprint{Screen: "Doctor", PreconditionClass: PCScanError, HasError: true, BlockReason: "scan"}
	b := StateFingerprint{Screen: "Doctor", PreconditionClass: PCScanError, HasError: true, BlockReason: "scan", InFlight: "scanning"}
	trace := &Trace{
		Visited: []VisitedState{{Fingerprint: a}, {Fingerprint: b}},
		Edges: []Edge{
			{From: a, Key: "r", To: b},
			{From: b, Key: "r", To: a},
		},
	}
	got := detectErrorCycles(trace)
	if len(got) != 1 {
		t.Fatalf("want 1 error-cycle, got %d", len(got))
	}
	if got[0].Kind != FindingErrorCycle {
		t.Fatalf("kind = %v", got[0].Kind)
	}
}

func TestAnalyze_DeterministicAndDeduped(t *testing.T) {
	// Identical silent-noop edges must collapse to a single finding by
	// MatrixID. detectDeadEnds + detectSilentNoops will both fire on this
	// trace, so the expected total is 2 (one per kind).
	fp := StateFingerprint{Screen: "PlanPreview"}
	tr := &Trace{
		Fixture: FixtureSpec{Name: "x"},
		Visited: []VisitedState{{Fingerprint: fp}},
		Edges: []Edge{
			{From: fp, Key: "enter", To: fp, FromViewDigest: "A", ToViewDigest: "A"},
			{From: fp, Key: "enter", To: fp, FromViewDigest: "A", ToViewDigest: "A"},
		},
	}
	got := Analyze([]*Trace{tr})
	if len(got) != 2 {
		t.Fatalf("expected 1 dead-end + 1 silent-noop after dedupe, got %d", len(got))
	}
	first, second := Analyze([]*Trace{tr}), Analyze([]*Trace{tr})
	for i := range first {
		if first[i].MatrixID != second[i].MatrixID {
			t.Errorf("Analyze is not deterministic at index %d", i)
		}
	}
}
