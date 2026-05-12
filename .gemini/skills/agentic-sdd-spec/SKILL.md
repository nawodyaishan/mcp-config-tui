---
name: agentic-sdd-spec
description: "Create or refine SDD feature, bug-fix, refactor, or migration specifications with clarification handling before technical planning."
allowed-tools: Read, Write, Edit, Glob, Grep, Bash, web_search_advanced_exa, web_fetch_exa
risk: safe
source: local
---

# Agentic SDD Spec

Use this skill when creating or improving the product-facing specification for a change.

## Goal

Capture what and why before deciding how. Do not design implementation in this phase.

## Inputs

Read:

- `docs/vision.md`, if present
- `.specify/memory/constitution.md`, if present
- existing `specs/*/spec.md`
- relevant README/product docs
- user request

Use Exa only when the spec depends on current external facts, official product behavior, regulations, API behavior, or tool versions.

## Spec Types

Choose one:

- feature spec
- bug fix spec
- refactor spec
- migration spec
- release/documentation spec

## Feature Spec Sections

Create or update `specs/<nnn-short-name>/spec.md` with:

- problem statement
- goals
- non-goals
- users or actors
- user journeys
- functional requirements
- acceptance criteria
- success criteria
- edge cases
- data sensitivity and compliance notes
- API or integration expectations, without implementation design
- assumptions
- open questions
- human approval status

## Clarification Rules

- Ask at most 3 clarification questions at a time.
- Ask only when the answer changes scope, security/privacy, user experience, data model, compatibility, or acceptance criteria.
- Use reasonable defaults for low-risk missing details and document them as assumptions.
- Do not proceed to planning while unresolved open questions affect implementation.

## Quality Checklist

Create or update `specs/<feature>/checklists/requirements.md` with:

- no implementation details in spec
- goals and non-goals clear
- requirements testable and unambiguous
- acceptance criteria complete
- success criteria measurable and technology-agnostic
- edge cases identified
- data sensitivity noted
- open questions resolved or explicitly deferred
- spec ready for planning

## Stop Conditions

Stop before planning if:

- user-visible behavior is ambiguous
- actors or permissions are unclear
- data sensitivity is unresolved
- success criteria are not verifiable
- the user has not approved or accepted critical clarifications

## Output

Return the spec path, checklist path, unresolved questions, assumptions, and readiness for `agentic-sdd-plan`.
