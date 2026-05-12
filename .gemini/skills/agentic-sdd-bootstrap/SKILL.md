---
name: agentic-sdd-bootstrap
description: "Initialize or upgrade a repository for production spec-driven development with constitution, templates, review checklists, and tool memory pointers."
allowed-tools: Read, Write, Edit, Glob, Grep, Bash
risk: medium
source: local
---

# Agentic SDD Bootstrap

Use this skill when a repository needs the SDD governance layer before feature work starts.

## Goal

Create the durable repository artifacts that agents and humans will use as the source of truth.

## Required Structure

Prefer this layout unless the repository already has an equivalent convention:

```text
.specify/
  memory/
    constitution.md
  templates/
docs/
  vision.md
  adr/
  review-checklists/
  release-notes/
  retrospectives/
specs/
AGENTS.md
```

Add `CLAUDE.md` and `GEMINI.md` when the user wants those tools to share the same SDD rules.

## Bootstrap Steps

1. Inspect the repo for existing governance files, README, CI, test commands, build commands, and source layout.
2. Create missing directories only.
3. Create or update `.specify/memory/constitution.md`.
4. Create templates for:
   - feature spec
   - implementation plan
   - tasks
   - architecture review
   - code review
   - bug fix spec
   - refactor spec
   - migration spec
   - test plan
   - release checklist
   - spec drift report
   - retrospective
5. Create `docs/review-checklists/architecture-review.md`.
6. Create `docs/review-checklists/code-review.md`.
7. Create or update `AGENTS.md` with source-of-truth pointers and safety boundaries.
8. If requested, create or update `CLAUDE.md` and `GEMINI.md` with short pointers to the same artifacts.

## Constitution Minimum

The constitution must define:

- product purpose and users
- production quality bar
- allowed technologies
- disallowed patterns
- dependency policy
- testing mandates
- security mandates
- documentation policy
- architecture review gate
- code review and release gate
- amendment process

## Safety Boundaries

Default to allowing agents to read, search, draft docs, plan, edit approved feature code, and run tests. Require human approval for:

- dependency installs or lockfile changes
- database migrations
- auth and authorization changes
- payment logic
- secrets and credentials
- production configuration and infrastructure
- destructive file operations
- deployment and release

## Output

Report created and updated files, existing files preserved, missing information, and the recommended first pilot feature.
