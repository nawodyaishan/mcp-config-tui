---
name: agentic-sdd-tasks
description: "Break an approved SDD plan into small, bounded, independently verifiable implementation tasks."
allowed-tools: Read, Write, Edit, Glob, Grep, Bash
risk: safe
source: local
---

# Agentic SDD Tasks

Use this skill only after the architecture plan is approved or the user explicitly accepts the risk of proceeding.

## Goal

Create `specs/<feature>/tasks.md` that an implementation agent can execute without reinterpreting the spec.

## Required Reading

- `.specify/memory/constitution.md`
- `specs/<feature>/spec.md`
- `specs/<feature>/plan.md`
- `specs/<feature>/review.md`
- `specs/<feature>/test-plan.md`, if present
- relevant ADRs

## Task Format

Each task must include:

- task id
- objective
- source artifacts
- allowed files or directories
- forbidden files or directories
- acceptance criteria
- verification command or observable result
- dependencies
- risk level
- approval needed, if any
- status

## Task Design Rules

- Keep each task small enough for one focused implementation pass.
- Prefer vertical slices by user story when possible.
- Put tests before implementation when behavior is risky or TDD is requested.
- Mark parallel-safe tasks only when write sets are disjoint.
- Do not create tasks that require unapproved dependencies, migrations, auth/payment edits, secrets, production config, or deployment.
- Include documentation and contract updates as explicit tasks when behavior changes.

## Recommended Sections

`specs/<feature>/tasks.md` should include:

- track summary
- prerequisites
- task list
- dependency order
- parallel-safe groups
- verification matrix
- blocked or approval-required work

## Stop Conditions

Stop before writing implementation-ready tasks when:

- architecture review is not approved
- tasks would require guessing file ownership
- verification commands are unknown
- a task touches high-risk areas without approval
- dependencies between tasks are unclear

## Output

Return task file path, number of tasks, blocked items, parallel-safe groups, and the first safe implementation task.
