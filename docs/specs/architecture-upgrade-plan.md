# usync Architecture Upgrade — Implementation Spec
**Last updated:** 2026-05-10  
**Status:** Approved for implementation  
**Audience:** Engineers picking up tasks. This doc is the single source of truth — no cross-referencing required.

---

## How to use this document

Each task has a unique ID (T01–T16), its phase, what it blocks, exact file paths with line numbers, Go code to write, and a concrete acceptance checklist. Work Phase 3A tasks in order (T01→T09) because they have sequential dependencies. Phase 3B tasks (T10–T15) can begin after T09 is merged. Phase 3C (T16) depends on T11.

Run `make test && make lint` after every task before opening a PR.

---

## Current architecture pain points (reference)

These are the specific problems every task traces back to. Read once, refer back as needed.

| # | Problem | Location | Symptom |
|---|---|---|---|
| P1 | `configForTarget()` hard-codes a single Exa×ClaudeDesktop bridge | `pkg/app/app.go:176` | Adding any new provider requires editing this function |
| P2 | `pkg/app/app.go` imports `pkg/exa` directly | `app.go:18` | Manager knows Exa domain: key parsing, URL redaction, tool count |
| P3 | `VerifyProviderFile()` returns hard failure for all non-Exa providers | `pkg/verify/verify.go:51` | GitHub/Brave/etc. configs cannot be verified after write |
| P4 | `FormatPlan` / `FormatApplyResult` are Exa-branded | `app.go:315,342` | CLI output says "Exa MCP update plan" — wrong for multi-provider |
| P5 | `TransportStreamableHTTP` constant missing | `pkg/provider/types.go` | 2025 MCP spec uses `"streamable-http"`; clients like Roo Code need it as first-class value, not an `extra` map hack |
| P6 | No capability matrix | scattered in `configForTarget()` + `prepareFileOperation()` | No single place to answer "does this client support stdio?" |
| P7 | `syncToContext()` in TUI hard-codes Exa UUID key parsing | `pkg/tui/setup_form.go:143` | Second multi-value provider cannot be added without editing TUI |

---

## Dependency graph

```
T01 (transport constant)
 └─ T02 (pkg/redact)
     └─ T03 (pkg/client capabilities)
         └─ T04 (pkg/client adapter)
             └─ T05 (client tests)
                 └─ T06 (app.go refactor)  ← core, longest task
                     ├─ T07 (verify generalize)
                     ├─ T08 (format strings + test updates)
                     └─ T09 (Makefile + CI coverage gate)
                         └─ [Phase 3B begins]
                             ├─ T10 (GitHubProvider)
                             │   └─ T11 (register in registry)
                             │       ├─ T12 (capability matrix review)
                             │       ├─ T13 (GitHub QA tests)
                             │       └─ T14 (verify for stdio)
                             └─ T15 (TUI preview stdio display)
                                 └─ [Phase 3C]
                                     └─ T16 (decouple TUI from pkg/exa)
```

---

## Phase 3A — Structural Cleanup

No new providers. No behavior changes. All existing tests must continue to pass after each task.

---

### T01 — Add `TransportStreamableHTTP` and `PackageRuntime` to provider types

**Phase:** 3A  
**Blocks:** T03, T04, T10  
**Files:** `pkg/provider/types.go`

#### What to change

`pkg/provider/types.go` currently has three transport constants. Add a fourth and a new struct:

```go
// pkg/provider/types.go — full file replacement

package provider

type TransportType string

const (
    TransportStdio          TransportType = "stdio"
    TransportStreamableHTTP TransportType = "streamable-http"
    TransportSSE            TransportType = "sse"
    TransportHTTP           TransportType = "http" // legacy; kept for VS Code "type":"http" compat
)

// PackageRuntime describes the packaging type of a stdio server.
// Used to communicate install context (npm, pypi, oci) to UI layers.
// Nil means the provider is a remote HTTP server.
type PackageRuntime struct {
    Type string // "npm" | "pypi" | "oci" | "mcpb"
}

// MCPConfig is a provider-agnostic description of one MCP server connection.
type MCPConfig struct {
    Type    TransportType
    URL     string            // HTTP / SSE / StreamableHTTP
    Command string            // stdio: executable name, e.g. "npx"
    Args    []string          // stdio: arguments after command
    Env     map[string]string // stdio: env vars injected into the subprocess
    Runtime *PackageRuntime   // non-nil for packaged stdio servers; nil for remote
}

// CredentialValidator validates one credential string value.
type CredentialValidator func(string) error

// CredentialSpec describes one credential field required by a provider.
type CredentialSpec struct {
    Key         string
    Label       string
    Description string
    Secret      bool
    MultiValue  bool
    Validator   CredentialValidator
}

// CredentialProfile is a collected set of credentials for one provider instance.
type CredentialProfile struct {
    ProviderID string
    Values     map[string]string
    Label      string // redacted display string shown in UI
}

// MCPProvider is the contract every MCP server plugin must implement.
type MCPProvider interface {
    ID() string
    Name() string
    Description() string
    RequiredCredentials() []CredentialSpec
    GenerateConfig(credentials map[string]string) (MCPConfig, error)
}
```

#### Tests to add

