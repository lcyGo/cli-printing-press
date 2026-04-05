---
title: "feat: MCP Readiness Layer — Per-Endpoint Auth Awareness and Public Library MCP Catalog"
type: feat
status: active
date: 2026-04-05
---

# feat: MCP Readiness Layer — Per-Endpoint Auth Awareness and Public Library MCP Catalog

## Overview

Add per-endpoint auth awareness to MCP server generation and surface MCP metadata in the public library registry. The printing press already generates companion MCP servers alongside CLIs, but today there is zero signaling about which tools require auth and which work without it. This means: (1) users can't discover MCP servers from the library, (2) agents calling MCP tools get no hint about auth requirements until a 401, and (3) the library misses the MCP marketplace distribution channel entirely.

This plan covers three layers: machine changes in this repo (generator, parser, templates, manifest, publish), public library repo changes (registry schema, README), and targeted fixups to the 6 existing published CLIs (without regeneration).

**Target repos:**
- **Primary:** `cli-printing-press` (this repo) — all machine changes
- **Secondary:** `mvanhorn/printing-press-library` — registry, README, and existing CLI fixups

## Problem Frame

The `fli` project (Google Flights CLI) grew to 1,400 GitHub stars and 12K PyPI downloads/month primarily through MCP marketplace listings — Smithery, LobeHub, MCPMarket, etc. The printing press library has 6 published CLIs, all with MCP binaries, but zero marketplace presence and no MCP metadata in the registry.

Even cookie-auth CLIs like Pagliacci Pizza have public endpoints (store finder, menus) that work as MCP tools without any auth. But today, the MCP tool descriptions don't say this, the registry doesn't surface it, and users have no way to know which tools work out of the box.

## Requirements Trace

- R1. Per-endpoint `NoAuth` detection from OpenAPI specs (`security: []` override)
- R2. MCP tool descriptions annotated with "(no auth required)" for public endpoints
- R3. CLI manifest extended with MCP metadata (binary name, tool counts, readiness level)
- R4. README template includes MCP server install section
- R5. Publish pipeline generates `smithery.yaml` marketplace metadata
- R6. Public library `registry.json` extended with MCP fields
- R7. Existing 6 published CLIs updated with auth annotations (no regeneration)
- R8. Scorecard reports public/auth tool split (informational, not new scored dimension)

## Scope Boundaries

- **In scope:** OpenAPI-sourced specs only. Sniffed/internal specs get `NoAuth` support later.
- **In scope:** Informational scorecard reporting of public tool counts. NOT a new scored dimension (avoids tier constant changes).
- **Out of scope:** Auto-submission to MCP marketplaces. We generate metadata files; humans decide whether to list.
- **Out of scope:** MCP OAuth2 native auth (MCP spec auth RFC). Worth monitoring; not building against today.
- **Out of scope:** Auto-refresh of cookies from MCP context. Separate feature.
- **Out of scope:** Changes to the MCP server binary structure or transport (stdio is correct).

## Context & Research

### Relevant Code and Patterns

- **Scorecard already parses per-operation security:** `parseSecurityRequirementSet()` at `internal/pipeline/scorecard.go:1113-1157` correctly handles `security: []` (empty array → `AllowsAnonymous = true`) and `security: [{}]` (empty object → anonymous). This is reference code for the parser change.
- **OpenAPI parser endpoint construction:** `mapResources()` at `internal/openapi/parser.go:800-851` iterates operations via `pathItem.Operations()`. Each `op` is `*openapi3.Operation` with `Security *SecurityRequirements` field. Currently ignored.
- **kin-openapi types:** `SecurityRequirements` is `[]SecurityRequirement`. When spec says `security: []`, kin-openapi sets `op.Security` to non-nil pointer to empty slice. When no per-operation security is declared, `op.Security` is nil (inherits global).
- **Template conditionals:** Existing pattern `{{- if and .Auth.Type (ne .Auth.Type "none")}}` in `mcp_tools.go.tmpl` shows how auth-conditional rendering works.
- **Endpoint.Meta:** Exists as `map[string]string` on Endpoint but is only used by crowd-sniff for source provenance. A dedicated `NoAuth bool` field is cleaner and matches the pattern of `Required`, `Positional` on `Param`.
- **CLIManifest:** Schema version 1, clean struct at `internal/pipeline/climanifest.go:27-41`. Adding fields with `omitempty` is backward-compatible.
- **Publish skill registry schema:** Documented in `skills/printing-press-publish/SKILL.md` lines 564-579.

### Institutional Learnings

