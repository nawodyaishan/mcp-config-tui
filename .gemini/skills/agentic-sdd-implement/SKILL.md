---
name: agentic-sdd-implement
description: "Implement one approved SDD task inside strict file, safety, and verification boundaries."
allowed-tools: Read, Write, Edit, Glob, Grep, Bash
risk: medium
source: local
---

# Agentic SDD Implement

Use this skill when implementing one approved task from `specs/<feature>/tasks.md`.

## Goal

Implement exactly one bounded task or a small approved group of independent tasks. Keep the change scoped and verifiable.

## Required Reading

- `.specify/memory/constitution.md`
- `specs/<feature>/spec.md`
- `specs/<feature>/plan.md`
- `specs/<feature>/tasks.md`
- `specs/<feature>/test-plan.md`, if present
- `specs/<feature>/review.md`, if present
- relevant ADRs
- relevant existing code and tests

## Before Editing

Confirm:

- the task is marked approved or safe to start
- allowed and forbidden files are clear
- verification command is defined
- high-risk areas are not involved, or approval is explicit

## High-Risk Approval Gate

Ask before touching:

- package manager files or dependencies
- database schema or migrations
- auth, authorization, sessions, permissions
- payments or billing
- secrets, credentials, environment files
- production config, infrastructure, deployment
- destructive deletes or broad rewrites

## Implementation Rules

- Stay within allowed files unless you ask first.
- Follow existing project patterns.
- Prefer simple direct code over new abstractions.
- Do not weaken tests to make code pass.
- Update contracts, docs, and specs only when the task requires it.
- Record any deviation from the plan.
- Do not use subagents unless the user explicitly asked for delegated or parallel agent work.
- If subagents are used, assign disjoint file ownership and run review before integration.

## Verification

Run the task's verification command. Also run the narrowest relevant tests, typecheck, lint, or build available for the touched area.

If verification fails:

- diagnose the failure
- fix if within scope
- report honestly if blocked or unrelated

## Completion Report

Return:

- task id
- summary of changes
- files changed
- tests and commands run
- results
- deviations from the plan
- follow-up risks or required review

## Stop Conditions

Stop when:

- implementation requires changing forbidden files
- a high-risk area needs approval
- the task is not independently understandable
- tests cannot be run and no acceptable manual verification exists
- implementation reveals the plan is wrong
