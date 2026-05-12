---
name: agentic-sdd-architecture-review
description: "Review an SDD implementation plan before code generation and decide whether it is approved, needs changes, or is rejected."
allowed-tools: Read, Write, Edit, Glob, Grep, Bash, web_search_advanced_exa, web_fetch_exa
risk: safe
source: local
---

# Agentic SDD Architecture Review

Use this skill as the mandatory gate between technical planning and task generation or code.

## Goal

Review the plan as a production architecture gate. Do not implement code.

## Required Packet

Review:

- `specs/<feature>/spec.md`
- `specs/<feature>/clarify.md`, if present
- `specs/<feature>/plan.md`
- `specs/<feature>/data-model.md`, if present
- `specs/<feature>/contracts/`, if present
- `specs/<feature>/test-plan.md`, if present
- `.specify/memory/constitution.md`
- relevant ADRs
- dependency-change notes
- rollback notes

## Checklist

Assess:

- scope is clear
- open questions are resolved or explicitly deferred
- plan follows existing architecture
- new abstractions are justified
- existing libraries/framework features are used before custom layers
- dependencies are justified and approved
- data ownership is clear
- migrations and rollback are documented
- sensitive data and retention rules are considered
- API compatibility and error responses are defined
- auth and authorization boundaries are explicit
- input validation and injection risks are considered
- observability and failure modes are defined
- deployment and rollback impact are understood
- tests cover the core risks

## Decision

Return one of:

- `Approved`: task breakdown may begin.
- `Needs changes`: list blocking changes before approval.
- `Rejected`: the plan is structurally unsafe or inconsistent with goals.

## Writing Review Artifacts

Create or update `specs/<feature>/review.md` with:

- reviewed artifacts
- findings by severity
- approval status
- required changes
- optional improvements
- reviewer and date placeholders

If the plan is approved, update the approval status in `plan.md` only if the user explicitly asked you to maintain artifacts.

## Stop Conditions

Do not approve when:

- a high-risk area lacks explicit human approval
- rollback is missing for migrations or operational risk
- dependencies are unjustified
- tests are absent for behavior, security, or data boundaries
- plan contradicts constitution or ADRs
- code generation would require guessing

## Output

Lead with approval status, then blocking findings, non-blocking suggestions, and next phase.
