# Phase 2 Implementation Plan: Provider Registry and Dynamic Wizard

## Objective

Phase 2 turns the Phase 1 provider abstraction into a usable product surface. Phase 1 introduced `pkg/provider.MCPProvider`, `provider.MCPConfig`, and provider-aware config mutators, but Exa is still hardwired through `app.Manager.Prepare`, CLI key flags, and the TUI setup form. This phase introduces a provider registry, provider selection, dynamic credential collection, and provider-aware planning while preserving the existing Exa behavior.

The goal is not to add many new MCP providers yet. The goal is to make Exa run through the same path future providers will use, with tests proving the current Exa output stays compatible.

## Research Baseline

- Exa's current MCP docs list `https://mcp.exa.ai/mcp` as the remote server URL, `claude mcp add --transport http`, Gemini `httpUrl`, Antigravity `serverUrl`, and the active tool set `web_search_exa`, `web_fetch_exa`, and optional `web_search_advanced_exa`.
- Charmbracelet `huh` forms are Bubble Tea models, so the existing router/sub-model approach can continue to delegate `Init`, `Update`, and `View` to a nested form instead of introducing a separate interaction framework.
- Bubble Tea's model contract still favors explicit state routing through `Init`, `Update`, `View`, and batched commands, matching the current `pkg/tui.Model` shape.
- Gemini CLI docs define `mcpServers.<name>` entries with `command`, `args`, `env`, `url`, and `httpUrl`, with `httpUrl` taking precedence for streamable HTTP.
- Claude MCP docs distinguish stdio servers from HTTP/SSE servers and show HTTP configs as `type: "http"` with a `url` field.

Sources:

- Exa MCP docs: https://docs.exa.ai/docs/reference/exa-mcp
- Huh Bubble Tea integration: https://github.com/charmbracelet/huh/blob/main/README.md
- Bubble Tea model contract: https://github.com/charmbracelet/bubbletea/blob/main/tea.go
- Gemini CLI configuration: https://google-gemini.github.io/gemini-cli/docs/get-started/configuration.html
- Claude MCP docs: https://docs.anthropic.com/en/docs/claude-code/sdk/sdk-mcp

## Current Phase 1 State

Implemented:

- `pkg/provider/types.go` defines `TransportType`, `MCPConfig`, and `MCPProvider`.
- `pkg/provider/exa.go` implements `ExaProvider`.
- JSON config writers accept `providerID` and `provider.MCPConfig`.
- Codex TOML writer accepts `providerID` and `provider.MCPConfig` for HTTP configs.
- `app.Operation` carries `ProviderID` and `Config`.
- TUI has a router/sub-model structure using `setupForm`, `assignmentModel`, `previewModel`, and `resultsModel`.

Still Exa-specific:

- `app.Manager.Prepare` accepts `[]string` keys and instantiates `provider.NewExaProvider()` internally.
- `Operation.Key` stores raw Exa key material and `FormatPlan` uses Exa-specific redaction.
- CLI flags are named `--keys` and `--keys-file`, and non-interactive mode always means Exa.
- `setupForm` renders hardcoded "Exa API Keys" copy and validates through `exa.ParseKeys`.
- Verification still assumes provider ID `exa` and Exa URL shape.
- No provider registry exists, so the TUI has no source of provider choices.

## Scope

In scope:

- Add a provider registry with Exa registered as the first provider.
- Replace unordered credential prompts with ordered credential metadata suitable for dynamic forms.
- Add provider selection to the TUI setup flow.
- Replace the hardcoded Exa key textarea with provider-generated credential fields.
- Refactor app planning so `Manager` receives provider and credential profiles instead of creating Exa internally.
- Keep compatibility wrappers for current Exa CLI flags and tests.
- Ensure user-facing output remains redacted and does not show full keys, full Exa URLs, or secret-bearing env values.

Out of scope:

- Adding GitHub, Context7, filesystem, or other new MCP providers.
- Rebranding the binary.
- Solving stdio support for every target app.
- Replacing the apply/rollback engine.
- OAuth flows or hosted connector setup.

## Architecture Decisions

### 1. Registry Is Explicit and Static

Add `pkg/provider/registry.go` with a small `Registry` type:

```go
type Registry struct {
    providers map[string]MCPProvider
    order     []string
}
```

Expose:

- `DefaultRegistry() Registry`
- `All() []MCPProvider`
- `Get(id string) (MCPProvider, bool)`

Rationale: static registration is enough for this CLI, simple to test, and avoids plugin loading before the product has multiple provider implementations.

### 2. Credential Metadata Must Be Ordered

Replace or supplement `RequiredCredentials() map[string]string` with ordered specs:

```go
type CredentialSpec struct {
    Key         string
    Label       string
    Description string
    Secret      bool
    MultiValue  bool
    Validator   CredentialValidator
}

type CredentialValidator func(string) error
```

Preferred interface shape:

```go
type MCPProvider interface {
    ID() string
    Name() string
    Description() string
    RequiredCredentials() []CredentialSpec
    GenerateConfig(credentials map[string]string) (MCPConfig, error)
}
```

Rationale: a map cannot preserve prompt order, cannot mark password fields, and cannot describe multi-value credentials like Exa's multiple keys. Exa can expose one `EXA_API_KEY` spec with `Secret: true` and `MultiValue: true`.

### 3. Planning Uses Credential Profiles

Introduce:

```go
type CredentialProfile struct {
    ProviderID string
    Values     map[string]string
    Label      string
}
```

Refactor `Manager` planning around:

```go
func (m *Manager) PrepareProvider(
    prov provider.MCPProvider,
    profiles []provider.CredentialProfile,
    selected map[config.AppID]bool,
    assignments map[config.AppID]int,
) (ExecutionPlan, error)
```

Keep the existing Exa wrapper:

```go
func (m *Manager) Prepare(keys []string, selected map[config.AppID]bool, assignments map[config.AppID]int) (ExecutionPlan, error)
```

Rationale: this keeps non-interactive Exa behavior stable while moving the real engine to provider-driven inputs.

### 4. Operations Store Labels, Not Secrets

Replace `Operation.Key string` with:

```go
CredentialLabel string
```

`PrepareProvider` should keep raw credential values only long enough to call `GenerateConfig`. `Operation.Config` may still contain secret-bearing URLs or env values because it is the generated payload, so all formatting, logging, and errors must continue to pass through redaction.

Rationale: `Operation.Key` creates unnecessary long-lived raw secret storage and is Exa-specific.

## TUI Flow

The Phase 2 TUI remains a Bubble Tea router, with `huh.Form` used only for structured setup input.

Stages:

1. Provider setup
   - Select provider from `provider.Registry`.
   - Initially this list contains Exa only, but the UI must render from registry data.

2. Credential collection
   - Render fields from `provider.CredentialSpec`.
   - For Exa multi-key input, preserve the current multiline paste behavior.
   - Password-style single-value credentials should use hidden input once added.
   - If credentials were loaded from flags or file, show only redacted labels and do not seed raw values into visible fields.

3. Target app selection
   - Keep the current app multi-select, but copy should be provider-neutral.
   - Default selected apps remain all detected apps.

4. Assignment
   - Continue assigning credential profiles across apps.
   - For one profile, auto-assign all selected apps.
   - For multiple Exa keys, preserve current default distribution.

5. Preview
   - Render provider name, target app, config transport, credential label, backup path, and warnings.
   - Do not render raw URLs, env values, keys, or tokens.

6. Results
   - Reuse existing apply result view.
   - Verification wording should be provider-aware where possible.

## Technical Breakdown

### Provider Package

- Add `CredentialSpec`, `CredentialProfile`, and `Registry`.
- Update `ExaProvider.RequiredCredentials` to return an ordered slice.
- Move Exa credential validation behind the provider spec while reusing `exa.ParseKeys` for multi-key input.
- Add helper methods for redacted labels:
  - Exa should label UUID keys with `exa.RedactKey`.
  - Future providers can use a generic secret redactor.

### App Package

- Add `PrepareProvider`.
- Keep `Prepare` as an Exa compatibility wrapper.
- Remove raw `Operation.Key`.
- Update `FormatPlan` to use `Operation.CredentialLabel`.
- Replace hardcoded verification command provider name where possible:
  - File verification can remain Exa-specific until Phase 3 if it only validates Exa URL semantics.
  - CLI command arguments should use `op.ProviderID` instead of literal `exa`.
- Keep `applyClaudeCode` provider-aware in command labels and error messages.

### TUI Package

- Extend `wizardContext`:

```go
registry    provider.Registry
providerID  string
provider    provider.MCPProvider
profiles    []provider.CredentialProfile
```

- Replace `keys []string` with `profiles` in assignment and preview paths.
- Keep Exa-specific parsing isolated in provider helper code, not in `setup_form.go`.
- Rename setup labels from "Exa API Keys" to provider-derived labels.
- Preserve the current security behavior where flag/file-loaded secrets are not displayed in the form.

### CLI Entrypoint

- Keep existing Exa flags for backward compatibility:
  - `--keys`
  - `--keys-file`
- Add optional provider selection only if it does not complicate Phase 2:
  - `--provider exa` defaulting to `exa`
