package uxexplore

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/tui"
)

// UnmappedKeys is the constant set of keys the probe fires to verify that
// hidden keys do not advance the product flow (I-02).
var UnmappedKeys = []string{"x", "5", "f1", "z"}

// maxDrainSteps bounds the number of follow-up tea.Cmd executions the probe
// will run between key presses. Async work in the fakes resolves in 1-2 steps;
// the bound stops any accidental loop.
const maxDrainSteps = 16

// Probe walks the reachable state space of a DashboardModel from a starting
// model, running per-visit invariants, firing advertised keys, firing unmapped
// keys, and verifying double-press stability for in-flight states.
type Probe struct {
	invariants []Invariant
	visited    map[string]struct{}
}

// NewProbe constructs a probe with the default invariant set.
func NewProbe() *Probe {
	return &Probe{
		invariants: Invariants(),
		visited:    make(map[string]struct{}),
	}
}

// Visit explores the state space starting at model m. Newly reached states,
// edges, and invariant violations are appended to trace.
func (p *Probe) Visit(m tui.DashboardModel, trace *Trace) {
	p.visit(m, trace, 0)
}

const maxDepth = 32

func (p *Probe) visit(m tui.DashboardModel, trace *Trace, depth int) {
	if depth > maxDepth {
		return
	}
	fp := Fingerprint(m)
	view := m.View()
	digest := viewDigest(view)
	key := fp.Screen + "|" + fp.PreconditionClass + "|" + fp.BlockReason + "|" + fp.InFlight + "|" + digest
	if _, seen := p.visited[key]; seen {
		return
	}
	p.visited[key] = struct{}{}

	trace.Visited = append(trace.Visited, VisitedState{
		Fingerprint: fp,
		ViewDigest:  digest,
		ModelSnap:   ModelSnap{Screen: fp.Screen, View: view},
	})

	for _, inv := range p.invariants {
		if err := inv.Check(m); err != nil {
			trace.Errors = append(trace.Errors, ExplorerError{
				Kind:    "invariant:" + inv.ID(),
				State:   fp,
				Message: err.Error(),
			})
		}
	}

	if IsTerminal(fp) {
		return
	}

	advertised := ParseActionBar(view)
	sort.Strings(advertised)

	// IMPORTANT ordering: unmapped-key + double-press passes run FIRST so they
	// observe the model in the same state we just fingerprinted. DashboardModel
	// contains map fields (e.g. selectedTargets) that drive() mutates via the
	// shared backing array when an advertised key writes into them. Running
	// advertised-key recursion before these checks would pollute the model and
	// produce false positives.

	// Unmapped-key pass: I-02. Fingerprint must not change.
	for _, k := range UnmappedKeys {
		next, ok := drive(m, k)
		if !ok {
			continue
		}
		if Fingerprint(next) != fp {
			trace.Errors = append(trace.Errors, ExplorerError{
				Kind:    "unmapped-key-advances",
				Key:     k,
				State:   fp,
				Message: fmt.Sprintf("unmapped key %q changed fingerprint", k),
			})
		}
	}

	// Double-press pass: I-03. If the state is in-flight, the first advertised
	// non-global key fired twice must leave fingerprint stable.
	if fp.InFlight != "" {
		primary := primaryAdvertisedKey(advertised)
		if primary != "" {
			once, ok1 := drive(m, primary)
			twice, ok2 := drive(once, primary)
			if ok1 && ok2 {
				if Fingerprint(twice) != Fingerprint(once) {
					trace.Errors = append(trace.Errors, ExplorerError{
						Kind:    "double-press-unstable",
						Key:     primary,
						State:   fp,
						Message: fmt.Sprintf("double-press of %q during %s advanced state", primary, fp.InFlight),
					})
				}
			}
		}
	}

	// Advertised-key pass: descend through each advertised key.
	for _, k := range advertised {
		if shouldSkipAdvertised(k) {
			continue
		}
		next, ok := drive(m, k)
		if !ok {
			continue
		}
		nextFP := Fingerprint(next)
		nextDigest := viewDigest(next.View())
		trace.Edges = append(trace.Edges, Edge{
			From: fp, Key: k, To: nextFP,
			FromViewDigest: digest, ToViewDigest: nextDigest,
		})
		p.visit(next, trace, depth+1)
	}
}

// shouldSkipAdvertised filters keys that the probe should not recurse into:
// quit/help would terminate or open an overlay outside the dashboard state
// machine; esc is exercised by the I-14 check rather than recursion.
func shouldSkipAdvertised(k string) bool {
	switch strings.ToLower(k) {
	case "q", "ctrl+c", "?":
		return true
	}
	return false
}

func primaryAdvertisedKey(keys []string) string {
	for _, k := range keys {
		switch strings.ToLower(k) {
		case "q", "ctrl+c", "?", "esc", "up", "down":
			continue
		}
		return k
	}
	return ""
}

// drive applies a single key to the model and drains any follow-up commands so
// the model reaches a settled state before fingerprinting. Returns (next,false)
// when the key results in tea.Quit, signaling the probe should not descend.
func drive(m tui.DashboardModel, key string) (tui.DashboardModel, bool) {
	msg := buildKeyMsg(key)
	nextModel, cmd := m.Update(msg)
	next, ok := nextModel.(tui.DashboardModel)
	if !ok {
		return m, false
	}
	settled, quit := drainCmd(next, cmd)
	if quit {
		return settled, false
	}
	return settled, true
}

func drainCmd(m tui.DashboardModel, cmd tea.Cmd) (tui.DashboardModel, bool) {
	for steps := 0; steps < maxDrainSteps && cmd != nil; steps++ {
		msg := cmd()
		if msg == nil {
			return m, false
		}
		if _, isQuit := msg.(tea.QuitMsg); isQuit {
			return m, true
		}
		nextModel, nextCmd := m.Update(msg)
		next, ok := nextModel.(tui.DashboardModel)
		if !ok {
			return m, false
		}
		m = next
		cmd = nextCmd
	}
	return m, false
}

// buildKeyMsg constructs a tea.KeyMsg matching what handleKey would receive
// from the bubbletea runtime for a given key token.
func buildKeyMsg(key string) tea.KeyMsg {
	switch strings.ToLower(key) {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "shift+tab":
		return tea.KeyMsg{Type: tea.KeyShiftTab}
	case "backspace":
		return tea.KeyMsg{Type: tea.KeyBackspace}
	case "space", " ":
		return tea.KeyMsg{Type: tea.KeySpace}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "left":
		return tea.KeyMsg{Type: tea.KeyLeft}
	case "right":
		return tea.KeyMsg{Type: tea.KeyRight}
	case "ctrl+c":
		return tea.KeyMsg{Type: tea.KeyCtrlC}
	case "f1":
		return tea.KeyMsg{Type: tea.KeyF1}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
}
