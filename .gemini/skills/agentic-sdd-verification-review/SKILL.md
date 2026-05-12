---
name: agentic-sdd-verification-review
description: "Verify implementation and review code against the approved SDD spec, plan, tasks, contracts, and constitution."
allowed-tools: Read, Write, Edit, Glob, Grep, Bash, web_search_advanced_exa, web_fetch_exa
risk: safe
source: local
---

# Agentic SDD Verification Review

Use this skill after implementation and before merge or release.

## Goal

Prove the implementation matches the approved artifacts and is safe to review or merge.

## Required Reading

- `.specify/memory/constitution.md`
- `specs/<feature>/spec.md`
- `specs/<feature>/plan.md`
- `specs/<feature>/tasks.md`
- `specs/<feature>/test-plan.md`, if present
- `specs/<feature>/contracts/`, if present
- relevant ADRs
- git diff or changed files

## Verification Commands

Run applicable checks when available:

- format
- lint
- typecheck
- unit tests
- integration tests
- build
- security scan
- dependency review
- secret scan
- e2e or browser checks for UI work

Do not claim success for checks that were not run.

## Review Checklist

Assess:

- implementation matches spec and acceptance criteria
- implementation matches approved plan and tasks
- no unrelated behavior changed
- no forbidden files changed
- tests cover core behavior and regression risk
- external inputs are validated
- error states are represented clearly
- auth and authorization remain correct
- no secrets are committed
- dependencies are approved
- logging is useful and not noisy
- docs, contracts, ADRs, and release notes are updated where relevant
- no spec drift exists

## Review Output

Use this structure:

```text
Status: Approved | Needs changes | Blocked

Blocking findings:
- file:line - issue

Non-blocking suggestions:
- file:line - suggestion

Verification:
- command - passed/failed/not run, reason

Spec drift:
- none found
- or required artifact updates
```

## Stop Conditions

Do not approve when:

- verification fails for relevant behavior
- implementation diverges from spec or plan
- high-risk changes lack approval
- test coverage is missing for changed behavior
- contracts or docs are stale
- secrets or unsafe config are present

## Artifact Updates

If requested, update:

- `specs/<feature>/review.md`
- `specs/<feature>/test-plan.md`
- `specs/<feature>/drift-report.md`
- release notes
