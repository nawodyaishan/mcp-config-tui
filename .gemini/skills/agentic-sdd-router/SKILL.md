---
name: agentic-sdd-router
description: "Route spec-driven development work to the correct phase skill. Use when the user asks for SDD, spec-driven development, Spec Kit style workflow, feature planning, implementation from specs, verification, or spec drift review."
allowed-tools: Read, Glob, Grep, Bash
risk: safe
source: local
---

# Agentic SDD Router

Use this skill first for spec-driven development work. It chooses the current phase, loads only the needed phase skill, and prevents accidental jumps from vague intent to code.

## Source Backing

This workflow is based on:

- Local guide: `/Users/nawodyaishan/Downloads/production-spec-driven-agentic-coding-guide.md`
- GitHub Spec Kit workflow: constitution, specify, clarify, plan, tasks, analyze/checklist, implement
- OpenAI Codex safety guidance: bounded sandboxing, explicit approvals for high-risk actions, and auditable tool activity

## Phase Map

| User intent | Use skill |
|---|---|
| Set up SDD in a repo | `agentic-sdd-bootstrap` |
| Write or improve product intent | `agentic-sdd-spec` |
| Resolve unclear requirements | `agentic-sdd-spec` |
| Create a technical plan | `agentic-sdd-plan` |
| Review plan before coding | `agentic-sdd-architecture-review` |
| Break plan into agent tasks | `agentic-sdd-tasks` |
| Implement an approved task | `agentic-sdd-implement` |
| Verify implementation or review diff | `agentic-sdd-verification-review` |
| Detect drift, release notes, retrospective | `agentic-sdd-drift-retro` |

## Required Order

For non-trivial production work, enforce this order:

```text
bootstrap, if needed
-> spec
-> clarify, if needed
-> plan
-> architecture review and approval
-> tasks
-> implementation
-> verification and code review
-> drift check, release notes, retrospective
```

## Routing Rules

- If there is no `spec.md` for the requested work, route to `agentic-sdd-spec`.
- If `spec.md` has open questions or ambiguous requirements, stay in `agentic-sdd-spec`.
- If there is no `plan.md`, route to `agentic-sdd-plan`.
- If `plan.md` exists but lacks approval, route to `agentic-sdd-architecture-review`.
- If there is no `tasks.md`, route to `agentic-sdd-tasks`.
- If implementation is requested and the plan is not approved, stop and ask for architecture approval.
- If implementation is requested for high-risk areas, require explicit approval before edits:
  - dependencies or lockfiles
  - database migrations
  - auth or authorization
  - payments
  - secrets or credentials
  - production config, infrastructure, deployment
- If the user asks for parallel agents or subagents, allow them only inside the implementation or review phases and only for bounded, disjoint work.

## Repository Discovery

Before choosing a phase, inspect:

- `.specify/memory/constitution.md`
- `docs/vision.md`
- `docs/adr/`
- `specs/`
- `AGENTS.md`, `CLAUDE.md`, `GEMINI.md`
- package/build/test files for verification commands

## Stop Conditions

Stop before implementation when:

- product behavior is unclear
- security or data sensitivity is unresolved
- the plan changes architecture without review
- migrations, auth, payments, secrets, dependencies, or production config are involved without approval
- verification commands cannot be identified

## Output

Return the selected phase, the skill to use, missing artifacts, and the immediate next action.
