---
title: "fix: Deduplicate config env var tags and add operations shorthand"
type: fix
status: active
date: 2026-04-09
origin: manuscripts/hubspot-pp-cli/20260408-231505/proofs/2026-04-09-hubspot-retro.md
---

# fix: Deduplicate config env var tags and add operations shorthand

## Overview

Two generator bugs surfaced during the HubSpot session that affect 20-30% of future CLIs. Both are build-blocking: they cause `go vet` failures or generation rejections. This plan addresses retro findings #1, #4, and #6 (WU-1 and WU-2).

## Problem Frame

1. **Config tag collision (WU-1, findings #1 + #6):** `config.go.tmpl` emits a hardcoded `AccessToken string ... "access_token"` field AND then loops over `Auth.EnvVars` to emit additional fields. When an env var like `HUBSPOT_ACCESS_TOKEN` produces placeholder `access_token` via `envVarPlaceholder()`, the Go struct has two fields with the same JSON tag. `go vet` rejects this, blocking the build. Additionally, when no `Config.Format` is set (common for internal YAML specs), the config file path renders as `config.` with a trailing dot.

2. **Operations shorthand (WU-2, finding #4):** The internal YAML spec format requires explicit `endpoints:` map entries for every operation. For APIs with uniform CRUD patterns (HubSpot has 14 resources, each needing list/get/create/update/delete), this means writing 70+ endpoint definitions by hand. An `operations` shorthand that auto-expands to standard CRUD endpoints would eliminate 80% of this boilerplate.

## Requirements Trace

- R1. A CLI generated with `HUBSPOT_ACCESS_TOKEN` or `DISCORD_TOKEN` env vars must pass `go vet` without manual edits
- R2. A CLI generated with `STRIPE_SECRET_KEY` (no collision) must still work identically
- R3. Config file path must never end with a trailing dot when format is unset
- R4. An internal YAML spec with `operations: [list, get, create, update, delete]` and a `path` must generate the equivalent of 5 explicit endpoints
- R5. Explicit `endpoints:` definitions must still work (no regression)
- R6. When both `operations` and `endpoints` are present, explicit endpoints override operations-generated defaults

## Scope Boundaries

- Does NOT change auth logic or env var resolution order
- Does NOT affect the OpenAPI parser (operations shorthand is internal YAML only)
- Does NOT add new operations beyond the standard CRUD set (list, get, create, update, delete, search)

## Context & Research

### Relevant Code and Patterns

- `internal/generator/templates/config.go.tmpl` lines 18-30: Struct field generation with hardcoded fields + env var loop
- `internal/generator/generator.go` lines 1039-1049: `envVarField()` converts env var names to Go field names
- `internal/generator/generator.go` lines 1331-1343: `envVarPlaceholder()` strips prefix for tag values
- `internal/spec/spec.go` lines 60-64: `Resource` struct (Endpoints map, no Operations field)
- `internal/spec/spec.go` lines 153-191: `Validate()` method, line 165 rejects resources with no endpoints
- `testdata/loops.yaml`: Reference internal YAML spec format

### Institutional Learnings

- AGENTS.md: "Default to machine changes." Both fixes are clearly machine-level.
- AGENTS.md: "Add tests for new non-trivial logic. Match the package's existing style (typically table-driven with `testify/assert`)."

## Key Technical Decisions

- **Dedup strategy for config tags:** Skip emitting the env-var-derived field when its `envVarPlaceholder` matches a hardcoded field's tag. Instead, add the `os.Getenv` call to populate the existing hardcoded field. This preserves the existing field names that `AuthHeader()`, `SaveTokens()`, and `ClearTokens()` reference. The alternative (removing hardcoded fields entirely) would require rewriting all auth methods to use dynamic field names, which is higher risk.

- **Operations expansion location:** Expand operations to endpoints in `ParseBytes()` after YAML unmarshaling but before `Validate()`. This keeps the expansion logic in one place and means validation sees fully-formed endpoints regardless of whether they came from explicit definitions or operations shorthand.

- **Config format default:** Use Go template `or` function: `{{or .Config.Format "json"}}`. Simple, zero-risk, consistent with how Go templates handle defaults.

## Open Questions

### Resolved During Planning

- **Q: Should env var fields use the hardcoded field name or a new name?** Resolution: Use the existing hardcoded field name. When `envVarPlaceholder("HUBSPOT_ACCESS_TOKEN")` = `"access_token"`, the env var populates `cfg.AccessToken` directly instead of creating a `cfg.HubspotAccessToken` that duplicates it. This avoids downstream changes to `AuthHeader()`, `SaveTokens()`, etc.

- **Q: What CRUD operations map to which HTTP methods?** Resolution: Standard REST conventions:
  - `list` -> `GET /path` (collection)
  - `get` -> `GET /path/{id}` (by ID, positional)
  - `create` -> `POST /path`
  - `update` -> `PATCH /path/{id}` (positional)
  - `delete` -> `DELETE /path/{id}` (positional)
  - `search` -> `POST /path/search`

- **Q: How should the `{id}` path param be named?** Resolution: Derive from the resource name: singular form + "Id" (e.g., resource "contacts" -> `contactId`, resource "deals" -> `dealId`). Use the existing `singularize()` helper if available, or simple trailing-s strip.

### Deferred to Implementation

- Exact singularization logic for irregular plurals (e.g., "properties" -> "property"). May need a small lookup table.
- Whether `batch_create`, `batch_update`, `batch_read` operations should be supported. Not needed for this fix; can be added later.

## Implementation Units

- [ ] **Unit 1: Deduplicate env var config fields (finding #1)**

**Goal:** Prevent duplicate JSON/TOML tags when env var placeholders collide with hardcoded field tags.

**Requirements:** R1, R2

**Dependencies:** None

**Files:**
- Modify: `internal/generator/templates/config.go.tmpl`
- Modify: `internal/generator/generator.go`
- Test: `internal/generator/generator_test.go`

**Approach:**
- Add a template helper function `envVarIsBuiltinField(envVar string) bool` that checks if `envVarPlaceholder(envVar)` matches any hardcoded config tag (`access_token`, `refresh_token`, `client_id`, `client_secret`, `base_url`, `auth_header`)
- In the struct field loop (lines 28-30), skip emitting the field when `envVarIsBuiltinField` returns true
- In the env var override loop (lines 65-70), when `envVarIsBuiltinField` is true, populate the existing hardcoded field (e.g., `cfg.AccessToken`) instead of the computed field name
- Register the new helper in the template FuncMap alongside `envVarField` and `envVarPlaceholder`

**Patterns to follow:**
- Existing template helper registration pattern in `generator.go` (line 110: `"envVarField": envVarField`)
- Existing `envVarPlaceholder` function for the comparison logic

**Test scenarios:**
- Happy path: Generate config with `STRIPE_SECRET_KEY` env var -> struct has both `AccessToken` and `StripeSecretKey` fields with unique tags, `go vet` passes
- Edge case: Generate config with `HUBSPOT_ACCESS_TOKEN` env var -> struct has `AccessToken` field only (no `HubspotAccessToken`), env var override populates `cfg.AccessToken`, `go vet` passes
- Edge case: Generate config with `DISCORD_TOKEN` env var -> if placeholder is `token`, no collision (no hardcoded `Token` field). Verify no false positive dedup
- Edge case: Generate config with two env vars, one colliding and one not -> colliding one uses builtin field, non-colliding one gets its own field
- Error path: Generate with `MY_API_CLIENT_ID` -> placeholder `client_id` collides with hardcoded `ClientID`, env var populates `cfg.ClientID` directly

**Verification:**
- `go test ./internal/generator/...` passes
- Generate a CLI with `HUBSPOT_ACCESS_TOKEN` env var, run `go vet ./...` in generated CLI -> passes

- [ ] **Unit 2: Default config format to "json" (finding #6)**

**Goal:** Prevent truncated config file path when `Config.Format` is empty.

**Requirements:** R3

**Dependencies:** None (can be done in parallel with Unit 1)

**Files:**
- Modify: `internal/generator/templates/config.go.tmpl`
- Test: `internal/generator/generator_test.go`

**Approach:**
- Replace `config.{{.Config.Format}}` with `config.{{or .Config.Format "json"}}` in the path construction (line 47)
- Also check for any other template locations that reference `.Config.Format` and apply the same default

**Patterns to follow:**
- Go template `or` function is already available in text/template

**Test scenarios:**
- Happy path: Generate with `Config.Format = "toml"` -> path ends with `config.toml`
- Edge case: Generate with `Config.Format = ""` (empty) -> path ends with `config.json`, not `config.`
- Edge case: Generate with no Config section at all -> path ends with `config.json`

**Verification:**
- Generated config.go contains `config.json` or `config.toml`, never `config.`

- [ ] **Unit 3: Add Operations field to Resource struct**

**Goal:** Allow internal YAML specs to use `operations: [list, get, create, update, delete]` as shorthand.

**Requirements:** R4, R5, R6

**Dependencies:** None (independent of Units 1-2)

**Files:**
- Modify: `internal/spec/spec.go`
- Test: `internal/spec/spec_test.go`

**Approach:**
- Add `Operations []string` field to `Resource` struct with yaml tag `operations,omitempty`
- Add an `expandOperations()` method on `APISpec` that runs after `ParseBytes()` unmarshaling
- For each resource with non-empty `Operations` and a `Path`:
  - For each operation string, generate the corresponding `Endpoint` entry
  - If the resource already has an explicit endpoint with the same key name, the explicit one wins (no override)
  - Derive the ID path param name from the resource key (e.g., "contacts" -> `contactId`)
- Call `expandOperations()` in `ParseBytes()` between unmarshal and `Validate()`

**Patterns to follow:**
- Existing `Resource` struct field pattern (line 60-64 of spec.go)
- Existing `ParseBytes()` flow (unmarshal -> return) at lines 125-151

**Test scenarios:**
- Happy path: Spec with `operations: [list, get]` on resource "items" with `path: /api/items` -> produces `list` (GET /api/items) and `get` (GET /api/items/{itemId}) endpoints
- Happy path: Spec with `operations: [list, get, create, update, delete]` -> produces all 5 standard endpoints with correct methods and paths
- Happy path: `operations: [search]` -> produces POST /path/search endpoint
- Edge case: Spec with both `operations: [list, get]` and explicit `endpoints: {list: {method: GET, path: /custom}}` -> explicit `list` wins, `get` is generated from operations
- Edge case: Spec with `endpoints:` only (no operations) -> works exactly as before (regression test)
- Edge case: Spec with `operations: [list]` but no `path` on the resource -> validation error (path is required for expansion)
- Error path: Spec with `operations: [invalid_op]` -> ignored or validation error

**Verification:**
- `go test ./internal/spec/...` passes
- Parse a YAML spec with `operations` field -> endpoints are populated correctly

- [ ] **Unit 4: Add testdata fixture for operations shorthand**

**Goal:** Provide a reference spec using the new operations shorthand for future skill authors.

**Requirements:** R4

**Dependencies:** Unit 3

**Files:**
- Create: `testdata/operations-shorthand.yaml`
- Modify: `internal/spec/spec_test.go` (add integration test using the fixture)

**Approach:**
- Create a minimal internal YAML spec using `operations` on 2-3 resources
- Include one resource with both `operations` and explicit `endpoints` to test the merge behavior
- Add a test in `spec_test.go` that parses this fixture and verifies the expanded endpoints

**Patterns to follow:**
- Existing `testdata/loops.yaml` fixture style
- Existing `TestParseBytesYAMLVariations` test pattern

**Test scenarios:**
- Integration: Parse `testdata/operations-shorthand.yaml` -> all resources have the expected number of endpoints with correct methods, paths, and param names
- Integration: Resource with both operations and explicit endpoints -> explicit endpoints preserved, operations fill gaps

**Verification:**
- `go test ./internal/spec/... -run TestOperationsShorthand` passes

## System-Wide Impact

- **Interaction graph:** Config template change affects every generated CLI's config.go. Operations shorthand affects only CLIs generated from internal YAML specs.
- **Error propagation:** No change to runtime error behavior. These are generation-time fixes.
- **Unchanged invariants:** OpenAPI parser is not touched. Auth resolution order is preserved. Existing internal YAML specs with explicit endpoints work identically.

## Risks & Dependencies

| Risk | Mitigation |
|------|------------|
| Dedup logic has false positives (skips fields that shouldn't be skipped) | Hardcoded list of builtin tags makes the match explicit, not heuristic. Test with non-colliding env vars. |
| Operations expansion produces wrong HTTP methods | Standard REST conventions are well-defined. Explicit mapping table in code, not inference. |
| Operations expansion breaks when resource name is irregular (e.g., "data") | ID param derivation uses simple singular + "Id". Add test for edge cases. Implementer may need a small lookup table. |

## Sources & References

- **Origin document:** [HubSpot Retro](manuscripts/hubspot-pp-cli/20260408-231505/proofs/2026-04-09-hubspot-retro.md)
- Config template: `internal/generator/templates/config.go.tmpl`
- Spec parser: `internal/spec/spec.go`
- Generator helpers: `internal/generator/generator.go` (envVarField, envVarPlaceholder)
- Reference spec: `testdata/loops.yaml`
