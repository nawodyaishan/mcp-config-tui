package uxexplore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ReportPaths returns the canonical artifact paths under dir.
type ReportPaths struct {
	FindingsJSON       string
	FindingsMD         string
	ProposedMatrixRows string
	CoverageJSON       string
	GraphDOT           string
}

func PathsIn(dir string) ReportPaths {
	return ReportPaths{
		FindingsJSON:       filepath.Join(dir, "findings.json"),
		FindingsMD:         filepath.Join(dir, "findings.md"),
		ProposedMatrixRows: filepath.Join(dir, "proposed-matrix-rows.md"),
		CoverageJSON:       filepath.Join(dir, "coverage.json"),
		GraphDOT:           filepath.Join(dir, "graph.dot"),
	}
}

// WriteFindings writes the five artifact files under dir.
func WriteFindings(dir string, findings []Finding, coverage Coverage, traces []*Trace) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	paths := PathsIn(dir)
	if err := writeJSON(paths.FindingsJSON, findings); err != nil {
		return err
	}
	if err := writeFindingsMD(paths.FindingsMD, findings); err != nil {
		return err
	}
	if err := writeMatrixRowStubs(paths.ProposedMatrixRows, findings); err != nil {
		return err
	}
	if err := writeCoverageJSON(paths.CoverageJSON, coverage); err != nil {
		return err
	}
	if err := writeGraphDOT(paths.GraphDOT, traces); err != nil {
		return err
	}
	return nil
}

func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(path, b, 0o644)
}

func writeFindingsMD(path string, findings []Finding) error {
	var b strings.Builder
	b.WriteString("# UX Explorer Findings\n\n")
	if len(findings) == 0 {
		b.WriteString("_No findings._\n")
		return os.WriteFile(path, []byte(b.String()), 0o644)
	}
	fmt.Fprintf(&b, "Total: %d\n\n", len(findings))
	b.WriteString("| MatrixID | Kind | Fixture | Screen | PC | Recommendation |\n")
	b.WriteString("|---|---|---|---|---|---|\n")
	for _, f := range findings {
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s |\n",
			f.MatrixID, f.Kind, f.Fixture, f.State.Screen, f.State.PreconditionClass, mdEscape(f.Recommendation))
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func mdEscape(s string) string {
	s = strings.ReplaceAll(s, "|", "\\|")
	s = strings.ReplaceAll(s, "\n", " ")
	return s
}

func writeMatrixRowStubs(path string, findings []Finding) error {
	var b strings.Builder
	b.WriteString("# Proposed Matrix Rows\n\n")
	b.WriteString("Stubs emitted from `make ux-explore`. Fill in `Expected` and paste into the active phase's `ux-flow-matrix.md`.\n\n")
	if len(findings) == 0 {
		b.WriteString("_No new rows._\n")
		return os.WriteFile(path, []byte(b.String()), 0o644)
	}
	for _, f := range findings {
		fmt.Fprintf(&b, "## %s — %s\n\n", f.MatrixID, f.Kind)
		fmt.Fprintf(&b, "- Fixture: `%s`\n", f.Fixture)
		fmt.Fprintf(&b, "- Preconditions: `%s/%s`", f.State.Screen, f.State.PreconditionClass)
		if f.State.BlockReason != "" {
			fmt.Fprintf(&b, " (block reason: %s)", f.State.BlockReason)
		}
		b.WriteString("\n")
		fmt.Fprintf(&b, "- Actual: %s\n", f.Recommendation)
		b.WriteString("- Expected: _(human-filled)_\n")
		fmt.Fprintf(&b, "- Invariants: %s\n\n", suggestedInvariants(f.Kind))
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func suggestedInvariants(kind FindingKind) string {
	switch kind {
	case FindingDeadEnd:
		return "I-13, I-17"
	case FindingSilentNoop:
		return "I-01, I-02"
	case FindingOrphan:
		return "I-17"
	case FindingErrorCycle:
		return "I-13, I-17"
	case FindingUnadvertisedKey:
		return "I-02"
	case FindingAdvertisedUnreachable:
		return "I-01"
	case FindingHiddenCursor:
		return "I-16"
	}
	return "I-01"
}

type coverageDoc struct {
	Reached []coverageCell `json:"reached"`
	Gaps    []string       `json:"gaps"`
}

type coverageCell struct {
	Screen            string `json:"screen"`
	PreconditionClass string `json:"precondition_class"`
	Hits              int    `json:"hits"`
}

func writeCoverageJSON(path string, c Coverage) error {
	doc := coverageDoc{Gaps: c.Gaps()}
	for _, cell := range c.SortedCells() {
		doc.Reached = append(doc.Reached, coverageCell{
			Screen:            cell.Screen,
			PreconditionClass: cell.PreconditionClass,
			Hits:              c.Reached[cell],
		})
	}
	return writeJSON(path, doc)
}

func writeGraphDOT(path string, traces []*Trace) error {
	var b strings.Builder
	b.WriteString("digraph ux_explore {\n")
	b.WriteString("  rankdir=LR;\n")
	seen := make(map[string]struct{})
	addNode := func(fp StateFingerprint) {
		id := nodeID(fp)
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		fmt.Fprintf(&b, "  %q [label=%q];\n", id, fp.Screen+"\\n"+fp.PreconditionClass)
	}
	for _, t := range traces {
		for _, v := range t.Visited {
			addNode(v.Fingerprint)
		}
		for _, e := range t.Edges {
			addNode(e.From)
			addNode(e.To)
		}
	}
	// Edges deduplicated for readability.
	edgeKey := func(e Edge) string { return nodeID(e.From) + "→" + nodeID(e.To) + ":" + e.Key }
	dedup := make(map[string]struct{})
	type emit struct{ from, to, key string }
	var emits []emit
	for _, t := range traces {
		for _, e := range t.Edges {
			k := edgeKey(e)
			if _, ok := dedup[k]; ok {
				continue
			}
			dedup[k] = struct{}{}
			emits = append(emits, emit{from: nodeID(e.From), to: nodeID(e.To), key: e.Key})
		}
	}
	sort.Slice(emits, func(i, j int) bool {
		if emits[i].from != emits[j].from {
			return emits[i].from < emits[j].from
		}
		if emits[i].to != emits[j].to {
			return emits[i].to < emits[j].to
		}
		return emits[i].key < emits[j].key
	})
	for _, e := range emits {
		fmt.Fprintf(&b, "  %q -> %q [label=%q];\n", e.from, e.to, e.key)
	}
	b.WriteString("}\n")
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func nodeID(fp StateFingerprint) string {
	return fp.Screen + "_" + fp.PreconditionClass
}
