package uxexplore

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteFindings_EmitsAllFiveArtifacts(t *testing.T) {
	dir := t.TempDir()
	findings := []Finding{
		NewFinding(FindingDeadEnd, "f1", []string{"TargetSelect", "missing-credentials"}, StateFingerprint{Screen: "TargetSelect", PreconditionClass: "missing-credentials"}, "stuck"),
	}
	cov := ComputeCoverage(nil, nil)
	if err := WriteFindings(dir, findings, cov, nil); err != nil {
		t.Fatalf("WriteFindings: %v", err)
	}
	paths := PathsIn(dir)
	for _, p := range []string{paths.FindingsJSON, paths.FindingsMD, paths.ProposedMatrixRows, paths.CoverageJSON, paths.GraphDOT} {
		info, err := os.Stat(p)
		if err != nil {
			t.Errorf("missing artifact %s: %v", p, err)
			continue
		}
		if info.Size() == 0 {
			t.Errorf("empty artifact: %s", p)
		}
	}
	// findings.json is parseable + content matches.
	b, err := os.ReadFile(paths.FindingsJSON)
	if err != nil {
		t.Fatalf("read findings.json: %v", err)
	}
	var parsed []Finding
	if err := json.Unmarshal(b, &parsed); err != nil {
		t.Fatalf("parse findings.json: %v", err)
	}
	if len(parsed) != 1 || parsed[0].MatrixID == "" {
		t.Errorf("findings.json content unexpected: %+v", parsed)
	}
	// proposed-matrix-rows.md mentions the MatrixID
	md, _ := os.ReadFile(paths.ProposedMatrixRows)
	if !strings.Contains(string(md), findings[0].MatrixID) {
		t.Errorf("proposed-matrix-rows.md missing MatrixID")
	}
}

func TestWriteFindings_EmptyEmitsEmptyArtifacts(t *testing.T) {
	dir := t.TempDir()
	cov := ComputeCoverage(nil, nil)
	if err := WriteFindings(dir, nil, cov, nil); err != nil {
		t.Fatalf("WriteFindings: %v", err)
	}
	b, _ := os.ReadFile(filepath.Join(dir, "findings.json"))
	if strings.TrimSpace(string(b)) != "null" {
		t.Errorf("empty findings should marshal to null, got %q", b)
	}
}