- **Scorecard update is mandatory when adding capabilities** (AGENTS.md + `docs/solutions/logic-errors/scorecard-accuracy-broadened-pattern-matching`). For this feature, we add informational reporting only — no new scored dimension — so tier constants stay unchanged.
- **Traversal protection for spec-derived strings** (`docs/solutions/security-issues/filepath-join-traversal-with-user-input`). MCP binary names derived from API slugs flow through existing `naming.CLI()` which is already safe. Smithery.yaml values are written to known paths, not user-derived paths.
- **Validation must not mutate source directory** (`docs/solutions/best-practices/validation-must-not-mutate-source-directory`). The smithery.yaml is written during `publish package` (which copies to staging), not during validation.
- **Layout contract** (`docs/solutions/best-practices/checkout-scoped-printing-press-output-layout`). MCP metadata extends `.printing-press.json` (the provenance manifest), not a separate file. Smithery.yaml is a new file alongside it in the published CLI directory.

## Key Technical Decisions

- **Dedicated `NoAuth bool` field over `Meta["no_auth"]`:** Type-safe, template-friendly (`{{if .NoAuth}}`), matches existing boolean field patterns on `Param`. Serialized as `no_auth,omitempty` to stay backward-compatible with existing YAML specs.
- **Detection logic: explicit override only, not inheritance:** Only set `NoAuth = true` when `op.Security` is non-nil AND empty. When `op.Security` is nil, the operation inherits global auth — which could be anything. This avoids false positives where a spec has no global security but individual operations aren't explicitly marked.
- **Post-parse sweep for no-auth specs, guarded by `!Auth.Inferred`:** When `Auth.Type == "none"` AND `!Auth.Inferred`, mark all endpoints `NoAuth = true`. The `Inferred` guard prevents false positives where `mapAuth()` failed to detect auth (returns `"none"` by default) but the API actually requires it. If auth was inferred from description keywords (`Inferred: true`) but resolved to `"none"`, that means the inference found nothing — but we can't be certain the API is truly public, so we don't sweep.
- **MCP binary naming: `{name}-pp-mcp` with `-pp-` infix:** Consistent with the CLI's `{name}-pp-cli` collision avoidance rationale. If the infix was worth doing for CLIs, it's worth doing for MCPs — especially since MCP marketplaces are where vendor collisions are most likely. The generator currently produces `{name}-mcp` (no infix) and all 6 published CLIs use that convention. This plan renames to `{name}-pp-mcp` everywhere: generator template, naming function, published CLI directories, goreleaser configs, smithery listings, and registry entries. The rename is cheapest now — before any marketplace listings exist and before users have wired `claude mcp add` configs.
- **`naming.MCP()` function:** Add to `internal/naming/naming.go` returning `name + "-pp-mcp"`. Update the generator at `generator.go:441` to use this function instead of the current inline `name + "-mcp"`. Centralizes the convention and applies the `-pp-` infix consistently.
- **`oneline()` truncation and the auth annotation:** The `oneline()` template function truncates descriptions at 120 characters. Naively appending `(no auth required)` after `oneline` would push long descriptions past the limit or get cut. Instead, register a new template function `mcpDescription(desc string, noAuth bool)` that appends the annotation *before* truncation, ensuring it's part of the truncated output rather than lost. If the description is already long, the suffix takes priority over trailing description text.
- **`GenerateManifestParams` needs the parsed spec:** `WriteManifestForGenerate()` currently takes only `APIName`, `SpecSrcs`, `DocsURL`, `OutputDir` — no access to the parsed `*spec.APISpec` for tool counting. Add a `Spec *spec.APISpec` field to `GenerateManifestParams` and thread it through both callers in `internal/cli/root.go` (lines 189 and 331). Both callers already have the parsed spec in scope (`parsed` and `apiSpec` respectively). For `writeCLIManifestForPublish()`, which operates on `PipelineState`, re-parse the spec from the output directory's `spec.json` — the file is always present at publish time.
- **`AuthEnvVars` in manifest:** Add `AuthEnvVars []string` to `CLIManifest` alongside `AuthType`. Without this, the publish skill cannot populate `env_vars` in the registry.json `mcp` block — it would have to parse CLI source code. The manifest is the right place to carry this data from generation to publish.
- **Informational scorecard reporting, not a new dimension:** Adding a scored MCP dimension would require tier constant changes, unscored handling for non-MCP CLIs, and broader calibration. Instead, the scorecard's summary output gets a one-line note: "MCP: 42 tools (15 public, 27 auth-required)". This satisfies the AGENTS.md rule (scorer reflects capability) without inflating scores.
- **Scorecard accesses NoAuth via `openapi.Parse()`:** The scorecard currently uses raw `map[string]any` parsing via `loadOpenAPISpec()`. Rather than duplicating the `security: []` detection in a second code path, the scorecard should call `openapi.Parse()` to get structured `spec.Endpoint` objects with `NoAuth` flags. This adds a dependency on the full parser but avoids logic divergence. The scorecard already imports the openapi package (`internal/openapi`) for `IsOpenAPI()`.
- **Smithery.yaml generated at publish time, not generate time:** The smithery file needs category, description, and auth metadata that may be enriched during publish. It lives alongside `.printing-press.json` in the published directory.
- **Registry.json `mcp` block is additive:** New fields are optional. Existing registry entries without `mcp` remain valid. The publish skill adds `mcp` when packaging.
- **Smithery.yaml `env` semantics for partial-readiness CLIs:** For cookie/composed CLIs, env vars are marked `required: false` with a description noting "required for authenticated endpoints only — some tools work without credentials." This accurately reflects MCP server startup behavior (server starts fine without cookies) while signaling that not all tools will work.

