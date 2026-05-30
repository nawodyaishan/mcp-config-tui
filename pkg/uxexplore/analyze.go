package uxexplore

import (
	"fmt"
	"sort"
	"strings"
)

// Analyze inspects a slice of traces and returns the union of findings produced
// by every detector. Findings are deduplicated by MatrixID and sorted for
// stable artifact output.
func Analyze(traces []*Trace) []Finding {
	var findings []Finding
	for _, t := range traces {
		findings = append(findings, detectDeadEnds(t)...)
		findings = append(findings, detectSilentNoops(t)...)
		findings = append(findings, detectOrphans(t)...)
		findings = append(findings, detectErrorCycles(t)...)
		findings = append(findings, detectInvariantViolations(t)...)
	}
	return dedupe(findings)
}

func dedupe(in []Finding) []Finding {
	seen := make(map[string]struct{}, len(in))
	out := make([]Finding, 0, len(in))
	for _, f := range in {
		if _, dup := seen[f.MatrixID]; dup {
			continue
		}
		seen[f.MatrixID] = struct{}{}
		out = append(out, f)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].MatrixID < out[j].MatrixID })
	return out
}

// detectDeadEnds emits a finding when a non-terminal state has at least one
// outbound edge AND every outbound edge target shares the same (Screen,
// PreconditionClass, BlockReason) with the source — meaning every advertised
// key fails to progress the user.
func detectDeadEnds(t *Trace) []Finding {
	var out []Finding
	outbound := outboundIndex(t)
	for _, v := range t.Visited {
		if IsTerminal(v.Fingerprint) {
			continue
		}
		edges := outbound[v.Fingerprint]
		if len(edges) == 0 {
			continue // orphan handled separately
		}
		stuck := true
		for _, e := range edges {
			if e.To.Screen != v.Fingerprint.Screen ||
				e.To.PreconditionClass != v.Fingerprint.PreconditionClass ||
				e.To.BlockReason != v.Fingerprint.BlockReason {
				stuck = false
				break
			}
		}
		if stuck {
			out = append(out, NewFinding(
				FindingDeadEnd,
				t.Fixture.Name,
				[]string{v.Fingerprint.Screen, v.Fingerprint.PreconditionClass},
				v.Fingerprint,
				fmt.Sprintf("every advertised key on %s/%s lands back in the same state", v.Fingerprint.Screen, v.Fingerprint.PreconditionClass),
			))
		}
	}
	return out
}

// detectSilentNoops finds edges where pressing an advertised key produces no
// observable state change. Navigation, idempotent-refresh, and space-toggle
// keys are excluded because their no-op behavior on degenerate state (single
// item, empty list, already-scanning) is intended product behavior, not a UX
// bug. Real silent-noops show up on primary action keys (enter, y, n, p, k,
// v, w, x).
func detectSilentNoops(t *Trace) []Finding {
	var out []Finding
	for _, e := range t.Edges {
		if e.From != e.To {
			continue
		}
		// A genuine silent-noop changes neither fingerprint nor view. If the
		// rendered view differs, the key produced *some* observable change
		// (cursor shift, toggle, inline edit) even though the high-level
		// fingerprint stayed the same.
		if e.FromViewDigest != "" && e.FromViewDigest != e.ToViewDigest {
			continue
		}
		if isQuietKey(e.Key) {
			continue
		}
		out = append(out, NewFinding(
			FindingSilentNoop,
			t.Fixture.Name,
			[]string{e.From.Screen, e.Key},
			e.From,
			fmt.Sprintf("advertised key %q does not change state on %s", e.Key, e.From.Screen),
		))
	}
	return out
}

func isQuietKey(k string) bool {
	switch strings.ToLower(k) {
	case "up", "down", "j", "left", "right",
		"tab", "shift+tab",
		"space", " ",
		"r": // rescan/refresh is idempotent by design (Doctor screen)
		return true
	}
	return false
}

// detectOrphans finds non-terminal states with zero outbound edges.
func detectOrphans(t *Trace) []Finding {
	var out []Finding
	outbound := outboundIndex(t)
	for _, v := range t.Visited {
		if IsTerminal(v.Fingerprint) {
			continue
		}
		if len(outbound[v.Fingerprint]) == 0 {
			out = append(out, NewFinding(
				FindingOrphan,
				t.Fixture.Name,
				[]string{v.Fingerprint.Screen},
				v.Fingerprint,
				fmt.Sprintf("state %s/%s has no outbound edges", v.Fingerprint.Screen, v.Fingerprint.PreconditionClass),
			))
		}
	}
	return out
}

