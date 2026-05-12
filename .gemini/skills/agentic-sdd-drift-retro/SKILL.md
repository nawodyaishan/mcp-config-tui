---
name: agentic-sdd-drift-retro
description: "Detect spec drift, prepare release notes, and write retrospectives after SDD implementation or review."
allowed-tools: Read, Write, Edit, Glob, Grep, Bash
risk: safe
source: local
---

# Agentic SDD Drift Retro

Use this skill after implementation, before release, or when a diff may have changed behavior without matching artifacts.

## Goal

Keep code, specs, plans, contracts, ADRs, tests, and release notes aligned.

## Required Reading

- git diff or changed files
- `specs/<feature>/spec.md`
- `specs/<feature>/plan.md`
- `specs/<feature>/tasks.md`
- `specs/<feature>/test-plan.md`, if present
- `specs/<feature>/contracts/`, if present
- relevant ADRs
- `docs/release-notes/`
- `.specify/memory/constitution.md`

## Drift Checks

Flag drift when:

- user-visible behavior changed without spec update
- API request or response changed without contract update
- database schema changed without data model and migration spec
- auth or permissions changed without security notes or ADR
- dependency changed without approval note
- observability changed without plan or test note
- deployment config changed without ops note and approval
- tests no longer match acceptance criteria

## Drift Report

Create or update `specs/<feature>/drift-report.md` with:

- drift found or no drift found
- evidence
- affected files
- missing or stale artifacts
- risk level
- required artifact updates
- prevention rule, hook, or CI check

## Retrospective

Create or update `docs/retrospectives/<date>-<feature>.md` with:

- what changed
- what the agent did well
- what the agent got wrong
- what the human caught
- what rule, template, hook, or CI check should be added
- follow-up actions

## Release Notes

For user-facing changes, create or update `docs/release-notes/<release-or-date>.md` with:

- summary
- user-visible changes
- migration or compatibility notes
- known limitations
- rollback note, if relevant

## Output

Return drift status, files updated, release-note status, retrospective path, and recommended process improvements.