## Open Questions

### Resolved During Planning

- **Should `NoAuth` go on Endpoint or AuthConfig?** → Endpoint. Auth config is spec-level. NoAuth is per-endpoint. These are different granularities.
- **Should the OpenAPI parser also detect when the global security array is empty?** → Yes, handled via post-parse sweep. Guarded by `Auth.Type == "none" && !Auth.Inferred` to avoid false positives from specs that simply omit security declarations.
- **What about operations that have security but also allow anonymous (`security: [{}, {"api_key": []}]`)?** → Set `NoAuth = true`. If anonymous is one of the alternatives, the tool works without auth. The agent can always provide auth for better results.
- **How does `WriteManifestForGenerate` get spec data for tool counts?** → Add `Spec *spec.APISpec` to `GenerateManifestParams`. Both callers in `root.go` already have the parsed spec in scope.
- **Should MCP binaries use `-pp-` infix?** → Yes. `{name}-pp-mcp` everywhere — binary, marketplace listing, registry, naming function. Consistent with CLI's `{name}-pp-cli` collision avoidance. Cheapest to rename now before any marketplace listings exist. The 6 published CLIs get renamed in the same library repo PR as the auth annotations (Unit 9).
- **How does the scorecard access NoAuth data?** → Call `openapi.Parse()` to get structured endpoints with `NoAuth` flags. Avoids duplicating detection logic in the raw-parse path.
- **Where do env var names come from in the registry?** → Add `AuthEnvVars []string` to `CLIManifest`, populated from `spec.Auth.EnvVars` at generation time. Publish skill reads from manifest.
- **How to handle `oneline()` truncation with auth annotation?** → New `mcpDescription()` template function that appends annotation before truncation, not after.

### Deferred to Implementation

- **Exact smithery.yaml field names** — will be confirmed against current Smithery docs during implementation.
- **Which Pagliacci/Steam endpoints are actually public** — determined during Unit 9 by making unauthenticated HTTP requests to each candidate endpoint. If 200 with meaningful data → public. If 401/403 → auth-required. Results documented in the PR.

## Implementation Units

### This Repo (cli-printing-press)

- [ ] **Unit 1: Add `NoAuth` field to Endpoint spec**

  **Goal:** Extend the spec model so endpoints can declare whether they require auth.

  **Requirements:** R1

  **Dependencies:** None

  **Files:**
  - Modify: `internal/spec/spec.go`
  - Modify: `internal/spec/spec_test.go`

  **Approach:**
  Add `NoAuth bool` to `Endpoint` struct with `yaml:"no_auth,omitempty" json:"no_auth,omitempty"` tags. Place it after `Meta` to keep auth-related fields grouped. Ensure the `Validate()` method doesn't reject it.

  **Patterns to follow:**
  - `Param.Required`, `Param.Positional` — same boolean-with-omitempty pattern
  - `Endpoint.Meta` — neighboring field in the struct

  **Test scenarios:**
  - Happy path: Endpoint with `NoAuth: true` round-trips through YAML marshal/unmarshal
  - Happy path: Endpoint with `NoAuth: false` (or unset) omits the field in JSON/YAML output
  - Edge case: Existing spec fixtures that don't have `no_auth` still parse without error (backward compat)

  **Verification:** `go test ./internal/spec/...` passes. Existing parser tests unaffected.

