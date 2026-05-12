import re

file_path = "/Users/nawodyaishan/Documents/DevOps/Articles/usync-platform-series/06-platform-engineering-lessons-from-usync.md"

with open(file_path, "r") as f:
    content = f.read()

sdd_section = """## Contribute

The best way to learn from the project is to trace one provider end to end, then make a small improvement using our local Spec-Driven Development (SDD) workflow:

1. Read `docs/contributors/adding-a-provider.md`.
2. Draft a specification using the `.gemini/skills/agentic-sdd-spec` skill.
3. Review the architecture using the `.gemini/skills/agentic-sdd-architecture-review` skill.
4. Implement the changes by explicitly passing the spec to `.gemini/skills/agentic-sdd-implement`.
5. Run `make fmt`, `make test`, and `make gitignore-check`.
6. Open a focused PR that explains the provider shape, client impact, and safety behavior.

This local-skill-driven contribution path ensures AI-generated code is reviewable, predictable, and aligned with platform safety habits."""

content = re.sub(r"## Contribute\n\nThe best way to learn.*multiple AI clients\.", sdd_section, content, flags=re.DOTALL)

with open(file_path, "w") as f:
    f.write(content)

print("Updated 06-platform-engineering-lessons-from-usync.md")
