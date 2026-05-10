# Dogfooding with Exa and Context7

As a contributor, you can use `usync` to configure your local AI tools to help build `usync`.

## Getting Started

1. **Build `usync`**
   ```bash
   make build
   ```

2. **Get API keys**
   - Get an Exa API key from [exa.ai](https://exa.ai).
   - Get a Context7 key from [context7.com/dashboard](https://context7.com/dashboard): open the API Keys card, create a key, and copy it immediately. Current keys usually look like `ctx7sk-...`; legacy `ctx7sk_...` keys are also accepted.

3. **Preview Exa from the CLI**
   ```bash
   ./bin/usync sync --keys-file ./exa_keys.txt --dry-run
   ```

   Confirm the target paths and redacted credentials look correct before applying.

4. **Configure providers through the TUI**
   ```bash
   ./bin/usync
   ```

   Follow the prompts to add Exa and Context7 to your preferred clients, such as Claude Code, Cursor, Windsurf, or Zed.

5. **Restart clients**
   Restart your AI clients so they load the new MCP servers.

## Example Prompts
- "Use Context7 to look up the Bubbletea documentation for creating a custom tea.Cmd."
- "Use Exa to search for recent discussions on Go 1.23 iterator patterns."

## Contributor Loop

When adding new providers, start with [Adding an MCP Provider](adding-a-provider.md), write or review the provider spec, then validate with:
```bash
make fmt
make test
make lint
```

If you use Claude Code in this repo, the `.claude/skills/add-provider` skill can follow an approved provider spec, for example:
```text
/add-provider using docs/specs/add-<name>-provider.md as the strict implementation contract
```
