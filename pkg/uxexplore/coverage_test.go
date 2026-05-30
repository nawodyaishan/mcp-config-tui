package uxexplore

import (
	"slices"
	"testing"
)

func TestComputeCoverage_NoFixturesYieldsAllGaps(t *testing.T) {
	c := ComputeCoverage(nil, nil)
	gaps := c.Gaps()
	for _, pc := range PreconditionClasses() {
		if !slices.Contains(gaps, pc) {
			t.Errorf("missing pc %q in gaps", pc)
		}
	}
}

func TestComputeCoverage_ExclusionRemovesGap(t *testing.T) {
	excluded := []Exclusion{{Screen: "CredentialEntry", PreconditionClass: PCScanError, Reason: "x"}}
	c := ComputeCoverage(nil, excluded)
	if slices.Contains(c.Gaps(), PCScanError) {
		t.Errorf("scan-error was excluded but appears in gaps")
	}
}

func TestComputeCoverage_NewPCConstantWithoutFixtureFails(t *testing.T) {
	// UXE-05 contract: every PreconditionClass constant must be reached by
	// at least one fixture OR be in the exclusions list. Adding a new
	// constant without a fixture causes Gaps() to grow.
	c := ComputeCoverage(nil, nil)
	if len(c.Gaps()) == 0 {
		t.Fatalf("empty fixture set must show gaps for every PC")
	}
}