In `pkg/provider/registry_test.go`, add one assertion to the existing `TestDefaultRegistry` test (or create one if it's minimal):

```go
func TestTransportConstants(t *testing.T) {
    if string(provider.TransportStreamableHTTP) != "streamable-http" {
        t.Fatalf("TransportStreamableHTTP must equal \"streamable-http\", got %q",
            provider.TransportStreamableHTTP)
    }
    if string(provider.TransportHTTP) != "http" {
        t.Fatalf("TransportHTTP must equal \"http\"")
    }
}
```

#### Acceptance

- `make test` passes
- `make lint` passes
- `grep -r "TransportStreamableHTTP" pkg/` finds the new constant

---

### T02 — Create `pkg/redact`

**Phase:** 3A  
**Depends on:** T01  
**Blocks:** T06  
**New files:** `pkg/redact/redact.go`, `pkg/redact/redact_test.go`

#### Why

`pkg/app/app.go` currently calls `exa.RedactText()` and `exa.RedactKey()` in its logger, rollback handler, and CLI runner. This creates a domain dependency (`pkg/app` → `pkg/exa`). A generic UUID redactor belongs in a shared package.

`pkg/exa/keys.go` keeps its own `RedactText` and `RedactKey` — those functions also redact Exa-specific URLs (`mcp.exa.ai`). They are not replaced. `pkg/redact` is a separate, simpler package that only handles UUID-shaped strings.

#### Implementation

```go
// pkg/redact/redact.go
package redact

import "regexp"

var uuidRE = regexp.MustCompile(
    `(?i)\b[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\b`,
)

// Text replaces every UUID-shaped substring in s with a truncated token.
func Text(s string) string {
    return uuidRE.ReplaceAllStringFunc(s, Key)
}

// Key returns the first 4 and last 4 characters of key separated by "...".
// Keys shorter than 9 characters are returned unchanged.
func Key(key string) string {
    if len(key) <= 8 {
        return key
    }
    return key[:4] + "..." + key[len(key)-4:]
}

// Attrs redacts string values in a slog-style key-value variadic slice.
// Non-string values are passed through unchanged.
func Attrs(attrs []any) []any {
    out := make([]any, 0, len(attrs))
    for _, a := range attrs {
        if s, ok := a.(string); ok {
            out = append(out, Text(s))
            continue
        }
        out = append(out, a)
    }
    return out
}
```

```go
// pkg/redact/redact_test.go
package redact_test

import (
    "testing"
    "github.com/nawodyaishan/universal-mcp-sync/pkg/redact"
)

func TestKey(t *testing.T) {
    tests := []struct{ in, want string }{
        {"11111111-1111-1111-1111-111111111111", "1111...1111"},
        {"abcd", "abcd"},                          // too short — unchanged
        {"12345678", "12345678"},                   // exactly 8 — unchanged
        {"123456789", "1234...6789"},               // 9 chars — truncated
    }
    for _, tt := range tests {
        if got := redact.Key(tt.in); got != tt.want {
            t.Errorf("Key(%q) = %q, want %q", tt.in, got, tt.want)
        }
    }
}

func TestText(t *testing.T) {
    uuid := "11111111-1111-1111-1111-111111111111"
    input := "error: key " + uuid + " rejected"
    got := redact.Text(input)
    if got == input {
        t.Fatal("Text should redact UUID-shaped substrings")
    }
    if contains(got, uuid) {
        t.Fatalf("Text output still contains full UUID: %s", got)
    }
    // non-UUID text preserved
    if !contains(got, "error: key") || !contains(got, "rejected") {
        t.Errorf("Text should preserve non-UUID parts: %s", got)
    }
}

func TestTextNoUUID(t *testing.T) {
    in := "plain text without secrets"
    if got := redact.Text(in); got != in {
        t.Errorf("Text should not modify strings without UUIDs")
    }
}

func TestAttrs(t *testing.T) {
    uuid := "11111111-1111-1111-1111-111111111111"
    attrs := []any{"error", uuid, "count", 42}
    got := redact.Attrs(attrs)
    if len(got) != 4 {
        t.Fatalf("Attrs should preserve length")
    }
    if s, ok := got[1].(string); !ok || s == uuid {
        t.Errorf("Attrs should redact UUID string values")
    }
    if n, ok := got[3].(int); !ok || n != 42 {
        t.Errorf("Attrs should pass non-string values through unchanged")
    }
}

func contains(s, sub string) bool {
    return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
        func() bool {
            for i := 0; i+len(sub) <= len(s); i++ {
                if s[i:i+len(sub)] == sub { return true }
            }
            return false
        }())
}
```

#### Acceptance

- `go test ./pkg/redact/...` passes
- `pkg/redact` has no imports from `pkg/exa` or `pkg/app`

---

### T03 — Create `pkg/client/capabilities.go`

**Phase:** 3A  
**Depends on:** T01  
**Blocks:** T04, T05  
**New file:** `pkg/client/capabilities.go`

#### What this replaces

The current `configForTarget()` in `app.go:176` and the implicit transport knowledge buried in `prepareFileOperation()` (the `switch op.AppID` at lines 477–512 that picks `urlFieldName` and `extra` per client).

`pkg/client` declares what each AI client *can do*. The adapter (T04) uses this to transform configs. The JSON writer (`pkg/config`) is NOT changed — it still accepts the `urlFieldName` and `extra` parameters, but those are now driven by the adapter, not hard-coded per provider.

#### Implementation

```go
// pkg/client/capabilities.go
package client

import (
    "github.com/nawodyaishan/universal-mcp-sync/pkg/config"
    "github.com/nawodyaishan/universal-mcp-sync/pkg/provider"
)

// TransportSupport declares which MCP transport types a client handles natively.
type TransportSupport struct {
    Stdio          bool
    StreamableHTTP bool
    SSE            bool
    HTTP           bool // legacy; VS Code uses "type":"http"
}

// BridgeConfig describes a stdio wrapper that proxies a remote transport.
// {url} in Args is replaced with the actual server URL at adapt time.
type BridgeConfig struct {
    Command string
    Args    []string
}

// Capability is the full capability profile of one AI client.
type Capability struct {
    Supports TransportSupport
    // Bridge maps a transport type the client cannot handle natively to a
    // stdio bridge that wraps it. If no bridge is declared and the client
    // does not support the transport, CanHandle returns false.
    Bridge map[provider.TransportType]*BridgeConfig
}

// Matrix is the authoritative source of what each AI client supports.
// When adding a new client: add its AppID here with accurate capabilities.
// When a client gains a new transport: update its TransportSupport.
var Matrix = map[config.AppID]Capability{
    config.AppClaudeDesktop: {
        // Claude Desktop only speaks stdio natively.
        // Remote HTTP/StreamableHTTP servers are bridged via mcp-remote.
        Supports: TransportSupport{Stdio: true},
        Bridge: map[provider.TransportType]*BridgeConfig{
            provider.TransportStreamableHTTP: {
                Command: "npx",
                Args:    []string{"-y", "mcp-remote", "{url}"},
            },
            provider.TransportHTTP: {
                Command: "npx",
                Args:    []string{"-y", "mcp-remote", "{url}"},
            },
        },
    },
    config.AppClaudeCode: {
        // Managed via `claude mcp add` CLI; supports both transports.
        Supports: TransportSupport{StreamableHTTP: true, HTTP: true, Stdio: true},
    },
    config.AppCursor: {
        Supports: TransportSupport{StreamableHTTP: true, HTTP: true, Stdio: true},
    },
    config.AppVSCode: {
        // VS Code uses "type":"http" (not streamable-http) for HTTP servers.
        Supports: TransportSupport{HTTP: true, Stdio: true},
    },
    config.AppWindsurf: {
        Supports: TransportSupport{StreamableHTTP: true, HTTP: true, Stdio: true},
    },
    config.AppZed: {
        Supports: TransportSupport{StreamableHTTP: true, HTTP: true, Stdio: true},
    },
    config.AppRooCode: {
        // Roo Code uses "type":"streamable-http" extra field.
        Supports: TransportSupport{StreamableHTTP: true, Stdio: true},
    },
    config.AppOpenCode: {
        Supports: TransportSupport{StreamableHTTP: true, HTTP: true, Stdio: true},
    },
    config.AppKiro: {
        Supports: TransportSupport{StreamableHTTP: true, HTTP: true, Stdio: true},
    },
    config.AppGeminiCLI: {
        // Gemini CLI does not support local stdio subprocess servers.
        Supports: TransportSupport{StreamableHTTP: true, HTTP: true},
    },
    config.AppAntigravity: {
        Supports: TransportSupport{StreamableHTTP: true, HTTP: true},
    },
    config.AppCodexCLI: {
        Supports: TransportSupport{StreamableHTTP: true, HTTP: true, Stdio: true},
    },
}
```

#### Acceptance

- `pkg/client` compiles: `go build ./pkg/client/...`
- All 12 `AppID` values from `pkg/config/paths.go` are present as keys in `Matrix`

---

### T04 — Create `pkg/client/adapter.go`

**Phase:** 3A  
**Depends on:** T03  
**Blocks:** T05, T06  
**New file:** `pkg/client/adapter.go`

```go
// pkg/client/adapter.go
package client

import (
    "strings"

    "github.com/nawodyaishan/universal-mcp-sync/pkg/config"
    "github.com/nawodyaishan/universal-mcp-sync/pkg/provider"
)

// Adapt returns a transport config suitable for appID.
//
// If the client supports cfg.Type natively, cfg is returned unchanged.
// If the client has a bridge for cfg.Type, the bridged stdio config is returned.
// If neither, cfg is returned unchanged — callers must check CanHandle first
// and set a SkipReason if it returns false.
func Adapt(appID config.AppID, cfg provider.MCPConfig) provider.MCPConfig {
    cap, ok := Matrix[appID]
    if !ok {
        return cfg
    }
    if supportsTransport(cap.Supports, cfg.Type) {
        return cfg
    }
    if bridge, ok := cap.Bridge[cfg.Type]; ok {
        return applyBridge(bridge, cfg)
    }
    return cfg
}

// CanHandle reports whether appID can handle transport, either natively or via a bridge.
// Returns false if the client has no support and no bridge for the given transport.
func CanHandle(appID config.AppID, transport provider.TransportType) bool {
    cap, ok := Matrix[appID]
    if !ok {
        return false
    }
    if supportsTransport(cap.Supports, transport) {
        return true
    }
    _, hasBridge := cap.Bridge[transport]
    return hasBridge
}

func supportsTransport(s TransportSupport, t provider.TransportType) bool {
    switch t {
    case provider.TransportStdio:
        return s.Stdio
    case provider.TransportStreamableHTTP:
        return s.StreamableHTTP
    case provider.TransportSSE:
        return s.SSE
    case provider.TransportHTTP:
        return s.HTTP
    }
    return false
}

func applyBridge(bridge *BridgeConfig, cfg provider.MCPConfig) provider.MCPConfig {
    args := make([]string, len(bridge.Args))
    for i, arg := range bridge.Args {
        args[i] = strings.ReplaceAll(arg, "{url}", cfg.URL)
    }
    return provider.MCPConfig{
        Type:    provider.TransportStdio,
        Command: bridge.Command,
        Args:    args,
    }
}
```

#### Acceptance

- `go build ./pkg/client/...` passes

---

### T05 — Tests for `pkg/client`

**Phase:** 3A  
**Depends on:** T04  
**Blocks:** T06  
**New files:** `pkg/client/adapter_test.go`, `pkg/client/capabilities_test.go`

```go
// pkg/client/adapter_test.go
package client_test

import (
    "testing"

    "github.com/nawodyaishan/universal-mcp-sync/pkg/client"
    "github.com/nawodyaishan/universal-mcp-sync/pkg/config"
    "github.com/nawodyaishan/universal-mcp-sync/pkg/provider"
)

func TestAdapt(t *testing.T) {
    remoteHTTP := provider.MCPConfig{
        Type: provider.TransportStreamableHTTP,
        URL:  "https://mcp.exa.ai/mcp?exaApiKey=test",
    }
    remoteHTTPLegacy := provider.MCPConfig{
        Type: provider.TransportHTTP,
        URL:  "https://mcp.exa.ai/mcp?exaApiKey=test",
    }
    stdioGitHub := provider.MCPConfig{
        Type:    provider.TransportStdio,
        Command: "npx",
        Args:    []string{"-y", "@modelcontextprotocol/server-github"},
        Env:     map[string]string{"GITHUB_PERSONAL_ACCESS_TOKEN": "ghp_test"},
    }

    tests := []struct {
        name      string
        appID     config.AppID
        input     provider.MCPConfig
        wantType  provider.TransportType
        wantCmd   string
        wantURL   string // non-empty means check URL preserved in bridge args
    }{
        {
            name:    "ClaudeDesktop bridges StreamableHTTP to stdio",
            appID:   config.AppClaudeDesktop,
            input:   remoteHTTP,
            wantType: provider.TransportStdio,
            wantCmd: "npx",
            wantURL: remoteHTTP.URL,
        },
        {
            name:    "ClaudeDesktop bridges legacy HTTP to stdio",
            appID:   config.AppClaudeDesktop,
            input:   remoteHTTPLegacy,
            wantType: provider.TransportStdio,
            wantCmd: "npx",
            wantURL: remoteHTTPLegacy.URL,
        },
        {
            name:    "ClaudeDesktop passes stdio through unchanged",
            appID:   config.AppClaudeDesktop,
            input:   stdioGitHub,
            wantType: provider.TransportStdio,
            wantCmd: "npx",
        },
        {
            name:    "Cursor passes StreamableHTTP through unchanged",
            appID:   config.AppCursor,
            input:   remoteHTTP,
            wantType: provider.TransportStreamableHTTP,
        },
        {
            name:    "Cursor passes stdio through unchanged",
            appID:   config.AppCursor,
            input:   stdioGitHub,
            wantType: provider.TransportStdio,
            wantCmd: "npx",
        },
        {
            name:    "GeminiCLI passes StreamableHTTP unchanged (no bridge needed)",
            appID:   config.AppGeminiCLI,
            input:   remoteHTTP,
            wantType: provider.TransportStreamableHTTP,
        },
        {
            name:    "GeminiCLI returns stdio unchanged (caller must check CanHandle)",
            appID:   config.AppGeminiCLI,
            input:   stdioGitHub,
            wantType: provider.TransportStdio, // pass-through, CanHandle=false
        },
        {
            name:    "Unknown AppID returns cfg unchanged",
            appID:   config.AppID("does-not-exist"),
            input:   remoteHTTP,
            wantType: provider.TransportStreamableHTTP,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := client.Adapt(tt.appID, tt.input)
            if got.Type != tt.wantType {
                t.Errorf("Type: got %q, want %q", got.Type, tt.wantType)
            }
            if tt.wantCmd != "" && got.Command != tt.wantCmd {
                t.Errorf("Command: got %q, want %q", got.Command, tt.wantCmd)
            }
            if tt.wantURL != "" {
                found := false
                for _, arg := range got.Args {
                    if arg == tt.wantURL {
                        found = true
                        break
                    }
                }
                if !found {
                    t.Errorf("URL %q not found in bridge args %v", tt.wantURL, got.Args)
                }
            }
        })
    }
}

func TestCanHandle(t *testing.T) {
    tests := []struct {
        appID     config.AppID
        transport provider.TransportType
        want      bool
    }{
        {config.AppClaudeDesktop, provider.TransportStdio, true},
        {config.AppClaudeDesktop, provider.TransportStreamableHTTP, true},  // via bridge
        {config.AppClaudeDesktop, provider.TransportHTTP, true},            // via bridge
        {config.AppGeminiCLI, provider.TransportStreamableHTTP, true},
        {config.AppGeminiCLI, provider.TransportStdio, false},              // no support, no bridge
        {config.AppAntigravity, provider.TransportStdio, false},
        {config.AppCursor, provider.TransportStdio, true},
        {config.AppID("unknown"), provider.TransportStdio, false},
    }
    for _, tt := range tests {
        got := client.CanHandle(tt.appID, tt.transport)
        if got != tt.want {
            t.Errorf("CanHandle(%q, %q) = %v, want %v", tt.appID, tt.transport, got, tt.want)
        }
    }
}
```

```go
// pkg/client/capabilities_test.go
package client_test

import (
    "testing"

    "github.com/nawodyaishan/universal-mcp-sync/pkg/client"
    "github.com/nawodyaishan/universal-mcp-sync/pkg/config"
)

func TestMatrixCoversAllAppIDs(t *testing.T) {
    for _, id := range config.AppOrder {
        if _, ok := client.Matrix[id]; !ok {
            t.Errorf("Matrix missing entry for AppID %q — add it to pkg/client/capabilities.go", id)
        }
    }
}

func TestClaudeDesktopCapabilities(t *testing.T) {
    cap := client.Matrix[config.AppClaudeDesktop]
    if !cap.Supports.Stdio {
        t.Error("ClaudeDesktop must support Stdio natively")
    }
    if cap.Supports.StreamableHTTP {
        t.Error("ClaudeDesktop does not natively support StreamableHTTP; it uses a bridge")
    }
    if cap.Bridge == nil || len(cap.Bridge) == 0 {
        t.Error("ClaudeDesktop must declare bridges for HTTP transports")
    }
}

func TestGeminiCLICapabilities(t *testing.T) {
    cap := client.Matrix[config.AppGeminiCLI]
    if cap.Supports.Stdio {
        t.Error("GeminiCLI does not support local stdio subprocess servers")
    }
    if !cap.Supports.StreamableHTTP {
        t.Error("GeminiCLI must support StreamableHTTP")
    }
}
```

#### Acceptance

- `go test ./pkg/client/...` passes with no failures

---

### T06 — Refactor `pkg/app/app.go` — remove `pkg/exa` imports, integrate `client.Adapt()`

**Phase:** 3A  
**Depends on:** T02, T05  
**Blocks:** T07, T08  
**Files changed:** `pkg/app/app.go`, `cmd/usync/main.go`

This is the largest task. Work through it section by section. After each sub-step, run `go build ./...` to confirm no compile errors before continuing.

#### Step 6.1 — Update import block in `app.go`

Remove line 18 (`"github.com/nawodyaishan/universal-mcp-sync/pkg/exa"`).  
Add two new imports:

```go
"github.com/nawodyaishan/universal-mcp-sync/pkg/client"
"github.com/nawodyaishan/universal-mcp-sync/pkg/redact"
```

The full import block becomes:

```go
import (
    "context"
    "errors"
    "fmt"
    "io"
    "log/slog"
    "os"
    "os/exec"
    "path/filepath"
    "sort"
    "strings"
    "time"

    "github.com/nawodyaishan/universal-mcp-sync/pkg/client"
    "github.com/nawodyaishan/universal-mcp-sync/pkg/config"
    "github.com/nawodyaishan/universal-mcp-sync/pkg/provider"
    "github.com/nawodyaishan/universal-mcp-sync/pkg/redact"
    "github.com/nawodyaishan/universal-mcp-sync/pkg/verify"
)
```

#### Step 6.2 — Replace `configForTarget()` call with `client.Adapt()` + skip logic

In `PrepareProvider()` at line 156–170, replace:

```go
// BEFORE (lines 156-170)
for _, file := range appConfig.Files {
    fileCfg := configForTarget(appConfig.ID, prov.ID(), cfg)
    plan.Operations = append(plan.Operations, Operation{
        ...
        Config: fileCfg,
        ...
    })
}
```

With:

```go
// AFTER
if !client.CanHandle(appConfig.ID, cfg.Type) {
    plan.Warnings = append(plan.Warnings, fmt.Sprintf(
        "%s does not support %s transport — skipping %s",
        appConfig.Name, cfg.Type, prov.ID(),
    ))
    continue
}
for _, file := range appConfig.Files {
    fileCfg := client.Adapt(appConfig.ID, cfg)
    plan.Operations = append(plan.Operations, Operation{
        AppID:           appConfig.ID,
        AppName:         appConfig.Name,
        FileLabel:       file.Label,
        Path:            file.Path,
        Kind:            file.Kind,
        CredentialLabel: profile.Label,
        ProviderID:      prov.ID(),
        Config:          fileCfg,
        BackupPath:      backupPathFor(file, m.Now()),
        WillCreate:      !file.Exists,
    })
}
```

#### Step 6.3 — Delete `configForTarget()` function

Delete lines 176–185 (the entire `configForTarget` function). It is fully replaced by `client.Adapt()`.

#### Step 6.4 — Update `Prepare()` to use `redact.Key` instead of `exa.RedactKey`

`Prepare()` at lines 187–205 uses `exa.RedactKey(key)` at line 201. Change that one call:

```go
// BEFORE (line 201)
Label: exa.RedactKey(key),

// AFTER
Label: redact.Key(key),
```

`Prepare()` otherwise stays in `app.go` unchanged — it is used by existing tests.

#### Step 6.5 — Move `LoadInitialKeys()` to `cmd/usync/main.go`

Delete lines 392–409 (`LoadInitialKeys` function) from `app.go`.

Add to `cmd/usync/main.go` as a package-level function (not a method):

```go
func loadInitialKeys(keysCSV, keysFile string) ([]string, string, error) {
    if keysCSV != "" {
        keys, err := exa.ParseKeysCSV(keysCSV)
        return keys, keysCSV, err
    }
    if keysFile != "" {
        keys, err := exa.ParseKeysFile(keysFile)
        if err != nil {
            return nil, "", err
        }
        data, err := os.ReadFile(keysFile)
        if err != nil {
            return nil, "", err
        }
        return keys, string(data), nil
    }
    return nil, "", nil
}
```

Update the call site in `main()` from:
```go
initialKeys, initialRaw, err := app.LoadInitialKeys(keysCSV, keysFile)
```
to:
```go
initialKeys, initialRaw, err := loadInitialKeys(keysCSV, keysFile)
```

#### Step 6.6 — Update `Apply()` CLI verification to use `op.ProviderID`

In `Apply()` at lines 245–253, replace the hardcoded `"exa"` strings in the CLI verification block.

The current pattern uses `seenApps[config.AppCodexCLI]` etc. Replace with a loop over CLI operations that uses the actual provider ID:

```go
// BEFORE (lines 244-253)
result.Verification = append(result.Verification, verifyFiles(prepared)...)
if seenApps[config.AppCodexCLI] {
    result.Verification = append(result.Verification, verify.VerifyOptionalCLI(m.Runner, "codex", "mcp", "get", "exa"))
}
if seenApps[config.AppClaudeCode] {
    result.Verification = append(result.Verification, verify.VerifyOptionalCLI(m.Runner, "claude", "mcp", "get", "exa"))
}
if seenApps[config.AppGeminiCLI] {
    result.Verification = append(result.Verification, verify.VerifyOptionalCLI(m.Runner, "gemini", "mcp", "get", "exa"))
}

// AFTER
result.Verification = append(result.Verification, verifyFiles(prepared)...)

type cliVerifyKey struct{ appID config.AppID; providerID string }
seen := make(map[cliVerifyKey]bool)
for _, op := range cliOps {
    key := cliVerifyKey{op.AppID, op.ProviderID}
    if seen[key] { continue }
    seen[key] = true
    switch op.AppID {
    case config.AppClaudeCode:
        result.Verification = append(result.Verification,
            verify.VerifyOptionalCLI(m.Runner, "claude", "mcp", "get", op.ProviderID))
    }
}
// Non-CLI app verifications from seenApps
for appID := range seenApps {
    // Find the provider ID from any operation for this app
    provID := ""
    for _, op := range plan.Operations {
        if op.AppID == appID && op.SkipReason == "" {
            provID = op.ProviderID
            break
        }
    }
    if provID == "" { continue }
    key := cliVerifyKey{appID, provID}
    if seen[key] { continue }
    seen[key] = true
    switch appID {
    case config.AppCodexCLI:
        result.Verification = append(result.Verification,
            verify.VerifyOptionalCLI(m.Runner, "codex", "mcp", "get", provID))
    case config.AppGeminiCLI:
        result.Verification = append(result.Verification,
            verify.VerifyOptionalCLI(m.Runner, "gemini", "mcp", "get", provID))
    }
}
```

#### Step 6.7 — Update `applyClaudeCode()` and `osRunner.Run()`

`applyClaudeCode()` at line 259–270:

```go
// Line 262 — BEFORE
warning := fmt.Sprintf("claude mcp remove exa: %s", exa.RedactText(err.Error()))
// AFTER
warning := fmt.Sprintf("claude mcp remove %s: %s", op.ProviderID, redact.Text(err.Error()))

// Line 265 — BEFORE
return fmt.Errorf("claude mcp add exa: %s", exa.RedactText(err.Error()))
// AFTER
return fmt.Errorf("claude mcp add %s: %s", op.ProviderID, redact.Text(err.Error()))

// Line 268 — BEFORE
result.UpdatedTargets = append(result.UpdatedTargets, "claude mcp add exa")
// AFTER
result.UpdatedTargets = append(result.UpdatedTargets, "claude mcp add "+op.ProviderID)
```

`osRunner.Run()` at lines 422–429:

```go
// BEFORE
func (osRunner) Run(name string, args ...string) (string, error) {
    cmd := exec.Command(name, args...)
    output, err := cmd.CombinedOutput()
    if err != nil {
        return exa.RedactText(string(output)), errors.New(exa.RedactText(strings.TrimSpace(string(output))))
    }
    return exa.RedactText(string(output)), nil
}

// AFTER
func (osRunner) Run(name string, args ...string) (string, error) {
    cmd := exec.Command(name, args...)
    output, err := cmd.CombinedOutput()
    if err != nil {
        return redact.Text(string(output)), errors.New(redact.Text(strings.TrimSpace(string(output))))
    }
    return redact.Text(string(output)), nil
}
```

#### Step 6.8 — Update `m.log()` and `redactAttrs()`

`m.log()` at line 596:
```go
// BEFORE
m.Logger.Log(context.Background(), level, exa.RedactText(msg), redactAttrs(attrs)...)
// AFTER
m.Logger.Log(context.Background(), level, redact.Text(msg), redactAttrs(attrs)...)
```

`redactAttrs()` at line 603:
```go
// BEFORE
redacted = append(redacted, exa.RedactText(value))
// AFTER
redacted = append(redacted, redact.Text(value))
```

#### Acceptance for T06

- `go build ./...` passes with no errors
- `grep -n "pkg/exa" pkg/app/app.go` returns nothing
- `make test` passes (all existing tests green)
- `make lint` passes

---

### T07 — Generalize `pkg/verify/verify.go`

**Phase:** 3A  
**Depends on:** T06  
**Files changed:** `pkg/verify/verify.go`

Add a generic fallback path to `VerifyProviderFile()` and two new helpers so non-Exa providers get meaningful verification rather than an immediate failure.

#### Changes

**`VerifyProviderFile()` at line 51** — add generic fallback:

```go
// BEFORE
func VerifyProviderFile(path string, kind config.FileKind, providerID string, cfg provider.MCPConfig) Result {
    if providerID == "exa" {
        return verifyExaProviderFile(path, kind, cfg)
    }
    return failure(path, fmt.Sprintf("verification not implemented for provider %s", providerID))
}

// AFTER
func VerifyProviderFile(path string, kind config.FileKind, providerID string, cfg provider.MCPConfig) Result {
    if providerID == "exa" {
        return verifyExaProviderFile(path, kind, cfg)
    }
    return verifyGenericProviderFile(path, kind, providerID, cfg)
}
```

**Add these new functions** (insert after `verifyExaProviderFile` at line 58):

```go
func verifyGenericProviderFile(path string, kind config.FileKind, providerID string, cfg provider.MCPConfig) Result {
    server, err := readServerEntryByKind(path, kind, providerID)
    if err != nil {
        return failure(path, err.Error())
    }
    if cfg.Type == provider.TransportStdio {
        return verifyGenericStdioServer(path, server, cfg)
    }
    return verifyGenericHTTPServer(path, server)
}

func verifyGenericStdioServer(path string, server map[string]any, cfg provider.MCPConfig) Result {
    command, _ := server["command"].(string)
    if command == "" {
        return failure(path, "missing stdio command field")
    }
    if command != cfg.Command {
        return Result{
            Target:  path,
            Status:  StatusWarning,
            Details: []string{fmt.Sprintf("command=%s (expected %s)", command, cfg.Command)},
        }
    }
    return Result{
        Target:  path,
        Status:  StatusOK,
        Details: []string{fmt.Sprintf("command=%s", command)},
    }
}

func verifyGenericHTTPServer(path string, server map[string]any) Result {
    urlValue := getURLField(server)
    if urlValue == "" {
        return failure(path, "missing URL field (checked: url, httpUrl, serverUrl)")
    }
    if _, err := url.Parse(urlValue); err != nil {
        return failure(path, fmt.Sprintf("invalid URL: %v", err))
    }
    return Result{
        Target:  path,
        Status:  StatusOK,
        Details: []string{"url present and valid"},
    }
}

// readServerEntryByKind dispatches to the correct reader based on FileKind.
func readServerEntryByKind(path string, kind config.FileKind, providerID string) (map[string]any, error) {
    switch kind {
    case config.FileKindMCPServers:
        return readNestedServerEntry(path, "mcpServers", providerID)
    case config.FileKindBareMCPServers:
        return readRootServerEntry(path, providerID)
    case config.FileKindNamedServer:
        // Try common root keys; fall back to root-level entry
        for _, rootKey := range []string{"servers", "context_servers", "mcp"} {
            s, err := readNestedServerEntry(path, rootKey, providerID)
            if err == nil {
                return s, nil
            }
        }
        return readRootServerEntry(path, providerID)
    default:
        return nil, fmt.Errorf("verification not supported for kind %q", kind)
    }
}
```

#### Tests to add

In `pkg/verify/verify_test.go`, add:

```go
func TestVerifyProviderFile_GenericStdio(t *testing.T) {
    dir := t.TempDir()
    path := filepath.Join(dir, "config.json")
    content := `{"mcpServers":{"github":{"command":"npx","args":["-y","@modelcontextprotocol/server-github"]}}}`
    os.WriteFile(path, []byte(content), 0o600)

    cfg := provider.MCPConfig{Type: provider.TransportStdio, Command: "npx"}
    result := VerifyProviderFile(path, config.FileKindMCPServers, "github", cfg)
    if result.Status != StatusOK {
        t.Errorf("expected OK, got %s: %v", result.Status, result.Details)
    }
}

func TestVerifyProviderFile_GenericHTTP(t *testing.T) {
    dir := t.TempDir()
    path := filepath.Join(dir, "config.json")
    content := `{"mcpServers":{"brave":{"url":"https://api.brave.com/mcp"}}}`
    os.WriteFile(path, []byte(content), 0o600)

    cfg := provider.MCPConfig{Type: provider.TransportStreamableHTTP, URL: "https://api.brave.com/mcp"}
    result := VerifyProviderFile(path, config.FileKindMCPServers, "brave", cfg)
    if result.Status != StatusOK {
        t.Errorf("expected OK, got %s: %v", result.Status, result.Details)
    }
}

func TestVerifyProviderFile_GenericMissingEntry(t *testing.T) {
    dir := t.TempDir()
    path := filepath.Join(dir, "config.json")
    os.WriteFile(path, []byte(`{"mcpServers":{}}`), 0o600)

    cfg := provider.MCPConfig{Type: provider.TransportStdio, Command: "npx"}
    result := VerifyProviderFile(path, config.FileKindMCPServers, "github", cfg)
    if result.Status != StatusFailed {
        t.Errorf("expected Failed for missing entry, got %s", result.Status)
    }
}
```

#### Acceptance

- `go test ./pkg/verify/...` passes
- `VerifyProviderFile` with `providerID="github"` returns `StatusOK` or `StatusWarning`, never `StatusFailed` on a well-formed file

---

### T08 — Generalize format strings in `app.go` and update affected tests

**Phase:** 3A  
**Depends on:** T06  
**Files changed:** `pkg/app/app.go`, `pkg/app/app_test.go`

#### Changes in `app.go`

**`FormatPlan()` at line 315–339:**

```go
// Line 315 — BEFORE
builder.WriteString("Exa MCP update plan\n")
builder.WriteString("===================\n")
// AFTER
builder.WriteString("MCP sync plan\n")
builder.WriteString("=============\n")
```

```go
// Line 336 — DELETE entirely:
fmt.Fprintf(&builder, "  tools: %d\n", len(exa.DefaultTools))
// This line is Exa-specific. Remove it. Tool count is not a generic concept.
```

**`FormatApplyResult()` at line 342–343:**

```go
// BEFORE
builder.WriteString("Exa MCP apply result\n")
builder.WriteString("====================\n")
// AFTER
builder.WriteString("MCP sync result\n")
builder.WriteString("===============\n")
```

**`FormatPlan()` at line 331** — update the backup label (currently says "not applicable"):

This is fine as-is. No change needed.

#### Update in `app_test.go`

Line 99 checks that format output does not contain raw API keys — this test is independent of "Exa MCP" branding. No change needed.

However, if any test asserts the exact string `"Exa MCP"`, update it. Run `grep -n "Exa MCP" pkg/app/` to confirm none exist in tests.

#### Acceptance

- `grep -n "Exa MCP" pkg/app/app.go` returns nothing
- `grep -n "exa.DefaultTools" pkg/app/app.go` returns nothing
- `make test` passes

---

### T09 — Add coverage gate to Makefile and CI

**Phase:** 3A  
**Depends on:** T08 (i.e., all Phase 3A code should be merged first so baseline is stable)  
**Files changed:** `Makefile`, `scripts/test.sh` (or CI directly), `.github/workflows/ci.yml`

#### Makefile

Add after the `test` target:

```makefile
.PHONY: coverage-check
coverage-check:
	go test -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out | awk \
	  '/total:/ { gsub(/%/,"",$$3); if ($$3+0 < 65.0) \
	    { print "FAIL: total coverage " $$3 "% is below 65% gate"; exit 1 } \
	    else { print "PASS: total coverage " $$3 "%" } }'
```

#### `.github/workflows/ci.yml`

In the `test` job, after the `Test` step (line 86), add:

```yaml
- name: Coverage gate
  run: make coverage-check
```

Do not add new jobs or workflows. Do not change the `compatibility` job.

#### Acceptance

- `make coverage-check` exits 0 when coverage ≥ 65%
- `make coverage-check` exits 1 and prints `FAIL: total coverage X%` when below threshold
- CI `test` job fails if coverage drops below 65%

---

## Phase 3B — GitHub Provider (first stdio provider)

Begin after all Phase 3A tasks are merged to `main`.

---

### T10 — Implement `pkg/provider/github.go`

**Phase:** 3B  
**Blocks:** T11, T13  
**New file:** `pkg/provider/github.go`

```go
package provider

import (
    "fmt"
    "regexp"
)

// githubPATRE matches the three known GitHub PAT formats:
// - Classic: 40-char hex (legacy)
// - Fine-grained: ghp_ prefix + 36 alphanumeric chars
// - Fine-grained v2: github_pat_ prefix + long alphanumeric+underscore string
var githubPATRE = regexp.MustCompile(
    `^(?:[0-9a-f]{40}|ghp_[A-Za-z0-9]{36}|github_pat_[A-Za-z0-9_]{59,})$`,
)

// GitHubProvider installs the official MCP GitHub server via npx.
// Reference: https://github.com/modelcontextprotocol/servers/tree/main/src/github
type GitHubProvider struct{}

func NewGitHubProvider() *GitHubProvider { return &GitHubProvider{} }

func (p *GitHubProvider) ID() string          { return "github" }
func (p *GitHubProvider) Name() string        { return "GitHub" }
func (p *GitHubProvider) Description() string {
    return "GitHub repository, issue, and PR management via the official MCP server."
}

func (p *GitHubProvider) RequiredCredentials() []CredentialSpec {
    return []CredentialSpec{
        {
            Key:         "GITHUB_PERSONAL_ACCESS_TOKEN",
            Label:       "GitHub Personal Access Token",
            Description: "Create at github.com/settings/tokens. Requires 'repo' scope. Format: ghp_... or github_pat_...",
            Secret:      true,
            MultiValue:  false,
            Validator: func(s string) error {
                if s == "" {
                    return fmt.Errorf("token is required")
                }
                if !githubPATRE.MatchString(s) {
                    return fmt.Errorf("unrecognised GitHub PAT format (expected ghp_..., github_pat_..., or 40-char hex)")
                }
                return nil
            },
        },
    }
}

func (p *GitHubProvider) GenerateConfig(credentials map[string]string) (MCPConfig, error) {
    pat := credentials["GITHUB_PERSONAL_ACCESS_TOKEN"]
    if pat == "" {
        return MCPConfig{}, fmt.Errorf("missing GITHUB_PERSONAL_ACCESS_TOKEN")
    }
    return MCPConfig{
        Type:    TransportStdio,
        Command: "npx",
        Args:    []string{"-y", "@modelcontextprotocol/server-github"},
        Env:     map[string]string{"GITHUB_PERSONAL_ACCESS_TOKEN": pat},
        Runtime: &PackageRuntime{Type: "npm"},
    }, nil
}
```

**New file: `pkg/provider/github_test.go`**

```go
package provider_test

import (
    "strings"
    "testing"

    "github.com/nawodyaishan/universal-mcp-sync/pkg/provider"
)

func TestGitHubProvider_ID(t *testing.T) {
    p := provider.NewGitHubProvider()
    if p.ID() != "github" {
        t.Errorf("ID() = %q, want \"github\"", p.ID())
    }
}

func TestGitHubProvider_GenerateConfig(t *testing.T) {
    validPAT := "ghp_" + strings.Repeat("a", 36)

    tests := []struct {
        name    string
        creds   map[string]string
        wantErr bool
        check   func(*testing.T, provider.MCPConfig)
    }{
        {
            name:  "valid ghp_ PAT produces correct stdio config",
            creds: map[string]string{"GITHUB_PERSONAL_ACCESS_TOKEN": validPAT},
            check: func(t *testing.T, cfg provider.MCPConfig) {
                if cfg.Type != provider.TransportStdio {
                    t.Errorf("Type = %q, want stdio", cfg.Type)
                }
                if cfg.Command != "npx" {
                    t.Errorf("Command = %q, want npx", cfg.Command)
                }
                if len(cfg.Args) < 2 || cfg.Args[1] != "@modelcontextprotocol/server-github" {
                    t.Errorf("Args = %v, expected @modelcontextprotocol/server-github", cfg.Args)
                }
                if cfg.Env["GITHUB_PERSONAL_ACCESS_TOKEN"] != validPAT {
                    t.Errorf("Env does not contain the PAT")
                }
                if cfg.Runtime == nil || cfg.Runtime.Type != "npm" {
                    t.Errorf("Runtime should be npm")
                }
            },
        },
        {
            name:    "missing PAT returns error",
            creds:   map[string]string{},
            wantErr: true,
        },
        {
            name:    "empty PAT returns error",
            creds:   map[string]string{"GITHUB_PERSONAL_ACCESS_TOKEN": ""},
            wantErr: true,
        },
    }

    p := provider.NewGitHubProvider()
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            cfg, err := p.GenerateConfig(tt.creds)
            if (err != nil) != tt.wantErr {
                t.Fatalf("GenerateConfig() error = %v, wantErr %v", err, tt.wantErr)
            }
            if tt.check != nil {
                tt.check(t, cfg)
            }
        })
    }
}

func TestGitHubProvider_CredentialValidator(t *testing.T) {
    p := provider.NewGitHubProvider()
    specs := p.RequiredCredentials()
    if len(specs) != 1 {
        t.Fatalf("expected 1 credential spec, got %d", len(specs))
    }
    validate := specs[0].Validator

    valid := []string{
        "ghp_" + strings.Repeat("a", 36),
        "github_pat_" + strings.Repeat("b", 59),
        strings.Repeat("a", 40), // classic 40-char hex
    }
    for _, v := range valid {
        if err := validate(v); err != nil {
            t.Errorf("validator rejected valid PAT %q: %v", v, err)
        }
    }

    invalid := []string{
        "",
        "not-a-token",
        "ghp_tooshort",
        "ghp_" + strings.Repeat("!", 36), // wrong chars
    }
    for _, v := range invalid {
        if err := validate(v); err == nil {
            t.Errorf("validator accepted invalid PAT %q", v)
        }
    }
}
```

#### Acceptance

- `go test ./pkg/provider/...` passes
- `GitHubProvider` satisfies `MCPProvider` interface (compile check)

---

### T11 — Register `GitHubProvider` in `pkg/provider/registry.go`

**Phase:** 3B  
**Depends on:** T10  
**Files changed:** `pkg/provider/registry.go`

Add one line to `DefaultRegistry()`:

```go
func DefaultRegistry() Registry {
    r := Registry{
        providers: make(map[string]MCPProvider),
        order:     []string{},
    }
    r.register(NewExaProvider())
    r.register(NewGitHubProvider()) // ← add this line
    return r
}
```

#### Acceptance

- `make test` passes
- `provider.DefaultRegistry().Get("github")` returns a non-nil provider

---

### T12 — Verify capability matrix for stdio clients (review task)

**Phase:** 3B  
**Depends on:** T11  
**Files reviewed:** `pkg/client/capabilities.go`, `pkg/config/paths.go`

This is a review + possible correction task, not new code.

For each client in `Matrix`, confirm whether it actually supports local stdio subprocess servers (i.e., local `npx`/`uvx` processes). The GitHub provider is stdio-only. Clients that do not support stdio will have `SkipReason` set and will be excluded from the GitHub plan.

**Verified behaviour (from official docs and testing):**

| Client | stdio support | Evidence |
|---|---|---|
| Claude Desktop | ✓ | Official MCP docs — stdio is primary transport |
| Claude Code | ✓ | `claude mcp add --transport stdio` |
| Cursor | ✓ | `.cursor/mcp.json` supports `command`/`args` |
| VS Code | ✓ | `mcp.json` supports `command`/`args` |
| Windsurf | ✓ | `mcp_config.json` supports stdio |
| Zed | ✓ | `settings.json` context_servers supports stdio |
| Roo Code | ✓ | Documented in extension settings |
| OpenCode | ✓ | Supports stdio |
| Kiro | ✓ | `mcp.json` supports stdio |
| Gemini CLI | ✗ | Only remote HTTP/StreamableHTTP in `.gemini/settings.json` |
| Antigravity | ✗ | Remote HTTP only |
| Codex CLI | ✓ | `config.toml` supports local servers |

**Action:** Confirm `Matrix` in `pkg/client/capabilities.go` matches this table. If any entry is wrong, fix it. The `capabilities_test.go` added in T05 will catch discrepancies.

**Note on Codex TOML:** `pkg/config/toml_update.go:10` currently returns an error for stdio: `"stdio transport is not supported in Codex TOML"`. This means even though Codex CLI _conceptually_ supports stdio, the TOML writer does not implement it yet.

**Action for Codex stdio:** Set `config.AppCodexCLI`'s `Supports.Stdio = false` in the matrix for now, to prevent the plan from generating an operation that will fail at write time. Add a `TODO` comment in `capabilities.go`. Implement Codex TOML stdio writing in a future task.

#### Acceptance

- Matrix entry for `AppCodexCLI` has `Stdio: false` (until TOML writer supports it)
- `TestMatrixCoversAllAppIDs` still passes

---

### T13 — Add GitHub QA scenarios to `pkg/app/qa_scenarios_test.go`

**Phase:** 3B  
**Depends on:** T11, T12  
**Files changed:** `pkg/app/qa_scenarios_test.go`

Add three new test functions:

```go
func TestQAGitHubStdioSupportedClients(t *testing.T) {
    homeDir := t.TempDir()

    // Write empty config files for all clients that support stdio
    paths := map[config.AppID]string{
        config.AppClaudeDesktop: filepath.Join(homeDir, "Library", "Application Support", "Claude", "claude_desktop_config.json"),
        config.AppCursor:        filepath.Join(homeDir, ".cursor", "mcp.json"),
        config.AppVSCode:        filepath.Join(homeDir, ".vscode", "mcp.json"),
        config.AppWindsurf:      filepath.Join(homeDir, ".codeium", "windsurf", "mcp_config.json"),
        config.AppZed:           filepath.Join(homeDir, ".config", "zed", "settings.json"),
        config.AppRooCode:       filepath.Join(homeDir, "Library", "Application Support", "Code", "User", "globalStorage", "saoudrizwan.claude-dev", "settings", "mcp_settings.json"),
        config.AppOpenCode:      filepath.Join(homeDir, ".opencode.json"),
        config.AppKiro:          filepath.Join(homeDir, ".kiro", "settings", "mcp.json"),
    }
    for _, p := range paths {
        mustWriteFile(t, p, []byte("{}"))
    }

    manager, err := NewManager(homeDir, fixedNow(), fakeRunner{available: map[string]bool{"claude": true}})
    if err != nil {
        t.Fatalf("NewManager: %v", err)
    }

    prov := provider.NewGitHubProvider()
    pat := "ghp_" + strings.Repeat("a", 36)
    profiles := []provider.CredentialProfile{{
        ProviderID: "github",
        Values:     map[string]string{"GITHUB_PERSONAL_ACCESS_TOKEN": pat},
        Label:      "ghp_...aaaa",
    }}

    selected := make(map[config.AppID]bool)
    for id := range paths {
        selected[id] = true
    }
    assignments := DefaultAssignments(selected, 1)

    plan, err := manager.PrepareProvider(prov, profiles, selected, assignments)
    if err != nil {
        t.Fatalf("PrepareProvider: %v", err)
    }

    // No operations should have SkipReason for stdio-capable clients
    for _, op := range plan.Operations {
        if op.SkipReason != "" {
            t.Errorf("unexpected skip for %s: %s", op.AppName, op.SkipReason)
        }
    }

    _, err = manager.Apply(plan)
    if err != nil {
        t.Fatalf("Apply: %v", err)
    }

    // Verify Claude Desktop got stdio command written (no bridge needed for stdio provider)
    data, _ := os.ReadFile(paths[config.AppClaudeDesktop])
    if !bytes.Contains(data, []byte(`"command": "npx"`)) {
        t.Errorf("Claude Desktop: expected stdio command\n%s", data)
    }
    if !bytes.Contains(data, []byte(`"@modelcontextprotocol/server-github"`)) {
        t.Errorf("Claude Desktop: expected GitHub server arg\n%s", data)
    }
    // PAT must appear in env block
    if !bytes.Contains(data, []byte(`"GITHUB_PERSONAL_ACCESS_TOKEN"`)) {
        t.Errorf("Claude Desktop: expected env key in config\n%s", data)
    }

    // Verify Cursor got stdio command written
    data, _ = os.ReadFile(paths[config.AppCursor])
    if !bytes.Contains(data, []byte(`"command": "npx"`)) {
        t.Errorf("Cursor: expected stdio command\n%s", data)
    }
}

func TestQAGitHubSkippedOnHTTPOnlyClients(t *testing.T) {
    homeDir := t.TempDir()

    geminiPath := filepath.Join(homeDir, ".gemini", "settings.json")
    antigravityPath := filepath.Join(homeDir, ".gemini", "antigravity", "mcp_config.json")
    mustWriteFile(t, geminiPath, []byte("{}"))
    mustWriteFile(t, antigravityPath, []byte("{}"))

    manager, err := NewManager(homeDir, fixedNow(), fakeRunner{})
    if err != nil {
        t.Fatalf("NewManager: %v", err)
    }

    prov := provider.NewGitHubProvider()
    pat := "ghp_" + strings.Repeat("a", 36)
    profiles := []provider.CredentialProfile{{
        ProviderID: "github",
        Values:     map[string]string{"GITHUB_PERSONAL_ACCESS_TOKEN": pat},
        Label:      "ghp_...aaaa",
    }}
    selected := map[config.AppID]bool{
        config.AppGeminiCLI:  true,
        config.AppAntigravity: true,
    }
    assignments := DefaultAssignments(selected, 1)

    plan, err := manager.PrepareProvider(prov, profiles, selected, assignments)
    if err != nil {
        t.Fatalf("PrepareProvider: %v", err)
    }

    // All operations should be skipped for HTTP-only clients with a stdio provider
    skipped := 0
    for _, op := range plan.Operations {
        if op.SkipReason != "" {
            skipped++
        }
    }
    if skipped == 0 && len(plan.Warnings) == 0 {
        t.Error("expected GeminiCLI and Antigravity to be skipped for stdio-only provider")
    }

    // Files should not be modified
    _, err = manager.Apply(plan)
    if err != nil {
        t.Fatalf("Apply: %v", err)
    }

    data, _ := os.ReadFile(geminiPath)
    if !bytes.Equal(data, []byte("{}")) {
        t.Errorf("Gemini settings should not be modified for GitHub stdio provider\n%s", data)
    }
}

func TestQAExaAndGitHubCoexist(t *testing.T) {
    homeDir := t.TempDir()
    cursorPath := filepath.Join(homeDir, ".cursor", "mcp.json")
    mustWriteFile(t, cursorPath, []byte("{}"))

    manager, err := NewManager(homeDir, fixedNow(), fakeRunner{})
    if err != nil {
        t.Fatalf("NewManager: %v", err)
    }
    selected := map[config.AppID]bool{config.AppCursor: true}
    assignments := DefaultAssignments(selected, 1)

    // Apply Exa first
    exaKey := "11111111-1111-1111-1111-111111111111"
    exaPlan, err := manager.Prepare([]string{exaKey}, selected, assignments)
    if err != nil {
        t.Fatalf("Prepare Exa: %v", err)
    }
    if _, err := manager.Apply(exaPlan); err != nil {
        t.Fatalf("Apply Exa: %v", err)
    }

    // Apply GitHub second
    pat := "ghp_" + strings.Repeat("a", 36)
    githubProv := provider.NewGitHubProvider()
    githubProfiles := []provider.CredentialProfile{{
        ProviderID: "github",
        Values:     map[string]string{"GITHUB_PERSONAL_ACCESS_TOKEN": pat},
        Label:      "ghp_...aaaa",
    }}
    githubPlan, err := manager.PrepareProvider(githubProv, githubProfiles, selected, assignments)
    if err != nil {
        t.Fatalf("PrepareProvider GitHub: %v", err)
    }
    if _, err := manager.Apply(githubPlan); err != nil {
        t.Fatalf("Apply GitHub: %v", err)
    }

    data, _ := os.ReadFile(cursorPath)
    // Both providers must be present
    if !bytes.Contains(data, []byte(`"exa"`)) {
        t.Errorf("Cursor: Exa entry should survive GitHub sync\n%s", data)
    }
    if !bytes.Contains(data, []byte(`"github"`)) {
        t.Errorf("Cursor: GitHub entry should be present\n%s", data)
    }
}
```

#### Acceptance

- `go test ./pkg/app/... -run TestQAGitHub` passes
- `go test ./pkg/app/... -run TestQAExa` still passes (no regressions)

---

### T14 — Generic verification for stdio in `pkg/verify`

This task is already covered by T07. The `verifyGenericStdioServer` added in T07 handles GitHub configs automatically. No additional code needed.

**Confirm:** Run `go test ./pkg/verify/... -run TestVerifyProviderFile_GenericStdio` after T11 is in place.

---

### T15 — Update `pkg/tui/preview.go` to display stdio transport details

**Phase:** 3B  
**Depends on:** T11  
**Files changed:** `pkg/tui/preview.go`

In `renderPreviewPlan()` at line 68–69, the mode line currently shows the first operation's transport. Improve it to show per-operation transport and, for stdio, show the command:

Replace lines 66–69:

```go
// BEFORE
fmt.Fprintf(&builder, "Targets   %d %s\n", len(plan.Operations), targetLabel)
fmt.Fprintf(&builder, "Provider  %s\n", first.ProviderID)
fmt.Fprintf(&builder, "Mode      %s transport\n", first.Config.Type)
builder.WriteString("Safety    backups before file writes; credentials stay redacted\n")
```

With:

```go
fmt.Fprintf(&builder, "Targets   %d %s\n", len(plan.Operations), targetLabel)
fmt.Fprintf(&builder, "Provider  %s\n", first.ProviderID)
builder.WriteString("Safety    backups before file writes; credentials stay redacted\n")
```

Then in the per-operation loop (inside the `for index, op := range plan.Operations` block, after `fmt.Fprintf(&builder, "   Config   %s\n", op.FileLabel)`), add:

```go
if op.Config.Type == provider.TransportStdio {
    fmt.Fprintf(&builder, "   Transport stdio (%s %s)\n",
        op.Config.Command, strings.Join(op.Config.Args, " "))
} else {
    fmt.Fprintf(&builder, "   Transport %s\n", op.Config.Type)
}
```

Add `"github.com/nawodyaishan/universal-mcp-sync/pkg/provider"` and `"strings"` to the import block in `preview.go` if not already present.

#### Acceptance

- `go build ./pkg/tui/...` passes
- Manual test: run `make run`, select GitHub provider, reach preview stage — transport line shows `stdio (npx -y @modelcontextprotocol/server-github)`

---

## Phase 3C — TUI Decoupling from `pkg/exa`

Begin after Phase 3B is merged.

---

### T16 — Decouple `pkg/tui/setup_form.go` from `pkg/exa`

**Phase:** 3C  
**Depends on:** T11  
**Files changed:** `pkg/provider/types.go`, `pkg/provider/exa.go`, `pkg/tui/setup_form.go`

#### Why

`setup_form.go:144` hard-codes Exa UUID parsing for multi-value credential fields. Any second multi-value provider would require editing the TUI.

#### Step 16.1 — Add `MultiValueParser` optional interface to `pkg/provider/types.go`

```go
// MultiValueParser is an optional interface for providers whose credential
// input accepts multiple values in one text area (e.g. Exa's multi-key paste field).
// If a provider does not implement this interface, the TUI creates one profile
// per form submission using the raw input values directly.
type MultiValueParser interface {
    // ParseMultiValue parses raw text for credential key into one or more profiles.
    // The returned profiles each represent one independent credential set.
    ParseMultiValue(credentialKey string, raw string) ([]CredentialProfile, error)
}
```

#### Step 16.2 — Implement `MultiValueParser` on `ExaProvider` in `pkg/provider/exa.go`

```go
// ParseMultiValue implements MultiValueParser.
// It extracts multiple UUID-format Exa API keys from a single pasted text blob.
func (p *ExaProvider) ParseMultiValue(credentialKey string, raw string) ([]CredentialProfile, error) {
    if credentialKey != "EXA_API_KEY" {
        return nil, fmt.Errorf("ExaProvider: unknown multi-value credential key %q", credentialKey)
    }
    keys, err := exa.ParseKeys(raw)
    if err != nil {
        return nil, err
    }
    profiles := make([]CredentialProfile, len(keys))
    for i, k := range keys {
        profiles[i] = CredentialProfile{
            ProviderID: p.ID(),
            Values:     map[string]string{credentialKey: k},
            Label:      exa.RedactKey(k),
        }
    }
    return profiles, nil
}
```

#### Step 16.3 — Update `syncToContext()` in `pkg/tui/setup_form.go`

Replace lines 143–166 (the `if sf.ctx.providerID == "exa"` block):

```go
// BEFORE
if sf.ctx.providerID == "exa" {
    rawKeys := *sf.credentialValues["exa:EXA_API_KEY"]
    keys, _ := exa.ParseKeys(rawKeys)
    for _, key := range keys {
        profiles = append(profiles, provider.CredentialProfile{
            ProviderID: "exa",
            Values:     map[string]string{"EXA_API_KEY": key},
            Label:      exa.RedactKey(key),
        })
    }
} else {
    // Generic fallback
    values := make(map[string]string)
    for _, spec := range specs {
        values[spec.Key] = *sf.credentialValues[sf.ctx.providerID+":"+spec.Key]
    }
    profiles = append(profiles, provider.CredentialProfile{
        ProviderID: sf.ctx.providerID,
        Values:     values,
        Label:      "Default",
    })
}

// AFTER
mv, isMultiValue := sf.ctx.provider.(provider.MultiValueParser)
for _, spec := range specs {
    raw := *sf.credentialValues[sf.ctx.providerID+":"+spec.Key]
    if spec.MultiValue && isMultiValue {
        parsed, err := mv.ParseMultiValue(spec.Key, raw)
        if err == nil {
            profiles = append(profiles, parsed...)
        }
        continue
    }
    // Single-value credential: one profile with all creds gathered
}
// Build single-profile for non-multi-value providers
if !isMultiValue || len(profiles) == 0 {
    values := make(map[string]string)
    label := "Default"
    for _, spec := range specs {
        val := *sf.credentialValues[sf.ctx.providerID+":"+spec.Key]
        values[spec.Key] = val
        if spec.Secret && label == "Default" && len(val) > 0 {
            label = redact.Key(val)
        }
    }
    profiles = append(profiles, provider.CredentialProfile{
        ProviderID: sf.ctx.providerID,
        Values:     values,
        Label:      label,
    })
}
```

Add `"github.com/nawodyaishan/universal-mcp-sync/pkg/redact"` to `setup_form.go` imports.
Remove `"github.com/nawodyaishan/universal-mcp-sync/pkg/exa"` from `setup_form.go` imports.

#### Acceptance

- `grep -n "pkg/exa" pkg/tui/setup_form.go` returns nothing
- `go build ./pkg/tui/...` passes
- `go test ./pkg/...` passes
- Manual test: Exa multi-key input still parses multiple UUIDs into separate profiles

---

## Appendix A — File change index

| File | Task(s) | Type |
|---|---|---|
| `pkg/provider/types.go` | T01, T16 | Modified |
| `pkg/provider/exa.go` | T16 | Modified |
| `pkg/provider/github.go` | T10 | New |
| `pkg/provider/github_test.go` | T10 | New |
| `pkg/provider/registry.go` | T11 | Modified |
| `pkg/redact/redact.go` | T02 | New |
| `pkg/redact/redact_test.go` | T02 | New |
| `pkg/client/capabilities.go` | T03, T12 | New |
| `pkg/client/adapter.go` | T04 | New |
| `pkg/client/adapter_test.go` | T05 | New |
| `pkg/client/capabilities_test.go` | T05 | New |
| `pkg/app/app.go` | T06, T08 | Modified |
| `pkg/app/app_test.go` | T08 | Modified (minor) |
| `pkg/app/qa_scenarios_test.go` | T13 | Modified |
| `pkg/verify/verify.go` | T07 | Modified |
| `pkg/verify/verify_test.go` | T07 | Modified |
| `pkg/tui/setup_form.go` | T16 | Modified |
| `pkg/tui/preview.go` | T15 | Modified |
| `cmd/usync/main.go` | T06 | Modified |
| `Makefile` | T09 | Modified |
| `.github/workflows/ci.yml` | T09 | Modified |

---

## Appendix B — Definition of done for any new provider

A provider is **complete** when:

1. `pkg/provider/<name>.go` implements `MCPProvider`
2. `pkg/provider/<name>_test.go` tests `GenerateConfig` for valid + invalid credentials
3. Provider is registered in `DefaultRegistry()` (`pkg/provider/registry.go`)
4. `pkg/client/capabilities.go` is reviewed — if the provider uses a transport not yet mapped for any client, update accordingly
5. `pkg/app/qa_scenarios_test.go` has:
   - A golden-path test applying to all supported clients
   - A skip test confirming unsupported clients are excluded with a `SkipReason`
   - A coexistence test with at least one other provider
6. `make test` passes with overall coverage ≥ 65%
7. `make lint` passes
8. PR description lists which clients are skipped and why

**Adding a provider requires zero changes to:** `pkg/app/app.go`, `pkg/verify/verify.go`, `pkg/config/*`, `pkg/tui/model.go`, `pkg/tui/assignments.go`, `pkg/tui/results.go`.
