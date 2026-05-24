---
name: ux-bug-hunt
description: |
  PROJECT-SPECIFIC to usync (mcp-config-tui). Use when iterating on TUI UI/UX
  bugs via the Docker-first bug-hunt harness. Triggers: "UX bug hunt",
  "iterate on UI bugs", "fix UX issues", "expand the matrix", "ux-fake-prod",
  "matrix scenario", "TUI regression", "DM-P<n>".
  Drives the 4-stage loop (expand → run → triage → fix+lock) and the lens
  framework for catching UX defect classes beyond functional bugs.
when_to_use: |
  Trigger on: any work that adds/removes/changes a TUI screen, key handler,
  action bar, or user-visible flow in pkg/tui/. Trigger when investigating
  DM-P* issues in artifacts/ux-fake-prod/issues.json. Trigger when adding new
  UX scenarios or extending the harness.
  Do NOT trigger for: non-TUI code changes, pure refactors with no
  user-visible diff, doc-only edits, or any repository other than
  mcp-config-tui.
allowed-tools: Read, Grep, Glob, Bash, Edit, Write
---

# UX Bug-Hunt Iteration (usync / mcp-config-tui)

> **Scope:** This skill is personalized to the `mcp-config-tui` repository.
> Every file path below is **relative to the repo root**
> (`/Users/nawodyaishan/Documents/GitHub/mcp-config-tui`). It will not work in
> any other project — the harness, scenarios, and IDs are project-owned.

The Docker-first UX harness (`make ux-fake-prod`) is a **bug-funnel**, not a
one-shot. Treat each iteration as a disciplined 4-stage loop and never skip
stage 3 (triage).

---

## 0. Required reading (read these before doing anything)

### Specs & protocol (authoritative intent)
- `docs/specs/doctor-mode-phase12/ux-bug-hunt-protocol.md` — the protocol; Docker run is the authoritative collection mechanism
- `docs/specs/doctor-mode-phase12/ux-flow-matrix.md` — scenario matrix; every row is a contract
- `docs/specs/doctor-mode-phase12/spec.md` — Phase 12 product intent
- `docs/specs/doctor-mode-phase12/plan.md` — implementation plan
- `docs/specs/doctor-mode-phase12/tasks.md` — task breakdown
- `docs/specs/doctor-mode-phase12/user-flow-audit.md` — audited journeys

### Harness source (the runtime)
- `tests/ux-fake-prod/Dockerfile` — fake-prod container build
- `tests/ux-fake-prod/docker-run.sh` — entrypoint invoked by `make ux-fake-prod`
- `tests/ux-fake-prod/run-flow.sh` — in-container scenario driver
- `.dockerignore` — keep this small; image hygiene affects determinism

### Test runners (the matrix)
- `pkg/tui/dashboard_fake_prod_matrix_test.go` — Docker matrix runner
- `pkg/tui/dashboard_flow_matrix_test.go` — local shortcut matrix runner
- `pkg/tui/dashboard_flow_test.go` — base teatest flows (read for patterns)
- `pkg/tui/dashboard_golden_test.go` — width/render goldens

### TUI surface under test
- `pkg/tui/dashboard.go` — `DashboardModel`, screens, key handlers
- `pkg/tui/dashboard_view.go` — all `render*` functions and action bars
- `pkg/tui/dashboard_test.go` — unit tests (regression baseline)

### Artifacts (the output of every run)
- `artifacts/ux-fake-prod/` — **do not edit by hand**; produced by Docker
  - `matrix.json` — canonical scenario × result table
  - `issues.json` — canonical open issues (`DM-P<n>` IDs)
  - `fake-prod-matrix.txt` — human-readable matrix
  - `teatest-matrix.txt` — local-test matrix
  - doctor JSON, plan JSON, fake home-after snapshot, TUI transcripts

### Build entrypoints
- `Makefile` targets: `ux-matrix`, `ux-fake-prod` (see Makefile lines around 65)

---

## 1. The 4-stage loop

### Stage 1 — Expand → discover
Add 1–3 new scenarios per iteration to `docs/specs/doctor-mode-phase12/ux-flow-matrix.md`. Each row needs:
preconditions, key sequence, expected end state, defect criteria.

Draw scenarios from these axes:
- **State permutations**: empty config, partial config, corrupted JSON, symlinked path, read-only file, permission-denied, conflict client, missing runtime, expired plan, concurrent on-disk edit.
- **Input permutations**: every key at every screen (`pkg/tui/dashboard.go` handler switch), unmapped keys, rapid double-presses, Esc from every state.
- **Boundary conditions**: 0/1/many clients, 0 providers, window resize mid-flow (60/80/120/200 cols).
- **User journeys**: "I made a mistake, can I back out?", "I left and came back, does state persist?".

