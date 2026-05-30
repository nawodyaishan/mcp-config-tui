package uxexplore

import "sort"

// Cell is a (Screen, PreconditionClass) coverage cell.
type Cell struct {
	Screen            string `json:"screen"`
	PreconditionClass string `json:"precondition_class"`
}

// Coverage records which (Screen, PreconditionClass) cells were reached by the
// explorer across all fixtures.
type Coverage struct {
	Reached    map[Cell]int
	Exclusions []Exclusion
}

// ComputeCoverage walks every visited state across the given traces and
// tallies (Screen, PreconditionClass) cells.
func ComputeCoverage(traces []*Trace, exclusions []Exclusion) Coverage {
	c := Coverage{Reached: make(map[Cell]int), Exclusions: exclusions}
	for _, t := range traces {
		for _, v := range t.Visited {
			cell := Cell{Screen: v.Fingerprint.Screen, PreconditionClass: v.Fingerprint.PreconditionClass}
			c.Reached[cell]++
		}
	}
	return c
}

// ExpectedCells returns the cells that exploration should cover. Currently
// limited to PreconditionClass dimension; analyzer derives screens from
// observation rather than enumeration since screens are an internal concern.
func ExpectedCells() []string {
	return PreconditionClasses()
}

// Gaps returns precondition classes that no fixture reached and that are not
// covered by an exclusion entry.
func (c Coverage) Gaps() []string {
	reached := make(map[string]bool, len(c.Reached))
	for cell := range c.Reached {
		reached[cell.PreconditionClass] = true
	}
	excluded := make(map[string]bool, len(c.Exclusions))
	for _, e := range c.Exclusions {
		excluded[e.PreconditionClass] = true
	}
	var gaps []string
	for _, pc := range ExpectedCells() {
		if reached[pc] || excluded[pc] {
			continue
		}
		gaps = append(gaps, pc)
	}
	sort.Strings(gaps)
	return gaps
}

// SortedCells returns the reached cells sorted for stable artifact output.
func (c Coverage) SortedCells() []Cell {
	cells := make([]Cell, 0, len(c.Reached))
	for cell := range c.Reached {
		cells = append(cells, cell)
	}
	sort.Slice(cells, func(i, j int) bool {
		if cells[i].Screen != cells[j].Screen {
			return cells[i].Screen < cells[j].Screen
		}
		return cells[i].PreconditionClass < cells[j].PreconditionClass
	})
	return cells
}
