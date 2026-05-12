---
name: agentic-sdd-plan
description: "Produce a technical implementation plan from an approved SDD spec without editing product code."
allowed-tools: Read, Write, Edit, Glob, Grep, Bash, web_search_advanced_exa, web_fetch_exa
risk: safe
source: local
---

# Agentic SDD Plan

Use this skill after the spec is clear enough to design the technical approach.

## Goal

Create `specs/<feature>/plan.md` and related planning artifacts. Do not implement code in this phase.

## Required Reading

- `.specify/memory/constitution.md`
- `docs/vision.md`
- `specs/<feature>/spec.md`
- `specs/<feature>/clarify.md`, if present
- relevant ADRs in `docs/adr/`
- relevant source files and tests
- existing API contracts, schemas, migrations, and deployment config when affected

## Research Rules

Use Exa advanced search when:

- a framework, SDK, or platform behavior may have changed
- security guidance is current-version sensitive
- a public API contract is involved
- the plan depends on official docs

Prefer official docs, maintained repositories, standards, and primary sources. Record only facts that affect the plan.

## Plan Sections

Create or update `specs/<feature>/plan.md` with:

- summary
- inputs reviewed
- assumptions
- architecture approach
- affected modules
- API and contract changes
- data model changes
- dependency changes with alternatives and approval status
- security impact
- authorization boundaries
- observability impact
- testing strategy
- failure modes
- rollback and recovery
- risks and mitigations
- human architecture approval status

Create additional files when relevant:

- `data-model.md`
- `contracts/openapi.yaml`
- `contracts/events.md`
- `test-plan.md`
- ADR draft in `docs/adr/`

## Architecture Rules

- Prefer existing project patterns and framework features.
- Avoid speculative abstractions.
- Justify every new dependency.
- Do not silently change auth, payments, data retention, or production operations.
- Include rollback for migrations and operationally risky changes.
- Include verification commands or observable results.

## Stop Conditions

Stop and request human input when:

- dependency addition is proposed
- database migration is required
- auth, authorization, payments, secrets, or production config are touched
- rollback is not realistic
- tests or verification cannot be defined
- the plan conflicts with constitution or ADRs

## Output

Return paths written, key decisions, risks, approval requirements, and readiness for `agentic-sdd-architecture-review`.
