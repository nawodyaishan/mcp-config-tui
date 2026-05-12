---
name: agentic-sdd-research-spec
description: "Create or improve spec-driven development artifacts using local source docs plus current Exa research. Use for SDD planning, gap analysis, documentation upgrades, architecture specs, task breakdowns, and docs-only preparation before implementation."
allowed-tools: Read, Write, Edit, Glob, Grep, Bash, web_search_advanced_exa, web_fetch_exa
risk: safe
source: local
---

# Agentic SDD Research Spec

Use this skill when a user asks for a spec-driven development document, a gap analysis against an SDD workflow, or an upgrade of a rough project draft into a high-quality planning artifact. The goal is a better specification, not implementation.

## Operating Rules

- Treat the spec as the source of truth. Do not start coding unless the user explicitly asks for implementation.
- Read local source material first, then use Exa advanced search to validate gaps, current terminology, and official behavior.
- Prefer official documentation, primary project docs, standards, and maintained repositories over generic posts.
- Separate stable product intent from technical choices. Keep "what and why" in the spec, and "how" in the plan.
- Add explicit verification criteria. A task is not complete unless the validation command or observable result is defined.
- Flag unresolved ambiguity as an open question or clarification marker; do not silently invent risky requirements.
- Keep outputs actionable for agents: exact file paths, commands, phase gates, acceptance criteria, and stop conditions.
- Preserve the user's goal. Improve accuracy, structure, and completeness without expanding into a different product.

## Workflow

1. **Source Intake**
   - Read the user's existing draft, SDD guide, README, and repository structure.
   - Identify the target artifact type: spec, plan, tasks, README, verification spec, or gap report.
   - Note constraints such as "docs only", "do not develop", "zero dependency", or "no server".

2. **Exa Research**
   - Use `web_search_advanced_exa` with domain filters when possible.
   - Query for current official docs and recent practice discussions.
   - Capture only the facts that change the document: workflow phases, verification gaps, API behavior, version-specific caveats, and security boundaries.

3. **Gap Analysis**
   - Compare the draft against the SDD guide and research.
   - Look for missing clarification phase, implementation boundaries, verification spec, success criteria, test strategy, drift prevention, dependency policy, and risk gates.
   - For technical topics, verify claims against official docs before presenting them as facts.

4. **Document Upgrade**
   - Produce a polished Markdown artifact with:
     - Purpose and scope
     - Non-goals
     - Audience
     - Source-backed facts
     - Proposed repository structure
     - Architecture or content model
     - Executable task breakdown
     - Verification checklist
     - Human review gates
     - Open questions
   - Keep tasks implementation-ready, but do not create source files unless asked.

5. **Final Pass**
   - Check that the document does not contradict the user's constraints.
   - Confirm there are no misleading absolutes, especially around scheduling, timing, security, or performance.
   - Save to the requested repository location and update index files such as `README.md` when requested.

## Quality Bar

- Every major claim must be either derived from the local SDD source, supported by Exa research, or clearly marked as an assumption.
- Every task must have a clear outcome and a verification method.
- Every autonomous-agent boundary must say what is safe, what needs approval, and what is forbidden.
- Documentation should enable a future implementation agent to proceed without reinterpreting the original conversation.

