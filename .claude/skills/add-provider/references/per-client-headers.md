# Per-Client Headers Output Ground Truth

| Client | File kind | URL field | Headers field | Special |
|---|---|---|---|---|
| Cursor / Roo Code / Kiro | mcpServers JSON | `url` | `headers` | Roo also `"type":"streamable-http"` |
| OpenCode | named server `mcp` | `url` | `headers` | also `"type":"remote","enabled":true` |
| VS Code | named server `servers` | `url` | `headers` | also `"type":"http"` |
| Antigravity / Windsurf | mcpServers JSON | `serverUrl` | `headers` | — |
| Gemini CLI | mcpServers/bare | `httpUrl` | `headers` | + `Accept: application/json, text/event-stream` |
| Codex CLI | TOML | `url` | `http_headers` inline table | — |
| Zed | named server `context_servers` | `url` | `headers` | — |
| Claude Desktop | mcpServers JSON | n/a (stdio) | n/a | `npx -y @upstash/context7-mcp --api-key <key>` |
| Claude Code (CLI) | `claude mcp add` | n/a | `--header` flag | `--header "CONTEXT7_API_KEY: <key>"` |