---
title: "fix: Crowd-sniff and generator improvements from Steam retro"
type: fix
status: active
date: 2026-03-30
origin: docs/retros/2026-03-30-steam-retro.md
---

# fix: Crowd-sniff and generator improvements from Steam retro

## Overview

The Steam CLI generation surfaced 5 actionable systemic issues (plus 1 skip). This plan addresses all worthwhile findings: crowd-sniff resource naming, auth detection, top-level command promotion, dead code emission, and YAML format safety. These are validated across two APIs (Postman Explore and Steam).

## Problem Frame

Crowd-sniff produces specs that crash the generator (slashes in resource names), miss auth config (every API call returns 401), and require Python post-processing that breaks Go tools. The generator still emits dead helpers and only produces nested API-path commands, requiring 7-10 manual command files per CLI. (see origin: docs/retros/2026-03-30-steam-retro.md)

## Requirements Trace

- R1. Crowd-sniff resource names never contain `/` — methods grouped under their interface/resource prefix
- R2. Crowd-sniff detects API key and bearer token auth patterns from npm SDK source code
- R3. Generator emits top-level user-friendly command aliases alongside nested API commands
- R4. Generator conditionally emits helpers based on which are actually referenced by the spec's endpoints
- R5. No regressions for standard REST APIs or existing catalog entries

## Scope Boundaries

- No changes to the profiler, OpenAPI parser, or spec parser (except YAML leniency if needed)
- No OAuth flow detection (complex multi-step auth) — only API key, bearer token, basic auth
- IPlayerService `input_json` wrapping is a Steam-only quirk, skipped
- Docs fallback for missing params (`--docs-fallback`) is deferred — significant new capability, needs separate design

## Context & Research

### Relevant Code and Patterns