- Non-interactive mode can reject non-Exa providers until Phase 3 adds provider-specific credential flag conventions.

### Verification

- Keep current Exa URL verification for Exa.
- Add a provider-aware wrapper:

```go
func VerifyProviderFile(path string, kind config.FileKind, providerID string, cfg provider.MCPConfig) Result
```

- In Phase 2 this can dispatch Exa to existing URL inspection and return a clear unsupported warning for any future provider until its verifier exists.

## Implementation Tasks

### Task 1: Provider Registry and Credential Specs

- Add registry and ordered credential spec types.
- Update Exa provider to use the new credential spec.
- Add unit tests for registry lookup, provider order, and Exa credential metadata.

### Task 2: Provider-Aware Planning

- Add `PrepareProvider`.
- Keep `Prepare` as an Exa wrapper.
- Remove raw key storage from `Operation`.
- Update CLI args, plan formatting, and apply labels to use provider ID.
- Add tests proving Exa plans match current target paths, provider ID, transport, and redacted labels.

### Task 3: Dynamic Setup Form

- Update `wizardContext` for registry/provider/profiles.
- Add provider select field sourced from registry.
- Generate credential fields from the selected provider.
- Preserve Exa multiline key input and hidden loaded-key behavior.
- Update setup form tests for provider selection and credential sync.

### Task 4: Assignment and Preview Refactor

- Move assignment logic from `keys` to credential profiles.
- Preserve current Exa default assignment behavior.
- Update preview output to show provider-neutral config summaries.
- Add tests for one profile, two Exa profiles, and selected target validation.

### Task 5: Verification and Compatibility

- Add provider-aware verification wrapper.
- Replace literal `exa` in CLI verification calls where the provider ID is available.
- Keep existing Exa verification behavior unchanged.
- Run full regression checks.

## Test Plan

Unit tests:

- Registry returns Exa and rejects unknown providers.
- Exa credential spec is ordered, secret, multi-value, and validates UUID-style keys.
- `PrepareProvider` creates the same Exa HTTP configs as current `Prepare`.
- `Operation` and formatted plan do not include raw UUID keys.
- TUI setup with loaded Exa keys does not render full keys.
- TUI setup without loaded keys parses multiline Exa input into credential profiles.
- Assignment defaults match existing Exa behavior for one, two, and three keys.
- Gemini config still writes `httpUrl`; Antigravity still writes `serverUrl`; Codex still writes TOML URL.

Regression commands:

```bash
make fmt
make test
make build
make gitignore-check
```

Manual checks:

- `make dry-run KEYS_FILE=~/Downloads/exa_keys.txt`
- TUI launch with no flags, paste two Exa keys, preview, and cancel.
- TUI launch with `--keys-file`, confirm full keys are not displayed.
- Fixture-home apply for Exa and verify rollback behavior remains intact.

## Acceptance Criteria

- Exa is selected through the provider registry, not by hardcoded construction in `Manager.PrepareProvider`.
- The existing `Manager.Prepare` and CLI Exa paths continue to work.
- The TUI setup form is provider-driven and contains no hardcoded Exa field labels outside Exa provider metadata.
- Multiple Exa keys still distribute across selected apps as before.
- No full Exa API key or full secret-bearing Exa URL appears in TUI, dry-run output, apply output, verification summaries, errors, or logs.
- Existing Exa config output remains compatible with current docs: Claude Code HTTP transport, Gemini `httpUrl`, Antigravity `serverUrl`, and the three active Exa tools.
- Full regression checks pass.

## Risks and Mitigations

- Risk: changing the provider interface creates churn across tests.
  - Mitigation: update Exa first, keep `Prepare` as compatibility wrapper, and avoid adding new providers in this phase.

- Risk: dynamic `huh` forms are harder to test than direct model state.
  - Mitigation: isolate form construction and context sync into small functions with table tests.

- Risk: provider-neutral output accidentally leaks values from `MCPConfig.Env` or URL query strings.
  - Mitigation: route every rendered config summary through redaction and test against UUID-style keys, `exaApiKey`, and generic token-like env values.

- Risk: verification remains Exa-specific while the UI becomes provider-neutral.
  - Mitigation: explicitly scope Phase 2 verification to Exa and return unsupported warnings for future providers until Phase 3.

## Phase 3 Handoff

After Phase 2, adding the first non-Exa provider should require:

- creating a provider implementation,
- registering it,
- adding credential validators,
- adding capability rules for target apps,
- adding provider-specific verification.

The expected Phase 3 provider should be a stdio provider because it will exercise the already-generic `MCPConfig.Command`, `Args`, and `Env` paths.
