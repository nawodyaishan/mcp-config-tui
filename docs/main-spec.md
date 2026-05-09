# Go TUI Spec: Exa MCP Config Manager for macOS

## Summary

Build a Go TUI app that updates Exa MCP configuration across local AI apps on macOS. The tool lets the user load multiple Exa API keys, choose target apps, assign keys across apps to spread usage, enable the maximum supported Exa MCP tool set, create backups, apply updates, and verify each config.

Target apps:

- Claude Desktop
- Claude Code
- Gemini CLI
- Antigravity
- Codex CLI

Use Bubble Tea for the TUI and Go standard libraries for config updates.

## Key Behavior

- Detect existing config files:
  - `~/Library/Application Support/Claude/claude_desktop_config.json`
  - `~/.claude.json`
  - `~/.gemini/settings.json`
  - `~/.gemini/mcp_config.json`
  - `~/.gemini/antigravity/mcp_config.json`
  - `~/.codex/config.toml`
- Accept keys from:
  - pasted multiline input
  - `--keys-file /path/to/exa_keys.txt`
  - manual TUI entry
- Parse UUID-style Exa keys from formats like:
  - raw key per line
  - `key1 = "..."`
  - `key2 = "..."`
- Never print full keys. Display only prefix/suffix, for example `6ea4...7887`.

## Exa MCP URL

Use this full tool set by default:

```text
web_search_exa,
web_fetch_exa,
web_search_advanced_exa,
get_code_context_exa,
company_research_exa,
crawling_exa,
people_search_exa,
linkedin_search_exa,
deep_researcher_start,
deep_researcher_check,
deep_search_exa
```

Generate URLs as:

```text
https://mcp.exa.ai/mcp?exaApiKey=<KEY>&tools=<comma-separated-tools>
```

## TUI Flow

1. Welcome screen
   - Show detected apps and config paths.
   - Show whether each config exists.

2. Key loading screen
   - Let user paste keys or choose `--keys-file`.
   - Validate at least one UUID-style key.
   - Show parsed key count and redacted key labels.

3. App selection screen
   - Default: all detected apps selected.
   - Allow toggling each app.
   - Missing config files can still be selected if the app supports creating a config.

4. Key distribution screen
   - Default rotation:
     - key 1: Claude Desktop, Gemini CLI, Codex
     - key 2: Claude Code, Antigravity
     - if more keys exist, round-robin across apps
   - Allow manual reassignment per app.

5. Preview screen
   - Show planned changes without full keys.
   - Show backup file paths that will be created.
   - Require explicit confirmation before writing.

6. Apply screen
   - Backup each touched config with suffix:
     - `.bak-exa-YYYYMMDD-HHMMSS`
   - Update selected configs.
   - Preserve unrelated MCP servers and existing non-MCP settings.

7. Verification screen
   - Parse updated configs and confirm:
     - Exa entry exists
     - API key exists
     - expected tool count is present
     - URL query is valid
   - Run optional CLI checks when available:
     - `codex mcp get exa`
     - `claude mcp get exa`
   - Show restart reminder for affected apps.

## Config Update Rules

- Claude Desktop:
  - Update `mcpServers.exa` in `claude_desktop_config.json`.
  - Use:
    ```json
    {
      "type": "sse",
      "url": "<exa-url>"
    }
    ```

- Gemini CLI:
  - Update both `~/.gemini/settings.json` and `~/.gemini/mcp_config.json` when selected.
  - Use:
    ```json
    {
      "type": "sse",
      "url": "<exa-url>"
    }
    ```

- Antigravity:
  - Update `~/.gemini/antigravity/mcp_config.json`.
  - Use:
    ```json
    {
      "serverUrl": "<exa-url>"
    }
    ```

- Codex:
  - Update `~/.codex/config.toml`.
  - Ensure this block exists exactly once:
    ```toml
    [mcp_servers.exa]
    url = "<exa-url>"
    ```

- Claude Code:
  - Prefer CLI update when `claude` is available:
    ```bash
    claude mcp remove exa -s user
    claude mcp add --transport sse -s user exa "<exa-url>"
    ```
  - If CLI is unavailable, show a clear warning and skip direct mutation of `~/.claude.json` unless a future explicit fallback is implemented.

## Implementation Structure

Use this project layout:

```text
exa-mcp-manager/
  go.mod
  cmd/exa-mcp-manager/main.go
  pkg/app/app.go
  pkg/config/paths.go
  pkg/config/json_update.go
  pkg/config/toml_update.go
  pkg/exa/keys.go
  pkg/exa/tools.go
  pkg/exa/url.go
  pkg/tui/model.go
  pkg/tui/views.go
  pkg/verify/verify.go
```

Core packages:

- `pkg/exa`
  - parse keys
  - redact keys
  - build Exa MCP URLs
  - define default tool set
- `pkg/config`
  - detect app config paths
  - read/write JSON safely
  - update Codex TOML block
  - create timestamped backups
- `pkg/tui`
  - Bubble Tea state machine
  - app selection, key entry, preview, apply, verification views
- `pkg/verify`
  - inspect updated config state
  - optionally run app CLI checks

## CLI Flags

Support non-interactive and TUI modes:

```bash
exa-mcp-manager
exa-mcp-manager --keys-file ~/Downloads/exa_keys.txt
exa-mcp-manager --keys "key1,key2"
exa-mcp-manager --dry-run --keys-file ~/Downloads/exa_keys.txt
exa-mcp-manager --apply --keys-file ~/Downloads/exa_keys.txt
```

Rules:

- Default with no `--apply`: launch TUI.
- `--dry-run`: print redacted plan only.
- `--apply`: update configs without TUI after validation.

## Test Plan

- Unit tests:
  - parse raw and labelled key files
  - reject no-key input
  - build Exa URL with all tools
  - redact keys correctly
  - update JSON while preserving unrelated fields
  - update Codex TOML with exactly one `[mcp_servers.exa]` block
  - create backup path format correctly

- Fixture tests:
  - Claude Desktop JSON with existing `context7`
  - Gemini settings with existing UI/security fields
  - Antigravity malformed old URL using double `?`
  - Codex config with existing Exa block
  - missing config files

- Manual acceptance:
  - Run `--dry-run` with `~/Downloads/exa_keys.txt`.
  - Run TUI and apply to test fixture directory first.
  - Run real apply.
  - Verify `codex mcp get exa`.
  - Verify `claude mcp get exa` if Claude Code is installed.

## Assumptions

- macOS is the target platform.
- Exa keys are UUID-style strings.
- Multiple keys spread usage across apps, not within a single app, to avoid duplicate MCP tool-name collisions.
- The default Exa tool set intentionally includes current and deprecated-compatible tools for maximum coverage.
- The tool must not display full API keys in the UI, logs, dry-run output, or verification output.
