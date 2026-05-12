import re

file_path = "/Users/nawodyaishan/Documents/DevOps/Articles/usync-platform-series/05-context7-provider-without-hardcoded-tui.md"

with open(file_path, "r") as f:
    content = f.read()

sdd_section = """
## Spec-Driven Agentic Workflow

As the platform scales, maintaining this clean separation of concerns becomes harder if contributors rely entirely on free-form AI code generation. To solve this, `usync` now uses a strict **Spec-Driven Development (SDD)** workflow.

Instead of prompting an AI assistant to "just build a new provider," contributors use local skills stored directly in the repository under `.gemini/skills/`:

1. **Draft the Spec**: Use the `agentic-sdd-spec` skill to draft a specification document detailing the provider's transport, auth method, and client adaptations.
2. **Review and Approve**: Ensure the spec captures the constraints before code is generated, using the `agentic-sdd-architecture-review` skill.
3. **Execution**: Trigger the `agentic-sdd-implement` skill, explicitly binding it to the approved spec to ensure generated code respects the provider contract and doesn't hardcode TUI changes.

This local-skill-driven approach ensures that AI contributions are as disciplined and reviewable as human ones.

## Contributor takeaway"""

content = content.replace("## Contributor takeaway", sdd_section)

with open(file_path, "w") as f:
    f.write(content)

print("Updated 05-context7-provider-without-hardcoded-tui.md")
