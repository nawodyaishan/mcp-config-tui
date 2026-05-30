package uxexplore

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// HandlerKeyMap maps a handleKey* function suffix (e.g. "Doctor") to the set
// of literal keys it switches on.
type HandlerKeyMap map[string]map[string]struct{}

// ParseHandlerKeys parses one or more Go source files and returns the set of
// literal keys the handleKey* functions switch on. It mirrors the runtime
// dispatch in pkg/tui/dashboard.go — see screen list there for the canonical
// set of handlers. Pass the directory's representative dashboard file; the
// scanner also reads sibling files in the same directory so handlers split
// into separate files (e.g. credential_entry.go) are covered.
func ParseHandlerKeys(path string) (HandlerKeyMap, error) {
	fset := token.NewFileSet()
	paths := []string{path}
	if matches, err := filepathSiblings(path); err == nil {
		paths = matches
	}
	out := make(HandlerKeyMap)
	for _, p := range paths {
		file, err := parser.ParseFile(fset, p, nil, parser.SkipObjectResolution)
		if err != nil {
			return nil, err
		}
		collectHandlerKeys(file, out)
	}
	return out, nil
}

func collectHandlerKeys(file *ast.File, out HandlerKeyMap) {
	ast.Inspect(file, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Recv == nil {
			return true
		}
		name := fn.Name.Name
		if !strings.HasPrefix(name, "handleKey") || name == "handleKey" {
			return true
		}
		screen := strings.TrimPrefix(name, "handleKey")
		keys := out[screen]
		if keys == nil {
			keys = make(map[string]struct{})
			out[screen] = keys
		}
		ast.Inspect(fn.Body, func(inner ast.Node) bool {
			cc, ok := inner.(*ast.CaseClause)
			if !ok {
				return true
			}
			for _, expr := range cc.List {
				lit, ok := expr.(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					continue
				}
				val, err := strconv.Unquote(lit.Value)
				if err != nil {
					continue
				}
				keys[val] = struct{}{}
			}
			return true
		})
		return true
	})
}

func filepathSiblings(path string) ([]string, error) {
	dir := filepath.Dir(path)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		out = append(out, filepath.Join(dir, name))
	}
	return out, nil
}

// AuditKeymap compares the keys advertised by the action bar (observed during
// exploration) against the keys the handleKey* switches handle. It emits two
// finding kinds:
//
//   - FindingUnadvertisedKey: handler case has no advertisement (hidden key)
//   - FindingAdvertisedUnreachable: advertised key has no handler case
//
// Skips global keys (q, ?, ctrl+c, esc) since they short-circuit in handleKey
// before dispatch.
func AuditKeymap(handlers HandlerKeyMap, traces []*Trace) []Finding {
	advertised := make(map[string]map[string]struct{}) // screen → keys
	for _, t := range traces {
		for _, v := range t.Visited {
			screen := v.Fingerprint.Screen
			if advertised[screen] == nil {
				advertised[screen] = make(map[string]struct{})
			}
			for _, k := range ParseActionBar(v.ModelSnap.View) {
				advertised[screen][k] = struct{}{}
			}
		}
	}
	var findings []Finding
	screens := sortedKeys(handlers)
	for _, screen := range screens {
		handled := handlers[screen]
		adv := advertised[screen]
		for _, k := range sortedSetKeys(handled) {
			if isGlobalKey(k) {
				continue
			}
			if _, ok := adv[k]; !ok {
				findings = append(findings, NewFinding(
					FindingUnadvertisedKey,
					"keymap-audit",
					[]string{screen, k},
					StateFingerprint{Screen: screen},
					fmt.Sprintf("handler %s accepts key %q but no action bar advertises it", screen, k),
				))
			}
		}
		for _, k := range sortedSetKeys(adv) {
			if isGlobalKey(k) {
				continue
			}
			if _, ok := handled[k]; !ok {
				findings = append(findings, NewFinding(
					FindingAdvertisedUnreachable,
					"keymap-audit",
					[]string{screen, k},
					StateFingerprint{Screen: screen},
					fmt.Sprintf("action bar advertises key %q on %s but no handler case matches", k, screen),
				))
			}
		}
	}
	return findings
}

func isGlobalKey(k string) bool {
	switch strings.ToLower(k) {
	case "q", "?", "ctrl+c", "esc", "up", "down", " ", "tab", "shift+tab", "backspace", "enter":
		// enter, esc, up/down are dispatched but commonly overridden per
		// screen — treat as global to reduce noise; per-screen overrides
		// still emit findings via their explicit cases.
		return true
	}
	return false
}

func sortedKeys(m HandlerKeyMap) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortedSetKeys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