- [ ] **Unit 2: Parse per-operation `security: []` in OpenAPI parser**

  **Goal:** Detect when an OpenAPI operation explicitly opts out of auth and set `NoAuth = true` on the corresponding endpoint.

  **Requirements:** R1

  **Dependencies:** Unit 1

  **Files:**
  - Modify: `internal/openapi/parser.go` (in `mapResources()`, ~line 839-845)
  - Modify: `internal/openapi/parser_test.go`
  - Create: `testdata/openapi/mixed-auth.yaml` (test fixture)

  **Approach:**
  In `mapResources()`, after constructing each `endpoint` struct (line 839), check `op.Security`:
  - If `op.Security != nil && len(*op.Security) == 0` → `endpoint.NoAuth = true` (explicit `security: []`)
  - If `op.Security != nil` and any element is an empty map → `endpoint.NoAuth = true` (anonymous alternative)
  - If `op.Security == nil` → leave `NoAuth` as false (inherits global)

  After all resources are built (post-parse sweep), if `out.Auth.Type == "none" && !out.Auth.Inferred`, iterate all endpoints and set `NoAuth = true`. The `!Inferred` guard prevents marking endpoints as public when the parser simply failed to detect auth (some APIs omit security from their specs but still require it).

  Reference `parseSecurityRequirementSet()` in `scorecard.go:1113-1157` for the anonymous-detection logic.

  **Patterns to follow:**
  - `scorecard.go` `parseSecurityRequirementSet()` — same security array interpretation
  - Existing `mapResources()` patterns for setting endpoint fields

  **Test scenarios:**
  - Happy path: Operation with `security: []` → `endpoint.NoAuth == true`
  - Happy path: Operation with `security: [{}]` (empty object alternative) → `endpoint.NoAuth == true`
  - Happy path: Operation with `security: [{"api_key": []}]` (normal auth) → `endpoint.NoAuth == false`
  - Happy path: Operation with no per-operation security (nil) inheriting global auth → `endpoint.NoAuth == false`
  - Happy path: Spec with no global security (`Auth.Type == "none"`, `Inferred: false`) → all endpoints get `NoAuth == true`
  - Edge case: Mixed spec — some operations with `security: []`, some with auth, some inheriting → correct per-endpoint flags
  - Edge case: Spec with `Auth.Type == "none"` but `Inferred: true` (parser couldn't detect auth) → endpoints do NOT get `NoAuth == true` (fails safe)
  - Edge case: Existing petstore fixture still parses identically (no regression)

  **Verification:** `go test ./internal/openapi/...` passes including new tests. Petstore tests unchanged.

- [ ] **Unit 3: Annotate MCP tool descriptions with auth status**

  **Goal:** MCP tool descriptions include "(no auth required)" for public endpoints, helping agents decide which tools to call without setup.

  **Requirements:** R2

  **Dependencies:** Unit 1

  **Files:**
  - Modify: `internal/generator/templates/mcp_tools.go.tmpl`
  - Modify: `internal/generator/generator_test.go`

  **Approach:**
  The `oneline()` template function truncates descriptions at 120 characters. Naively appending `(no auth required)` after `oneline` would lose the annotation on long descriptions. Instead:

  1. Register a new template function `mcpDescription(desc string, noAuth bool) string` in `generator.go`'s FuncMap. This function appends " (no auth required)" when `noAuth` is true, *then* applies the same oneline cleanup and truncation. The suffix is part of the truncation input, not tacked on after.
  2. Update the template to call: `mcplib.WithDescription("{{mcpDescription $endpoint.Description $endpoint.NoAuth}}")`
  3. Apply the same change for both resource endpoints and sub-resource endpoints.

  **Files:**
  - Modify: `internal/generator/templates/mcp_tools.go.tmpl`
  - Modify: `internal/generator/generator.go` (register `mcpDescription` in FuncMap)
  - Modify: `internal/generator/generator_test.go`

  **Patterns to follow:**
  - Existing template function registration: `"oneline": oneline` in FuncMap at `generator.go:104`
  - Generator test pattern: `gen.Generate()` then `os.ReadFile()` and `assert.Contains()`

  **Test scenarios:**
  - Happy path: Spec with `NoAuth: true` endpoint → generated `tools.go` contains `(no auth required)` in that tool's description
  - Happy path: Spec with `NoAuth: false` endpoint → generated `tools.go` does NOT contain `(no auth required)` for that tool
  - Happy path: Mixed spec (some public, some auth) → annotation appears only on public tools
  - Edge case: Spec with `Auth.Type == "none"` (all public) → all tool descriptions get the annotation

  **Verification:** `go test ./internal/generator/...` passes. Generated MCP tools file compiles.

- [ ] **Unit 4: Add MCP metadata to CLI manifest**

  **Goal:** The `.printing-press.json` provenance manifest includes MCP metadata so published CLIs are self-describing for MCP capabilities.

  **Requirements:** R3

  **Dependencies:** Unit 1

  **Files:**
  - Modify: `internal/pipeline/climanifest.go`
  - Modify: `internal/pipeline/publish.go` (where manifest is populated)
  - Modify: `internal/naming/naming.go` (add `MCP()` function)
  - Modify: `internal/cli/root.go` (thread spec to manifest writer at lines 189 and 331)
  - Modify: `internal/generator/generator.go` (replace inline `name + "-mcp"` with `naming.MCP()`)
  - Modify: `internal/generator/templates/main_mcp.go.tmpl` (update server name string)
  - Modify: `internal/generator/templates/makefile.tmpl` (update MCP build target binary name)
  - Modify: `internal/generator/templates/goreleaser.yaml.tmpl` (update MCP build ID and binary name)

  **Approach:**
  Add fields to `CLIManifest`:
  - `MCPBinary string` (`json:"mcp_binary,omitempty"`) — e.g., "notion-pp-mcp"
  - `MCPToolCount int` (`json:"mcp_tool_count,omitempty"`) — total tools registered
  - `MCPPublicToolCount int` (`json:"mcp_public_tool_count,omitempty"`) — tools with `NoAuth`
  - `MCPReady string` (`json:"mcp_ready,omitempty"`) — "full", "partial", or "cli-only"
  - `AuthType string` (`json:"auth_type,omitempty"`) — from spec auth config
  - `AuthEnvVars []string` (`json:"auth_env_vars,omitempty"`) — from spec.Auth.EnvVars

  **Data plumbing for `WriteManifestForGenerate()`:** Add `Spec *spec.APISpec` to `GenerateManifestParams`. Both callers in `root.go` already have the parsed spec in scope — the `--docs` path has `parsed` (line 189) and the `--spec` path has `apiSpec` (line 331). Thread it through. Tool counts are computed by a new helper `countMCPTools(spec *spec.APISpec) (total, public int)` that iterates `spec.Resources` and their sub-resources.

  **Data plumbing for `writeCLIManifestForPublish()`:** `PipelineState` does not carry the parsed spec. Re-parse from `spec.json` in the working directory (already present at publish time). If the spec file is missing (e.g., `--docs` runs), MCP fields are left empty in the manifest — the publish skill can still populate them from the generated source.

  Add `naming.MCP(name string) string` returning `name + "-pp-mcp"` to `internal/naming/naming.go`. Update `generator.go:441` to use `naming.MCP()` instead of the current inline `name + "-mcp"`. This centralizes the convention and applies the `-pp-` collision avoidance infix.

  MCP readiness is computed: "full" if auth is `none` or `api_key`/`bearer_token` (env-var passable), "partial" if some endpoints are public but auth is cookie/composed, "cli-only" if all endpoints need cookie/composed auth and none are public.

  **Patterns to follow:**
  - Existing `CLIManifest` field patterns with `omitempty`
  - `WriteManifestForGenerate()` catalog lookup enrichment pattern
  - `naming.CLI()` pattern for `naming.MCP()`

  **Test scenarios:**
  - Happy path: Manifest for api-key CLI → `mcp_ready: "full"`, correct tool counts
  - Happy path: Manifest for no-auth CLI → `mcp_ready: "full"`, all tools are public
  - Happy path: Manifest for cookie CLI with mixed endpoints → `mcp_ready: "partial"`, public count < total
  - Happy path: Manifest for cookie CLI with no public endpoints → `mcp_ready: "cli-only"`
  - Edge case: Manifest fields omitted when zero (backward compat with schema_version 1)

  **Verification:** `go test ./internal/pipeline/...` passes. Manifests are valid JSON.

- [ ] **Unit 5: Add MCP section to README template**

  **Goal:** Every generated CLI README includes instructions for using the MCP server.

  **Requirements:** R4

  **Dependencies:** Unit 1

  **Files:**
  - Modify: `internal/generator/templates/readme.md.tmpl`

  **Approach:**
  Add a new "## Use as MCP Server" section after the "Agent Usage" section. Content varies by auth type:
  - API key: show `claude mcp add` with `-e ENV_VAR=<key>` and link to key URL
  - OAuth/bearer: show `claude mcp add` and note to run CLI `auth login` first
  - Cookie/composed: show `claude mcp add` and note partial availability, list that some tools work without auth
  - No auth: show bare `claude mcp add` — works immediately

  Also include Claude Desktop JSON config snippet for all auth types.

  **Patterns to follow:**
  - Existing auth-type conditional blocks in the README template (lines 24-83)
  - `{{.Name}}-pp-cli` naming pattern

  **Test scenarios:**
  - Test expectation: none — README template rendering is verified indirectly through compilation tests and manual review. No behavioral logic to unit test.

  **Verification:** Generated README for a test spec includes the MCP section with correct auth-type-specific content.

- [ ] **Unit 6: Generate smithery.yaml at publish time**

  **Goal:** The publish pipeline produces a `smithery.yaml` marketplace metadata file alongside each published CLI.

  **Requirements:** R5

  **Dependencies:** Unit 4

  **Files:**
  - Modify: `internal/pipeline/publish.go`
  - Modify: `internal/cli/publish.go` (in `publish package` command)

  **Approach:**
  After writing `.printing-press.json`, write a `smithery.yaml` to the same directory. Content is derived from the manifest and spec:
  - `name`: MCP binary name
  - `description`: from manifest description + " — generated from official OpenAPI spec" (or "sniffed" for sniffed specs)
  - `startCommand.command`: `./` + MCP binary name
  - `env`: map of env var name → `{description, required}` from auth config

  Only write smithery.yaml when `MCPReady` is "full" or "partial". Skip for "cli-only" (no useful MCP without browser setup).

  Apply traversal protection per institutional learning: smithery values don't flow into path construction, but validate that the MCP binary name matches expected patterns.

  **Patterns to follow:**
  - `WriteCLIManifest()` pattern — write file to dir with known name
  - Non-mutating publish pattern — write during `publish package`, not during `publish validate`

  **Test scenarios:**
  - Happy path: API-key CLI → smithery.yaml with env var marked required
  - Happy path: No-auth CLI → smithery.yaml with no env section
  - Happy path: Cookie CLI with partial readiness → smithery.yaml generated with env vars optional
  - Edge case: CLI-only readiness → no smithery.yaml written

  **Verification:** `go test ./internal/pipeline/...` passes. Generated smithery.yaml is valid YAML.

- [ ] **Unit 7: Report MCP tool split in scorecard**

  **Goal:** The scorecard summary includes an informational line about MCP tool counts (public vs auth-required) without adding a new scored dimension.

  **Requirements:** R8

  **Dependencies:** Unit 2

  **Files:**
  - Modify: `internal/pipeline/scorecard.go`

  **Approach:**
  In the scorecard's summary output section, add a one-line note when MCP tools exist: "MCP: {total} tools ({public} public, {auth} auth-required)". This is informational only — does not affect the score, tier constants, or grade.

  The scorecard currently uses raw `map[string]any` parsing via `loadOpenAPISpec()`, which doesn't produce structured `spec.Endpoint` objects with `NoAuth` flags. To get tool counts, call `openapi.Parse()` on the spec file (already available from the CLI directory). This avoids duplicating the `security: []` detection logic. The scorecard already imports `internal/openapi` for `IsOpenAPI()`. Reuse the same `countMCPTools()` helper from Unit 4.

  If the spec file is not present (sniffed CLIs), skip the MCP line.

  **Patterns to follow:**
  - Existing summary/gap reporting in scorecard output
  - `openapi.Parse()` call pattern from generator

  **Test scenarios:**
  - Happy path: Scorecard for CLI with mixed auth endpoints → summary line shows correct split
  - Happy path: Scorecard for CLI with all-public endpoints → summary shows "X tools (X public, 0 auth-required)"
  - Edge case: Scorecard for CLI without MCP → no MCP line in summary

  **Verification:** `go test ./internal/pipeline/...` passes. Scorecard output includes MCP line when applicable.

### Public Library Repo (mvanhorn/printing-press-library)

- [ ] **Unit 8: Extend registry.json with MCP metadata**

  **Goal:** Each entry in `registry.json` includes an `mcp` block with binary name, tool counts, auth type, and readiness level.

  **Requirements:** R6

  **Dependencies:** Unit 4 (uses same schema as manifest MCP fields)

  **Files:**
  - Modify: `registry.json` (in `mvanhorn/printing-press-library`)

  **Approach:**
  Add `mcp` object to each of the 6 entries:
  ```json
  "mcp": {
    "binary": "<name>-pp-mcp",
    "transport": "stdio",
    "tool_count": <N>,
    "public_tool_count": <N>,
    "auth_type": "<api_key|none|composed>",
    "env_vars": ["<VAR>"],
    "mcp_ready": "<full|partial>"
  }
  ```

  Values per CLI:
  - `dub-pp-cli`: 53 tools, api_key (DUB_TOKEN), mcp_ready: full
  - `espn-pp-cli`: 3 tools, none, mcp_ready: full (all public)
  - `linear-pp-cli`: tools TBD, api_key (LINEAR_TOKEN), mcp_ready: full
  - `pagliacci-pizza-pp-cli`: 41 tools, composed, mcp_ready: partial (public endpoints TBD during fixup)
  - `postman-explore-pp-cli`: 9 tools, none, mcp_ready: full (all public)
  - `steam-web-pp-cli`: 164 tools, api_key (STEAM_WEB_API_KEY), mcp_ready: full

  Exact public_tool_counts will be determined during Unit 9.

  **Patterns to follow:**
  - Existing registry.json entry schema
  - Additive schema extension (new fields, existing fields unchanged)

  **Test scenarios:**
  - Test expectation: none — registry.json is a data file. Validated by the publish skill reading it successfully.

  **Verification:** `registry.json` is valid JSON. All 6 entries have `mcp` blocks.

- [ ] **Unit 9: Annotate existing CLI MCP tool descriptions**

  **Goal:** Each published CLI gets two changes: (1) MCP binary renamed from `{name}-mcp` to `{name}-pp-mcp` for collision avoidance, and (2) "(no auth required)" appended to descriptions of public endpoints in `internal/mcp/tools.go`. Targeted source edits — no regeneration.

  **Requirements:** R7

  **Dependencies:** None (can run in parallel with machine changes)

  **Files:**
  For each of the 6 CLIs:
  - Rename: `cmd/{name}-mcp/` → `cmd/{name}-pp-mcp/`
  - Modify: `cmd/{name}-pp-mcp/main.go` (update server name string from `"{name}-mcp"` to `"{name}-pp-mcp"`)
  - Modify: `internal/mcp/tools.go` (auth annotations)
  - Modify: `Makefile` (update MCP build target)
  - Modify: `.goreleaser.yaml` (update MCP binary name and build ID)
  - Modify: `go.mod` imports if any reference the old cmd path

  **Approach:**
  For each CLI, rename the MCP binary and annotate public endpoints:

  **Step 0 — Rename MCP binary:**
  For all 6 CLIs: rename `cmd/{name}-mcp/` to `cmd/{name}-pp-mcp/`, update the server name string in `main.go`, update `Makefile` and `.goreleaser.yaml` build targets. This is a mechanical find-and-replace per CLI.

  **Step 1 — Classify endpoints by auth requirement:**
  - **espn, postman-explore**: Auth type is `none` → ALL endpoints are public. Annotate all.
  - **dub, linear**: Auth type is api_key → Skip. These APIs require auth for all operations. No annotations needed.
  - **pagliacci-pizza**: Composed cookie auth. For each candidate public endpoint (store finder, menu, location search), make one actual HTTP request without credentials using `curl`. If 200 with meaningful data → public. If 401/403 → auth-required. Document results in the PR.
  - **steam-web**: API key auth but Steam has many public endpoints. Same verification protocol: test each endpoint path with `curl` without the API key. Steam's public endpoints (app details, store search, community profiles) typically return data without auth. Document results.

  **Step 2 — Annotate `tools.go`:**
  For confirmed public endpoints, append ` (no auth required)` to the description string inside the `mcplib.WithDescription("...")` call. Be conservative — only annotate endpoints verified to work without auth.

  **Step 3 — Verify:**
  After editing, run `gofmt -w` on each modified file, then `go build ./...` in each CLI directory.

  **Patterns to follow:**
  - The exact string format: append ` (no auth required)` before the closing `"` in `mcplib.WithDescription("...")` calls

  **Test scenarios:**
  - Happy path: espn tools.go → all WithDescription calls include "(no auth required)"
  - Happy path: pagliacci tools.go → verified-public tools annotated, unverified/auth-required tools not
  - Happy path: steam tools.go → verified-public tools annotated
  - Edge case: Annotations don't break Go compilation — verify with `go build ./...` in each CLI dir

  **Verification:** Each modified CLI compiles: `cd <cli-dir> && go build ./...`. Every annotated endpoint was verified with an actual unauthenticated HTTP request. Verification results documented in the PR body.

- [ ] **Unit 10: Update library README with dual-interface install paths**

  **Goal:** The public library README shows both CLI and MCP install options for each published CLI.

  **Requirements:** R6

  **Dependencies:** Unit 8

  **Files:**
  - Modify: `README.md` (in `mvanhorn/printing-press-library`)

  **Approach:**
  Update the library's main README to include MCP install commands alongside CLI install commands for each entry. Show the `claude mcp add` one-liner with the correct env vars. Group entries by MCP readiness level to highlight what works immediately vs. what needs setup.

  **Patterns to follow:**
  - Existing README structure in the library repo

  **Test scenarios:**
  - Test expectation: none — documentation file, validated by manual review.

  **Verification:** README renders correctly in GitHub. All 6 CLIs have both install paths shown.

### Publish Skill Update (this repo)

- [ ] **Unit 11: Update publish skill to write MCP registry fields**

  **Goal:** The `/printing-press-publish` skill populates the `mcp` block in `registry.json` entries when packaging CLIs.

  **Requirements:** R6

  **Dependencies:** Unit 4, Unit 8

  **Files:**
  - Modify: `skills/printing-press-publish/SKILL.md`

  **Approach:**
  Update the registry entry construction in the publish skill to include the `mcp` block. The skill reads `.printing-press.json` for metadata — the new MCP fields from Unit 4 provide `mcp_binary`, `mcp_tool_count`, `mcp_public_tool_count`, `mcp_ready`, and `auth_type`. The skill maps these into the registry entry format.

  Also update the skill's documentation of the registry schema to include the new `mcp` block.

  **Patterns to follow:**
  - Existing registry entry construction in SKILL.md Step 8

  **Test scenarios:**
  - Test expectation: none — skill definitions are tested through end-to-end pipeline runs, not unit tests.

  **Verification:** Skill documentation accurately reflects the new registry schema. A manual publish dry-run produces correct registry entries.

## System-Wide Impact

- **Interaction graph:** The `NoAuth` field flows: OpenAPI parser → spec model → generator templates (MCP + potentially CLI help) → manifest → publish pipeline → registry.json → library README. Each hop is additive — no existing behavior changes.
- **Error propagation:** No new error paths. `NoAuth` defaults to `false` (zero value), so missing data fails safe (assume auth required).
- **State lifecycle risks:** None. `NoAuth` is computed at parse time and immutable thereafter.
- **API surface parity:** The CLI commands themselves don't surface `NoAuth` today. The MCP description annotation is the first consumer. CLI `--help` output could be enhanced later but is out of scope.
- **Integration coverage:** The end-to-end path (parse spec → generate MCP → read manifest → write registry) should be verified with one full pipeline run on a mixed-auth spec.
- **Unchanged invariants:** CLI binary generation, auth flows, doctor command, verify/dogfood, scorecard scoring formula — none of these change. The scorecard gets an informational line but no scoring changes.

## Risks & Dependencies

| Risk | Mitigation |
|------|------------|
| Existing published CLIs may not compile after annotation edits | Run `go build ./...` in each CLI directory after editing. Revert on failure. |
| Some OpenAPI specs may have ambiguous security (e.g., `security: [{}]` meaning different things) | Follow the same interpretation as the scorecard's `parseSecurityRequirementSet()` which is already battle-tested. |
| Smithery.yaml schema may change | Pin to current schema. The file is opt-in metadata — if Smithery changes, regeneration fixes it. |
| Registry.json schema extension breaks existing consumers | All new fields are optional (`mcp` block). Existing consumers that don't read `mcp` are unaffected. |
| Steam/Pagliacci public endpoint identification may be wrong | Verify every candidate with actual HTTP request without credentials. Only annotate confirmed-public endpoints. Document results in PR. |
| MCP binary rename breaks existing users | No marketplace listings exist yet. The 6 published CLIs are source-only in the library repo — no users have `claude mcp add` configs pointing to the old names. Rename cost is zero. |
| Specs that omit security but actually require auth (false public sweep) | Post-parse sweep guarded by `!Auth.Inferred`. Only marks all endpoints public when parser actively concluded no auth exists, not when detection failed. |
| `oneline()` truncation drops auth annotation on long descriptions | New `mcpDescription()` template function appends annotation before truncation, not after. Annotation is part of the truncation input. |

## Sources & References

- **Inspiration:** [fli project](https://www.punitarani.com/projects/fli) — Google Flights CLI/MCP that grew via marketplace listings
- Scorecard security parsing: `internal/pipeline/scorecard.go:1113-1157`
- OpenAPI parser endpoint construction: `internal/openapi/parser.go:800-851`
- MCP tools template: `internal/generator/templates/mcp_tools.go.tmpl`
- CLI manifest: `internal/pipeline/climanifest.go`
- Publish pipeline: `internal/pipeline/publish.go`, `internal/cli/publish.go`
- README template: `internal/generator/templates/readme.md.tmpl`
- Publish skill: `skills/printing-press-publish/SKILL.md`
- Institutional learnings: scorecard accuracy, filepath traversal, non-mutating validation, layout contract
- kin-openapi Operation.Security: `*SecurityRequirements` (`[]SecurityRequirement`)