// detectErrorCycles finds strongly-connected components of size >= 2 (or
// self-loops) in which every state has HasError == true and shares a
// BlockReason. Tarjan's SCC over the edge graph.
func detectErrorCycles(t *Trace) []Finding {
	nodes := nodeList(t)
	if len(nodes) == 0 {
		return nil
	}
	index := make(map[StateFingerprint]int, len(nodes))
	for i, n := range nodes {
		index[n] = i
	}
	adj := make([][]int, len(nodes))
	for _, e := range t.Edges {
		f, okF := index[e.From]
		to, okT := index[e.To]
		if !okF || !okT {
			continue
		}
		adj[f] = append(adj[f], to)
	}
	sccs := tarjanSCC(adj)
	var out []Finding
	for _, comp := range sccs {
		if len(comp) < 2 {
			// self-loop only — handled by silent-noop
			continue
		}
		errOnly := true
		reason := ""
		for _, idx := range comp {
			fp := nodes[idx]
			if !fp.HasError {
				errOnly = false
				break
			}
			if reason == "" {
				reason = fp.BlockReason
			} else if fp.BlockReason != reason {
				errOnly = false
				break
			}
		}
		if !errOnly {
			continue
		}
		fp := nodes[comp[0]]
		out = append(out, NewFinding(
			FindingErrorCycle,
			t.Fixture.Name,
			[]string{fp.Screen, reason},
			fp,
			fmt.Sprintf("error cycle (%d states) with shared block reason %q", len(comp), reason),
		))
	}
	return out
}

// detectInvariantViolations promotes probe-recorded invariant errors into
// findings so downstream tooling (matrix stubs, reports) treats them
// uniformly.
func detectInvariantViolations(t *Trace) []Finding {
	var out []Finding
	for _, e := range t.Errors {
		kind := FindingInvariantViolation
		switch e.Kind {
		case "unmapped-key-advances":
			kind = FindingUnadvertisedKey
		case "double-press-unstable":
			kind = FindingInvariantViolation
		}
		out = append(out, NewFinding(
			kind,
			t.Fixture.Name,
			[]string{e.State.Screen, e.Kind, e.Key},
			e.State,
			e.Message,
		))
	}
	return out
}

func outboundIndex(t *Trace) map[StateFingerprint][]Edge {
	out := make(map[StateFingerprint][]Edge, len(t.Visited))
	for _, e := range t.Edges {
		out[e.From] = append(out[e.From], e)
	}
	return out
}

func nodeList(t *Trace) []StateFingerprint {
	seen := make(map[StateFingerprint]struct{}, len(t.Visited))
	var out []StateFingerprint
	for _, v := range t.Visited {
		if _, ok := seen[v.Fingerprint]; ok {
			continue
		}
		seen[v.Fingerprint] = struct{}{}
		out = append(out, v.Fingerprint)
	}
	for _, e := range t.Edges {
		if _, ok := seen[e.From]; !ok {
			seen[e.From] = struct{}{}
			out = append(out, e.From)
		}
		if _, ok := seen[e.To]; !ok {
			seen[e.To] = struct{}{}
			out = append(out, e.To)
		}
	}
	return out
}

// tarjanSCC returns the strongly-connected components of a directed graph
// given as an adjacency list of node indices.
func tarjanSCC(adj [][]int) [][]int {
	n := len(adj)
	index := 0
	stack := make([]int, 0, n)
	onStack := make([]bool, n)
	indices := make([]int, n)
	lowlink := make([]int, n)
	for i := range indices {
		indices[i] = -1
	}
	var sccs [][]int
	var strongConnect func(v int)
	strongConnect = func(v int) {
		indices[v] = index
		lowlink[v] = index
		index++
		stack = append(stack, v)
		onStack[v] = true
		for _, w := range adj[v] {
			if indices[w] == -1 {
				strongConnect(w)
				if lowlink[w] < lowlink[v] {
					lowlink[v] = lowlink[w]
				}
			} else if onStack[w] {
				if indices[w] < lowlink[v] {
					lowlink[v] = indices[w]
				}
			}
		}
		if lowlink[v] == indices[v] {
			var comp []int
			for {
				w := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				onStack[w] = false
				comp = append(comp, w)
				if w == v {
					break
				}
			}
			sccs = append(sccs, comp)
		}
	}
	for v := range n {
		if indices[v] == -1 {
			strongConnect(v)
		}
	}
	return sccs
}