### Stage 2 — Run → collect
```
make ux-fake-prod
```
Reads `tests/ux-fake-prod/docker-run.sh`; writes deterministic artifacts to `artifacts/ux-fake-prod/`. New issues get IDs `DM-P<n>` with: screen, severity, scenario, observed vs expected, artifact pointer.

### Stage 3 — Triage before coding ← never skip
Classify every issue in `artifacts/ux-fake-prod/issues.json` before any fix:
- **Bug** — behavior wrong → fix in `pkg/tui/dashboard.go` (handlers)
- **Missing affordance** — works but invisible → fix in `pkg/tui/dashboard_view.go` (action bars, help)
- **Wrong mental model** — user expectation mismatch → fix flow + spec
- **Spec gap** — matrix right, spec silent → update `docs/specs/doctor-mode-phase12/spec.md` first

Without triage you whack-a-mole: "fix" DM-P10 by advertising `r`, then DM-P34 appears because `r` does nothing on the doctor screen.

### Stage 4 — Fix → re-run → lock in
For each fix, in a **single commit**:
1. Update the matrix scenario in `docs/specs/doctor-mode-phase12/ux-flow-matrix.md` to assert the *correct* behavior (not just "no issue").
2. `make ux-fake-prod` — that scenario passes; nothing else regresses.
3. Commit fix + matrix update + golden snapshot (`pkg/tui/testdata/`) together.

When `issues.json` is empty, flip `make ux-fake-prod` to CI-required. From then on a regression *creates* an issue, CI blocks, you fix and re-lock.

---

## 2. UX defect lenses (extend beyond functional bugs)

The functional-flow lens (in `pkg/tui/dashboard_fake_prod_matrix_test.go`) catches behavior bugs. Add specialized lenses that read the same Docker artifacts in `artifacts/ux-fake-prod/` — no new harness needed.

| Defect class | Lens (proposed path) | What it checks |
|---|---|---|
| Confusing copy | `tools/uxlens/lint-copy.go` | Action-bar verbs (`pkg/tui/dashboard_view.go`) against approved list; jargon; unmapped keys |
| Inconsistent layout | `pkg/tui/dashboard_golden_test.go` at widths 60/80/120/200 | Overflow, truncation, alignment |
| Information overload | line-count budget per screen (≤24 at 80×24) | Forces summarization |
| Silent state changes | transcript diff before/after each key | Every key must produce visible feedback |
| Dead-end errors | grep `artifacts/ux-fake-prod/teatest-matrix.txt` for `Error:` without adjacent `Press X to…` | Every error offers recovery |
| Inaccessibility | strip ANSI from transcripts; no color-only signals (NO_COLOR=1) | Screen-reader friendliness |
| Performance | wall-clock keystroke→render in transcript | >100ms is a UX bug |
| State memory | scenario: do X → restart → does TUI remember? | Catches lost work |

Each lens is a small Go program (~50 LOC) consuming `artifacts/ux-fake-prod/*`.

---

## 3. Project rules

1. **No PR merges into `pkg/tui/` without a matrix entry** in `docs/specs/doctor-mode-phase12/ux-flow-matrix.md`. New screen/key/flow → new row first, then code.
2. **Severity drives velocity.** Blockers same-day, majors same-week, minors batched, cosmetic backlog.
3. **The matrix is the spec.** When `ux-flow-matrix.md` and `spec.md` disagree, matrix wins — it's executable.
4. **One issue, one commit.** Don't bundle; isolated reverts when triage was wrong.
5. **Periodic hostile-user sessions.** Once a sprint, manually run `go run ./cmd/usync` and try to break the TUI for 30 min; every surprise becomes a matrix row even if not fixed immediately.

---

## 4. Anti-patterns

- Editing files under `artifacts/ux-fake-prod/` by hand (they're generated).
- Adding a code fix in `pkg/tui/` without a matrix scenario in `docs/specs/doctor-mode-phase12/ux-flow-matrix.md` that locks it in.
- Bundling multiple unrelated fixes per commit.
- Skipping triage and patching the first observable symptom.
- Letting `issues.json` grow without a velocity target.
- Treating `make ux-fake-prod` failures as flakes instead of bugs.
- Using local matrix tests (`dashboard_flow_matrix_test.go`) as the authority — they're a shortcut; the Dockerized run in `dashboard_fake_prod_matrix_test.go` is canonical.

---

## 5. Verification

A skill invocation is successful when:
- New rows exist in `docs/specs/doctor-mode-phase12/ux-flow-matrix.md` with full preconditions/expected behavior.
- `make ux-fake-prod` runs clean and updates `artifacts/ux-fake-prod/`.
- Every closed `DM-P<n>` issue has: code fix in `pkg/tui/` + matrix row that asserts correct behavior + green CI in one commit.
- `artifacts/ux-fake-prod/issues.json` strictly decreases or stays at 0 over time.
