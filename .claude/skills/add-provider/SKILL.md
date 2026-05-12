---
name: add-provider
description: |
  Use when adding a new MCP server provider to usync. Triggers: "add a new
  provider", "add provider X", "scaffold provider", "support <NAME> MCP",
  "register an MCP server". Walks through official doc lookup, scaffolding,
  registration, header dispatch, QA scenarios, and documentation updates.
when_to_use: |
  Trigger on: 'add a provider', 'support <name> MCP server', 'scaffold provider'.
  Do NOT trigger for general Go refactors or test-only changes.
allowed-tools: Read, Grep, Glob, Bash, Edit, Write
---

# Add Provider Procedure

We follow a strict Spec-Driven Development (SDD) workflow. When requested to add a new MCP provider to `usync`, strictly follow these steps:

## 0. SDD Pre-requisite
Ensure an approved specification document exists (e.g., `docs/specs/add-<name>-provider.md`). If it doesn't exist or hasn't been approved, use the `agentic-sdd-router` or `agentic-sdd-spec` skill to create and refine the spec first. Do not proceed to implementation without a spec.

## 1. STOP and gather
Read the following files to understand the system context:
- `pkg/provider/types.go`
- `pkg/provider/exa.go`
- `pkg/provider/context7.go`
- `docs/contributors/adding-a-provider.md`
- `docs/specs/add-context7-provider.md`

## 2. Look up official docs
Use your search tools (or the Context7 MCP if available) to query the target server's configuration schema. Identify if it requires an API key, env vars, or specific arguments.

## 3. Decide capabilities
Use the decision tree from `adding-a-provider.md`:
- Is it `stdio` or a remote transport?
- Does it use URL query auth or Header auth?
- Is it a single-key or multi-key setup?

## 4. Scaffold helpers
If necessary, create a package at `pkg/<id>/` for validation logic (e.g., `keys.go`, `keys_test.go`).
Reference: `references/code-templates.md` (Template 1).

## 5. Implement provider
Create `pkg/provider/<id>.go` implementing the `MCPProvider` interface.
Reference: `references/code-templates.md` (Template 2).

## 6. Register
Register your new provider in `DefaultRegistry()` inside `pkg/provider/registry.go`.

## 7. Per-client adaptation
If the provider uses headers, verify if any client-specific adaptations are needed in `pkg/client/adapter.go` (e.g., `HeadersFor`).

## 8. QA scenarios
Write end-to-end scenarios in `pkg/app/qa_scenarios_test.go` confirming the config gets emitted accurately.
Reference: `references/code-templates.md` (Template 3).

## 9. Docs
Add a row to the Provider Matrix in `README.md`.
Update `docs/contributors/adding-a-provider.md` if the architectural approach expands.

Review the `references/checklist.md` before concluding.