- **`deriveResourceKey()`** (`internal/crowdsniff/specgen.go:114`): Returns `strings.Join(segments, "/")` — this is the slash bug. Same bug exists in `internal/websniff/specgen.go:392`.
- **`sanitizeResourceName()`** (`internal/openapi/parser.go:1760`): Existing precedent for stripping slashes from resource names. Strips `.`, `/`, `\` and trims underscores.
- **`significantSegments()`** (`internal/crowdsniff/specgen.go:125`): Filters path segments, removing params, version segments, and "api". The filtered segments become the resource key.
- **Generator resource name usage** (`internal/generator/generator.go:181`): `filepath.Join("internal", "cli", name+".go")` — a slash in `name` creates subdirectories that don't exist.
- **`helpers.go.tmpl`**: 25+ functions emitted unconditionally. `classifyDeleteError`, `firstNonEmpty`, `printOutputFiltered` are consistently dead across multiple APIs.

### Institutional Learnings

- **filepath.Join traversal** (`docs/solutions/security-issues/filepath-join-traversal-with-user-input-2026-03-29.md`): Resource names used in `filepath.Join` need traversal protection. Belt-and-suspenders: reject `/`, `\`, `..` at input AND verify resolved path.
- **Multi-source discovery** (`docs/solutions/best-practices/multi-source-api-discovery-design-2026-03-30.md`): Two-step path normalization for crowd-sniff. Param discovery infrastructure (brace scanner) already exists and can be extended for auth detection.

## Key Technical Decisions

- **Resource grouping by first significant segment**: For `/ISteamUser/GetPlayerSummaries/v2`, the resource is `ISteamUser` and the endpoint is `get_player_summaries`. For `/v1/users/{id}/posts`, the resource is `users` and the endpoint is `create_posts`. Only the first significant segment becomes the resource key — joining multiple segments with `/` is the bug we're fixing.
- **Auth detection is additive, not override**: Crowd-sniff populates auth only when patterns are detected AND the spec doesn't already have auth configured. The `--auth` flag or user-provided spec auth takes precedence.
- **Command promotion uses resource names from spec**: The generator already has the resource names (e.g., `ISteamUser`, `collections`). Promotion creates top-level commands that delegate to the nested ones. The `api` group stays as an escape hatch.
- **Dead code: conditional emission over post-processing**: Checking which helpers the spec's endpoints actually need at generation time is cheaper than a separate `polish` command. The spec tells us: no DELETE endpoints → no `classifyDeleteError`.
- **Fix websniff specgen.go too**: The same `deriveResourceKey` slash bug exists in `internal/websniff/specgen.go`. Fix both to prevent the same issue for sniffed specs.

## Open Questions

### Resolved During Planning

- **Q: Should resource key use only the first segment or allow multi-segment?** First segment only. Multi-segment keys (`users/posts`) create filesystem problems and confusing CLI structure. Sub-resources should be nested endpoints within a resource, not separate resources.
- **Q: How does the generator determine which helpers are needed?** From the spec: if there are DELETE endpoints → emit `classifyDeleteError`. If there are endpoints with `response_path` → emit `extractItemsAtPath`. The spec's endpoint methods and features determine the helper set.

### Deferred to Implementation

- **Q: Exact helper-to-spec-feature mapping** — Need to trace each helper to find which spec features trigger its use. Build this mapping during implementation.
- **Q: How to handle resource name collisions after flattening** — If two paths produce the same first-segment resource name (e.g., `/v1/users` and `/v2/users`), need a collision strategy. Likely: append version as suffix (`users_v2`).

## Implementation Units

- [ ] **Unit 1: Fix crowd-sniff resource naming in specgen.go**

  **Goal:** `deriveResourceKey()` returns a single-segment resource name (no slashes) by using only the first significant path segment as the resource key and grouping all methods under it.

  **Requirements:** R1, R5

  **Dependencies:** None

  **Files:**
  - Modify: `internal/crowdsniff/specgen.go`
  - Modify: `internal/websniff/specgen.go` (same bug)
  - Test: `internal/crowdsniff/specgen_test.go`

  **Approach:**
  - Change `deriveResourceKey()` to return `segments[0]` as the resource key (not `strings.Join(segments, "/")`)
  - The endpoint name derivation (`deriveEndpointName`) already uses the last segment, so endpoints within a resource will have distinct names
  - Handle the collision case: if two endpoints produce the same resource key + endpoint name, use `uniqueEndpointName()` (already exists)
  - Apply the same fix to `internal/websniff/specgen.go:392`

  **Patterns to follow:**
  - `sanitizeResourceName()` in `internal/openapi/parser.go:1760` — existing precedent for cleaning resource names

  **Test scenarios:**
  - Happy path: Steam-like path `/ISteamUser/GetPlayerSummaries/v2` → resource `ISteamUser`, endpoint `get_player_summaries`
  - Happy path: REST path `/v1/users/{id}/posts` → resource `users`, endpoint `create_posts` (not `users/posts`)
  - Happy path: Simple path `/v1/users` → resource `users`, endpoint `list_users` (unchanged)
  - Edge case: Path with only params and version `/v1/{id}` → resource `default` (fallback)
  - Edge case: Two paths same first segment `/v1/users` and `/v1/users/{id}/posts` → both in `users` resource
  - Negative test: No resource key ever contains `/` — assert across all test cases

  **Verification:**
  - `go build ./...` and `go test ./internal/crowdsniff/...` pass
  - Run `printing-press crowd-sniff --api steam` → spec resource names have no slashes
  - Run `printing-press generate --spec <crowd-sniff-spec>` → no directory creation errors

- [ ] **Unit 2: Add auth pattern detection to crowd-sniff**

  **Goal:** Crowd-sniff detects API key and bearer token auth patterns from npm SDK source code and populates the spec's auth section.

  **Requirements:** R2, R5

  **Dependencies:** None (independent of Unit 1)

  **Files:**
  - Modify: `internal/crowdsniff/npm.go`
  - Modify: `internal/crowdsniff/types.go`
  - Modify: `internal/crowdsniff/aggregate.go`
  - Modify: `internal/crowdsniff/specgen.go`
  - Test: `internal/crowdsniff/npm_test.go`
  - Test: `internal/crowdsniff/specgen_test.go`

  **Approach:**
  - Add `DiscoveredAuth` type to `types.go`: `{Type, Header, In, Format, EnvVarHint string}`
  - In `npm.go`, after extracting endpoints and params from SDK source, scan for auth patterns:
    - Constructor: `new Client(apiKey)`, `new SteamAPI(key)` → detect key param name
    - Header: `headers['Authorization'] = 'Bearer ' + token` → `type: bearer_token`
    - Header: `headers['X-Api-Key'] = key` → `type: api_key, header: X-Api-Key, in: header`
    - Query: `params.key = this.apiKey` → `type: api_key, header: key, in: query`
  - Add `Auth []DiscoveredAuth` to `SourceResult` and merge in `aggregate.go` (highest-tier source wins)
  - In `specgen.go`, populate `spec.Auth` from aggregated auth when no auth is already configured

  **Patterns to follow:**
  - Param discovery in `npm.go` / `params.go` — same pattern of scanning SDK source with regex + brace scanner
  - `DiscoveredParam` type structure — follow same pattern for `DiscoveredAuth`

  **Test scenarios:**
  - Happy path: SDK with `params.key = this.apiKey` → auth detected as `api_key, in: query, header: key`
  - Happy path: SDK with `Authorization: Bearer ${token}` → auth detected as `bearer_token`
  - Happy path: SDK with `headers['X-Api-Key'] = apiKey` → auth detected as `api_key, in: header, header: X-Api-Key`
  - Edge case: SDK with no auth patterns → `auth.type: none` (unchanged)
  - Edge case: Multiple SDKs with conflicting auth → highest-tier source wins
  - Negative test: Existing auth config in spec → not overridden by crowd-sniff detection

  **Verification:**
  - `go test ./internal/crowdsniff/...` passes
  - Run `printing-press crowd-sniff --api steam` → spec has `auth.type: api_key`, `auth.in: query`

- [ ] **Unit 3: Add top-level command promotion to generator**

  **Goal:** Generator emits user-friendly top-level commands (e.g., `player`, `games`, `search`) alongside the nested API commands (e.g., `ISteamUser get_player_summaries`). The nested commands stay as an escape hatch.

  **Requirements:** R3, R5

  **Dependencies:** None (independent)

  **Files:**
  - Modify: `internal/generator/generator.go`
  - Create: `internal/generator/templates/command_promoted.go.tmpl`
  - Modify: `internal/generator/templates/root.go.tmpl`
  - Test: `internal/generator/generator_test.go`

  **Approach:**
  - After generating the nested API commands, add a promotion pass
  - For each spec resource with a "list" endpoint (GET without path params), emit a top-level command named after the resource (pluralized or as-is depending on conventions)
  - The promoted command delegates to the nested command's handler — it's a thin wrapper with better UX (vanity name, clearer help text)
  - Resources with entity-type enum params (like Postman's `entityType=collection`) get one promoted command per enum value
  - The `command_promoted.go.tmpl` template receives the resource name, the target endpoint, and the promoted command name
  - Root template registers both the nested group and the promoted commands

  **Patterns to follow:**
  - Existing command generation in `generator.go` lines 166-245 — same data structs, same template rendering
  - How the postman-explore CLI's manual `cmd_browse.go` works — that's what the promoted command should look like

  **Test scenarios:**
  - Happy path: Spec with `users` resource having list/get/create endpoints → top-level `users` command emitted alongside nested `users list-users`
  - Happy path: Spec with `ISteamUser` resource → top-level `ISteamUser` command (or promoted as `steam-user`)
  - Edge case: Resource with no list endpoint → no promoted command
  - Edge case: Promoted command name collides with a built-in command (e.g., `version`, `help`) → skip promotion for that resource
  - Negative test: Existing commands (doctor, auth, sync, search) not affected by promotion

  **Verification:**
  - `go build ./...` and `go test ./internal/generator/...` pass
  - Generated CLI has both `ISteamUser get_player_summaries` AND a top-level shortcut

- [ ] **Unit 4: Conditional helper emission in generator**

  **Goal:** `helpers.go.tmpl` only emits helper functions that the spec's endpoints actually need, eliminating dead code at generation time.

  **Requirements:** R4, R5

  **Dependencies:** None (independent)

  **Files:**
  - Modify: `internal/generator/templates/helpers.go.tmpl`
  - Modify: `internal/generator/generator.go` (pass helper-selection flags to template)
  - Test: `internal/generator/generator_test.go`

  **Approach:**
  - Analyze the spec at generation time to determine which helpers are needed:
    - Has DELETE endpoints → emit `classifyDeleteError`
    - Has endpoints (always) → emit `classifyAPIError`, `usageErr`, `notFoundErr`, `authErr`, `apiErr`, `rateLimitErr`
    - Has list endpoints → emit `paginatedGet`
    - Has any output → emit `printOutput`, `printOutputWithFlags`, `filterFields`, `compactFields`, `printCSV`, `wantsHumanTable`, `printAutoTable`
    - Has text fields → emit `truncate`
    - Has pagination → emit `formatCompact` (used in progress reporting)
  - Functions that are always needed (error types, output formatting, tab writer, color utilities) stay unconditional
  - Functions gated by spec features: `classifyDeleteError` (DELETE), `firstNonEmpty` (never used — remove entirely), `printOutputFiltered` (never used — remove entirely)
  - Pass a `HelperFlags` struct to the template: `HasDelete bool`, etc.

  **Patterns to follow:**
  - VisionSet conditional logic in `generator.go` — same pattern of checking spec features to decide what to emit
  - `{{if .VisionSet.Store}}` in root.go.tmpl — same conditional template pattern

  **Test scenarios:**
  - Happy path: Spec with no DELETE endpoints → generated helpers.go does not contain `classifyDeleteError`
  - Happy path: Spec with DELETE endpoints → generated helpers.go contains `classifyDeleteError`
  - Happy path: Any spec → `firstNonEmpty` and `printOutputFiltered` never emitted (dead code across all tested APIs)
  - Edge case: Minimal spec (1 GET endpoint) → only essential helpers emitted
  - Negative test: Spec with full feature set → all helpers present (no regression)

  **Verification:**
  - `go build ./...` and `go test ./internal/generator/...` pass
  - Generate a CLI with no DELETE endpoints → `grep classifyDeleteError` finds nothing
  - Scorecard dead_code improves from 0/5

- [ ] **Unit 5: YAML format safety net for spec parser**

  **Goal:** Spec parser tolerates common YAML style variations so specs from non-Go tools (Python yaml.dump) don't break dogfood/verify.

  **Requirements:** R5

  **Dependencies:** Unit 1 (if resource naming is fixed, Python restructuring is no longer needed — but this is a safety net)

  **Files:**
  - Modify: `internal/spec/reader.go`
  - Test: `internal/spec/spec_test.go`

  **Approach:**
  - The spec parser currently validates strictly. Python's `yaml.dump` produces valid YAML that Go's strict parser rejects (possibly due to indentation style, string quoting, or flow vs block style differences)
  - Add a pre-parse normalization step or relax validation to accept common YAML variations
  - This is a safety net — with Unit 1 fixed, Python restructuring should be unnecessary. But specs from other tools (JS, Python, manual editing) should work too.

  **Patterns to follow:**
  - Current validation in `reader.go` — understand what specifically fails before deciding the fix

  **Execution note:** Start by reproducing the failure with a Python-generated YAML to identify what specifically the parser rejects.

  **Test scenarios:**
  - Happy path: Python yaml.dump output with 2-space indent → parses correctly
  - Happy path: Go-generated YAML → still parses correctly (no regression)
  - Edge case: YAML with quoted string values where Go would use unquoted → parses
  - Edge case: YAML with flow-style arrays `[a, b]` → parses

  **Verification:**
  - `go test ./internal/spec/...` passes
  - `printing-press dogfood --spec <python-generated-yaml>` no longer fails with "at least one resource is required"

## System-Wide Impact

- **Crowd-sniff changes (Units 1-2)** affect every crowd-sniff-generated spec. The resource naming fix is universal; auth detection is additive.
- **Generator changes (Units 3-4)** affect every generated CLI. Command promotion is additive (doesn't remove existing commands). Helper conditional emission reduces generated code size.
- **Spec parser changes (Unit 5)** affect all spec consumers (generator, dogfood, verify, scorecard). Must not break any existing specs.
- **websniff parity**: Unit 1 also fixes `internal/websniff/specgen.go` which has the same slash bug.
- **Unchanged invariants**: The internal YAML spec format, the generator's template rendering pipeline, the profiler, and the OpenAPI parser are unchanged.

## Risks & Dependencies

| Risk | Mitigation |
|------|------------|
| Resource name flattening changes existing crowd-sniff specs | Specs are regenerated each run — no backward compatibility concern |
| Auth detection produces false positives | Only populate auth when confidence is high (pattern seen in 2+ sources). Never override existing auth. |
| Command promotion names collide with built-in commands | Skip promotion for resource names matching built-in commands (doctor, auth, sync, search, version, help, etc.) |
| Helper conditional emission accidentally removes a needed helper | Test with full-featured specs (Stripe, Stytch) to verify no regressions |
| YAML leniency introduces security risk (traversal in parsed values) | The leniency is about formatting, not content. Resource names still go through `sanitizeResourceName`. |

## Sources & References

- **Origin document:** [docs/retros/2026-03-30-steam-retro.md](docs/retros/2026-03-30-steam-retro.md)
- **Prior retro:** [docs/retros/2026-03-30-postman-explore-retro.md](docs/retros/2026-03-30-postman-explore-retro.md) (findings #1, #4, #7 confirmed on second API)
- Crowd-sniff specgen: `internal/crowdsniff/specgen.go`
- Generator resource usage: `internal/generator/generator.go:166-245`
- sanitizeResourceName precedent: `internal/openapi/parser.go:1760`
- filepath-join traversal learning: `docs/solutions/security-issues/filepath-join-traversal-with-user-input-2026-03-29.md`
- Param discovery gap doc: `manuscripts/postman-explore/20260330-105847/proofs/2026-03-30-crowd-sniff-param-discovery-gap.md`
