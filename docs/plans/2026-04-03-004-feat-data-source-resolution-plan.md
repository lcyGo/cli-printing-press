---
title: "feat: Data Source Resolution — Live-First with Honest Local Fallback"
type: feat
status: completed
date: 2026-04-03
---

# Data Source Resolution — Live-First with Honest Local Fallback

## Overview

Add a unified `--data-source auto|live|local` flag to every generated CLI so that all read commands share the same data resolution model: default to live API, fall back to local SQLite honestly on network failure, and always tell the user where results came from. Today, regular commands always hit the API (no fallback), and search always hits local SQLite (no live option). These two paths have zero coordination. This change makes every read command data-source-aware with provenance metadata on every response.

## Problem Frame

Generated CLIs have two disconnected data paths:

1. **Regular commands** (`links list`, `orders get`) always hit the live API. If the network is down, they fail. There is no way to use locally synced data.
2. **`search`** always hits local SQLite FTS5. If sync hasn't run, it errors. If sync ran last week, it silently shows stale data. If the API has a search endpoint, the search command ignores it.

No command tells the user whether data is live or cached, how old cached data is, or why a particular path was chosen. Agents calling `search --json` get results with no freshness metadata — they cannot reason about trustworthiness.

The fundamental issue: **no shared model for "where does data come from."**

## Requirements Trace

- R1. Every read command defaults to live API calls — no behavior change for existing users on non-search commands. Search behavior intentionally changes: it now tries the API search endpoint when available (R8), which is an improvement over the previous local-only behavior.
- R2. `--data-source local` works for any read command after `sync`, not just search
- R3. `--data-source auto` falls back to local on network failure with clear warning and sync timestamp
- R4. Provenance metadata (source, sync age, fallback reason) appears on every response — human and JSON
- R5. Write commands always hit the API regardless of `--data-source`
- R6. `sync` command unchanged — remains the only path to populate local data
- R7. No TTL, no auto-sync, no read-through caching, no silent fallbacks
- R8. When API has a search endpoint and mode is auto/live, `search` hits the API endpoint
- R9. When API has no search endpoint, `search` uses local FTS with an explanation of why

## Scope Boundaries

- **Not changing sync.** Sync stays explicit, user-initiated, identical to today.
- **Not adding read-through caching.** Regular commands do NOT populate the local database as a side effect. However, if local data exists from an explicit `sync`, `auto` mode may read it on network failure. This is fallback usage of pre-synced data, not read-through caching.
- **Not adding deletion detection.** `sync --full` remains the workaround for deleted-upstream records.
- **Not adding per-resource data source config.** V1 uses a single flag for all resources.
- **Not changing write commands.** POST/PUT/PATCH/DELETE always hit the API.

## Context & Research

### Relevant Code and Patterns

- `internal/generator/templates/root.go.tmpl` — rootFlags struct definition, persistent flag registration, `newClient()` helper (line 151)
- `internal/generator/templates/command_endpoint.go.tmpl` — per-endpoint command generation; GET commands call `c.Get(path, params)` at line 86, POST/PUT/PATCH/DELETE use `c.Post()`/`c.Put()`/etc.
- `internal/generator/templates/search.go.tmpl` — always queries local SQLite via `db.Search{{pascal .Name}}()`, no live option, no provenance
- `internal/generator/templates/store.go.tmpl` — `GetSyncState()` (line 385), `GetLastSyncedAt()` (line 417), `SaveSyncState()` (line 374) already exist
- `internal/generator/templates/client.go.tmpl` — pure HTTP client with `Get()`, `Post()`, etc. Adaptive rate limiter.
- `internal/generator/templates/helpers.go.tmpl` — shared helper functions, output formatting, error classification
- `internal/generator/templates/channel_workflow.go.tmpl` — `defaultDBPath()` at line 200, workflow archive/status commands
- `internal/profiler/profiler.go` — `hasSearchEndpoint` detected at line 133 but only used to gate `NeedsSearch`; `SyncableResource` struct has `{Name, Path}`
- `internal/generator/schema_builder.go` — `TableDef` with `FTS5` bool, FTS5 tables get `Search{{pascal .Name}}()` methods
- `internal/generator/generator.go` — template data assembly, vision feature selection

### Institutional Learnings

- **DB path inconsistency** (Postman Explore retro): `defaultDBPath()` in `channel_workflow.go.tmpl` uses `~/.config/`, but search/sync used `~/.local/share/`. All local data access must go through a single canonical path. This change should consolidate `defaultDBPath()` into `helpers.go.tmpl`.
- **Store template search gap** (5 retros): The store template gates FTS5 Search methods on `{{if .FTS5}}` but the profiler underdetects searchable fields. The generated `Search{{pascal .Name}}()` methods are present for high-gravity tables — we can rely on them for local reads.
- **Verify gap for local-data commands** (Postman Explore Run 2 retro): Verify has no mechanism to distinguish "needs local data" commands from "hits live API" commands. The `--data-source` flag gives verify a way to test local-data paths.

## Key Technical Decisions

- **Resolution logic lives in a separate vision-gated template (`data_source.go.tmpl`), not in helpers**: The HTTP client stays a pure HTTP client. `resolveRead()` lives in a new `data_source.go.tmpl` that is only rendered when `VisionSet.Store` is true — following the same pattern as `search.go.tmpl` and `sync.go.tmpl`. This avoids a compilation failure: `helpers.go.tmpl` is rendered for ALL CLIs unconditionally, but `resolveRead()` imports the `store` package which only exists when `VisionSet.Store` is true. CLIs without a store continue calling `c.Get()` directly (zero change from today). The endpoint template conditionally calls `resolveRead()` when the data source template exists.

- **GET commands call `resolveRead()` instead of `c.Get()` directly (when store is available)**: The endpoint template for GET commands swaps `c.Get(path, params)` for `resolveRead(c, flags, resourceType, isList, path, params)` when `VisionSet.Store` is true. The function takes an `isList bool` parameter (set at generation time based on whether the endpoint has pagination) to distinguish list vs get-by-ID operations — avoids fragile runtime path parsing. Write commands are unchanged.

- **Provenance always wraps JSON output**: A `DataProvenance` struct (`source string`, `syncedAt *time.Time`, `reason string`) travels with results. JSON output always wraps in `{"results": [...], "meta": {...}}` — including for successful live calls. This is a breaking change to JSON output shape but there are no existing users yet. Human output prints a one-line provenance to stderr (stdout stays clean for piping).

- **`defaultDBPath()` moves to `helpers.go.tmpl` with canonical path `~/.local/share/`**: Consolidates the DB path in one place. The canonical path is `~/.local/share/<cli-name>/data.db` (XDG compliant, where sync already writes). ALL templates that reference DB paths must be updated: `channel_workflow.go.tmpl` (currently `~/.config/`), `search.go.tmpl` (currently hardcodes `~/.local/share/`), workflow and insight templates (currently `~/.config/`). The `~/.config/` path is eliminated entirely.

- **Search endpoint detection flows into template data**: The profiler already detects `hasSearchEndpoint`. We add `SearchEndpointPath` and `SearchQueryParam` to the template data so the search template can conditionally hit the API. For live search response normalization, use the same unwrapping heuristic that `paginatedGet()` already uses — walk known wrapper paths (`data`, `results`, `items`) to extract the result array.

- **`store.List()` output is normalized to match API response shape**: `store.List()` returns `[]json.RawMessage` while API responses may be wrapped in `{"data": [...]}` etc. `resolveRead()` marshals `[]json.RawMessage` from the store into a single `json.RawMessage` array. Local-mode results will NOT include API envelope fields (pagination metadata, totals) — this is documented.

## Open Questions

### Resolved During Planning

- **Where does the resolution function live?** In a new `data_source.go.tmpl` that is only rendered when `VisionSet.Store` is true. NOT in `helpers.go.tmpl` — helpers is rendered unconditionally for all CLIs, but `resolveRead()` imports the `store` package which only exists when VisionSet.Store is true. The separate template follows the same pattern as `search.go.tmpl` and `sync.go.tmpl`.

- **How do GET endpoint commands read local data?** Via `store.List()` and `store.Get()` which already exist. `resolveRead()` takes an `isList bool` parameter (set at generation time based on whether the endpoint has pagination) to choose `List()` vs `Get()`. The store's `[]json.RawMessage` from `List()` is marshaled into a single `json.RawMessage` array for compatibility with the downstream rendering pipeline.

- **Does `--data-source` affect non-GET read operations?** No. Only `GET` and `HEAD` methods. The endpoint template already branches on method — the data source branch nests inside the GET branch only.

- **How does the search command know the API search endpoint path?** The profiler extracts it at generation time and passes it via the `visionData` struct in `generator.go`. The search template conditionally includes the live-search code path when `SearchEndpointPath` is non-empty.

- **How is the JSON output shape changing?** Every `--json` response now wraps in `{"results": [...], "meta": {...}}` — including default auto+live. This is a breaking change accepted because there are no existing users. The envelope is always present for consistency — agents always know where provenance lives.

- **How does `resolveRead()` handle paginated GET commands?** When mode is `local`, skip `paginatedGet()` entirely and call `store.List()` directly (local store has all synced data). When mode is `live` or `auto`, use `paginatedGet()` as today. When `auto` + network error on first page, fall back to `store.List()`.

- **Which DB path is canonical?** `~/.local/share/<cli-name>/data.db` (XDG compliant, where sync already writes). The `~/.config/<name>/store.db` path in `channel_workflow.go.tmpl` and workflow/insight templates is eliminated. All templates updated to use `defaultDBPath()` from helpers.

- **How does `resolveRead()` know the resource type?** The endpoint template passes `.ResourceName` (the spec resource map key, e.g., "links", "domains"). The sync command upserts using `SyncableResource.Name` from the profiler. These should match but must be verified — add a test that sync writes with a resource type retrievable by `resolveRead()`.

- **How are API search results normalized?** Use the same unwrapping heuristic `paginatedGet()` uses — walk known wrapper paths (`data`, `results`, `items`) to extract the result array. If none match, treat the response as a bare array.

### Deferred to Implementation

- **Exact conditional import mechanics for endpoint templates.** The endpoint template needs `{{if}}` guards around the `store` import and `resolveRead()` call, gated on whether the VisionSet includes Store. The exact template conditionals depend on what fields are available in the endpoint template data — may need to add a `HasStore bool` to the endpoint data struct.

## High-Level Technical Design

> *This illustrates the intended approach and is directional guidance for review, not implementation specification. The implementing agent should treat it as context, not code to reproduce.*

```
Command (GET endpoint)
  │
  ├─ flags.dataSource == "live" or ("auto" default)?
  │    │
  │    ├─ Call c.Get(path, params)
  │    │    │
  │    │    ├─ Success → return results, provenance{source:"live"}
  │    │    │
  │    │    └─ Network error + mode == "auto"?
  │    │         │
  │    │         ├─ Open store, query local data
  │    │         │    │
  │    │         │    ├─ Data exists → return results, provenance{source:"local", reason:"api_unreachable", syncedAt:...}
  │    │         │    │
  │    │         │    └─ No data → error: "API unreachable and no local data. Run 'sync' first."
  │    │         │
  │    │         └─ mode == "live"? → error: "API unreachable."
  │    │
  │    └─ (Non-network error) → return error as usual
  │
  └─ flags.dataSource == "local"?
       │
       ├─ Open store, query local data
       │    │
       │    ├─ Data exists → return results, provenance{source:"local", reason:"user_requested", syncedAt:...}
       │    │
       │    └─ No data → error: "No local data. Run 'sync' first."
       │
       └─ (Store not initialized) → error with hint to sync


Search command:
  │
  ├─ mode == "local"?
  │    └─ Use FTS5 (existing behavior + provenance)
  │
  ├─ API has search endpoint?
  │    │
  │    ├─ Yes → Call c.Get(searchPath, {queryParam: query})
  │    │    │
  │    │    ├─ Success → return results, provenance{source:"live"}
  │    │    │
  │    │    └─ Network error + mode == "auto"?
  │    │         └─ Fall back to FTS5 + provenance{reason:"api_unreachable"}
  │    │
  │    └─ No → Use FTS5 + provenance{reason:"no_search_endpoint"}
  │
  └─ mode == "live" + no search endpoint?
       └─ Use FTS5 + provenance{reason:"no_search_endpoint"} with explanation
```

## Implementation Units

- [ ] **Unit 1: Add `--data-source` flag and provenance types**

**Goal:** Add the `dataSource` field to `rootFlags`, register the `--data-source` persistent flag, define the `DataProvenance` struct, and add provenance output helpers.

**Requirements:** R1, R4, R7

**Dependencies:** None

**Files:**
- Modify: `internal/generator/templates/root.go.tmpl`
- Modify: `internal/generator/templates/helpers.go.tmpl`
- Test: `internal/generator/generator_test.go` (verify generated code compiles with new flag)

**Approach:**
- Add `dataSource string` to `rootFlags` struct in `root.go.tmpl`
- Register `--data-source` as a persistent string flag with default `"auto"` and allowed values `auto|live|local`
- In `PersistentPreRunE`, validate the flag value (must be auto, live, or local)
- Define `DataProvenance` struct in `helpers.go.tmpl`: `Source string`, `SyncedAt *time.Time`, `Reason string`, `ResourceType string`
- Add `printProvenance(cmd, provenance)` helper that writes the one-line stderr message (e.g., "3 results (live)" or "3 results (cached, synced 2 hours ago)")
- Add `wrapWithProvenance(data, provenance)` helper that wraps JSON results in `{"results": [...], "meta": {...}}` when `--json` is set
- Move `defaultDBPath()` from `channel_workflow.go.tmpl` to `helpers.go.tmpl` with canonical path `~/.local/share/<cli-name>/data.db` (XDG compliant, where sync already writes). Update ALL templates that reference DB paths: `channel_workflow.go.tmpl` (remove its copy), `search.go.tmpl` (replace hardcoded path at line 66), `sync.go.tmpl` (replace hardcoded path), and any workflow/insight templates using `~/.config/`

**Patterns to follow:**
- Existing flag registration pattern in `root.go.tmpl` (lines 50-69)
- Existing `PersistentPreRunE` validation for `--agent` flag (lines 71-88)
- Existing helper functions in `helpers.go.tmpl` (e.g., `classifyAPIError`, `wantsHumanTable`)

**Test scenarios:**
- Happy path: generated CLI compiles with `--data-source` flag present, `--help` shows the flag with correct description
- Happy path: `--data-source auto` is the default when flag is not explicitly set
- Edge case: `--data-source invalid` rejected in PersistentPreRunE with clear error
- Happy path: `printProvenance()` formats "3 results (live)" for live source
- Happy path: `printProvenance()` formats "3 results (cached, synced 2 hours ago)" with relative time
- Happy path: `wrapWithProvenance()` produces `{"results": [...], "meta": {"source": "live"}}` JSON envelope
- Edge case: `wrapWithProvenance()` includes `synced_at` and `reason` only when source is "local"

**Verification:**
- Generate a test CLI, build it, confirm `--data-source` flag appears in `--help`
- Confirm `--data-source invalid` produces a validation error

---

- [ ] **Unit 2: Add `resolveRead()` in new `data_source.go.tmpl`**

**Goal:** Implement the core resolution logic in a new vision-gated template that dispatches reads to either the HTTP client or local store based on the `--data-source` flag value, network availability, and local data existence.

**Requirements:** R1, R2, R3, R7

**Dependencies:** Unit 1

**Files:**
- Create: `internal/generator/templates/data_source.go.tmpl`
- Modify: `internal/generator/generator.go` (add `data_source.go.tmpl` to vision-gated rendering, only when `VisionSet.Store` is true)
- Test: `internal/generator/generator_test.go`

**Approach:**
- Create `data_source.go.tmpl` in the `cli` package. This template is only rendered when `VisionSet.Store` is true (following the pattern of `search.go.tmpl`, `sync.go.tmpl`). CLIs without a store never get this file — they call `c.Get()` directly.
- Add `resolveRead()` function with signature: `client *client.Client`, `flags *rootFlags`, `resourceType string`, `isList bool`, `path string`, `params map[string]string` → returns `(json.RawMessage, DataProvenance, error)`
- The `isList` parameter is set at generation time based on whether the endpoint has `.Endpoint.Pagination`. This avoids fragile runtime path parsing to distinguish list vs get-by-ID.
- Resolution logic:
  - If `dataSource == "local"`: open store via `defaultDBPath()`, call `store.List(resourceType, limit)` when `isList` or `store.Get(resourceType, id)` when not. Marshal `[]json.RawMessage` from `List()` into a single `json.RawMessage` array. If no data, return error with sync hint. Attach provenance with `reason: "user_requested"`.
  - If `dataSource == "live"`: call `c.Get(path, params)`. On success, return with `source: "live"`. On network error, return the error directly (no fallback).
  - If `dataSource == "auto"` (default): call `c.Get(path, params)`. On success, return with `source: "live"`. On network error, try the store. If store has data, return it with `source: "local"`, `reason: "api_unreachable"`, and the sync timestamp. If store has no data, return error: "API unreachable and no local data."
- Also add `resolvePaginatedRead()` for paginated GET commands: when `local`, skip `paginatedGet()` and call `store.List()` directly. When `live`/`auto`, delegate to `paginatedGet()`. When `auto` + network error, fall back to `store.List()`.
- Network error detection: check for `*net.OpError`, `*url.Error`, DNS errors, connection refused, and timeout errors. HTTP 4xx/5xx errors are NOT network errors — they propagate normally.
- Store is opened lazily — only when local data is actually needed.
- The function queries existing `store.GetSyncState(resourceType)` to get the sync timestamp for provenance.

**Patterns to follow:**
- Existing `classifyAPIError()` pattern in helpers for error inspection
- Existing `store.Open()` / `store.List()` / `store.Get()` API

**Test scenarios:**
- Happy path: `auto` mode with live API reachable → returns live data with `source: "live"`
- Happy path: `auto` mode with network error and local data exists → returns local data with `source: "local"`, `reason: "api_unreachable"`, populated `synced_at`
- Edge case: `auto` mode with network error and no local data → returns error mentioning both API unreachability and sync hint
- Happy path: `live` mode with live API reachable → returns live data
- Error path: `live` mode with network error → returns error "API unreachable" (no fallback)
- Happy path: `local` mode with synced data → returns local data with `reason: "user_requested"`
- Error path: `local` mode with no synced data → returns error "No local data. Run 'sync' first."
- Edge case: HTTP 401/403/500 errors are NOT treated as network errors — they propagate as API errors even in `auto` mode
- Edge case: Store DB file doesn't exist → graceful handling (no panic, clear error)

**Verification:**
- Generated CLI compiles with `resolveRead()` function present
- Unit tests cover all branches of the resolution matrix

---

- [ ] **Unit 3: Update GET command template for data source resolution**

**Goal:** Modify the endpoint command template so that GET commands call `resolveRead()` instead of `c.Get()` directly, enabling live/local/auto data source behavior.

**Requirements:** R1, R2, R3, R5

**Dependencies:** Unit 1, Unit 2

**Files:**
- Modify: `internal/generator/templates/command_endpoint.go.tmpl`
- Modify: `internal/generator/templates/command_promoted.go.tmpl` (promoted commands also call `c.Get()` and `paginatedGet()` directly — lines 54, 70)
- Test: `internal/generator/generator_test.go`

**Approach:**
- In the `{{if or (eq .Endpoint.Method "GET") (eq .Endpoint.Method "HEAD")}}` branch (line 68), replace the direct `c.Get(path, params)` call with `resolveRead(c, &flags, "{{resourceName .}}", path, params)`
- Apply the same swap in `command_promoted.go.tmpl` for its GET branch (line 70) and `paginatedGet()` call (line 54), so promoted aliases respect `--data-source` identically to their non-promoted equivalents
- The `resourceName` template function maps the endpoint's parent resource to the store's resource type name (same normalization sync uses)
- Add conditional `store` import when the VisionSet includes Store/Sync features
- After `resolveRead()` returns, call `printProvenance()` for human output (stderr) and `wrapWithProvenance()` for JSON output
- Write commands (POST, PUT, PATCH, DELETE) remain unchanged — they continue calling `c.Post()` etc. directly
- The `paginatedGet()` helper also needs wrapping — when `--data-source local`, pagination is not needed (local store returns all matching rows)

**Patterns to follow:**
- Existing method branching in `command_endpoint.go.tmpl` (line 68-153)
- Existing `wantsHumanTable()` output mode detection pattern

**Test scenarios:**
- Happy path: generated GET command with `--data-source auto` (default) calls API and returns live data — no behavior change from today
- Happy path: generated GET command with `--data-source local` returns locally synced data
- Happy path: generated POST command ignores `--data-source` flag entirely — always hits API
- Integration: GET command with `--data-source auto` and simulated network error falls back to local data with provenance warning on stderr
- Edge case: paginated GET with `--all` flag and `--data-source local` returns all local data without pagination calls
- Happy path: `--json` output includes `meta.source` field for GET commands

**Verification:**
- Generate a CLI from a known spec, confirm GET commands show provenance in both table and JSON modes
- Confirm POST/PUT/PATCH/DELETE commands are unchanged

---

- [ ] **Unit 4: Expose search endpoint details in profiler**

**Goal:** When the profiler detects a search endpoint, extract and expose the endpoint path and query parameter name so the search template can conditionally hit the API.

**Requirements:** R8, R9

**Dependencies:** None (can run in parallel with Units 1-3)

**Files:**
- Modify: `internal/profiler/profiler.go`
- Modify: `internal/generator/generator.go` (extend the `visionData` struct to include `SearchEndpointPath` and `SearchQueryParam`, populated from `g.profile`)
- Test: `internal/profiler/profiler_test.go`

**Approach:**
- Add `SearchEndpointPath string` and `SearchQueryParam string` fields to `APIProfile`
- When `hasSearchEndpoint` is true (line 133), also record the endpoint's path and identify the query parameter (look for params named `q`, `query`, `search`, `keyword`, `term` — pick the first match)
- Extend the `visionData` struct in `generator.go` (around line 413) to include `SearchEndpointPath` and `SearchQueryParam`, sourced from `g.profile`. The existing pipeline does NOT automatically flow APIProfile fields into visionData — the struct must be manually extended.

**Patterns to follow:**
- Existing search endpoint detection at line 130-134 of profiler.go
- Existing `SyncableResource` struct pattern for exposing detected data to templates

**Test scenarios:**
- Happy path: API with `/search` endpoint and `q` query param → `SearchEndpointPath` = "/search", `SearchQueryParam` = "q"
- Happy path: API with `/users/search` endpoint and `query` param → correctly extracted
- Edge case: API with no search endpoint → both fields are empty strings
- Edge case: API with search endpoint but no recognizable query param → `SearchEndpointPath` set, `SearchQueryParam` defaults to "q"
- Happy path: multiple search endpoints (e.g., `/users/search` and `/orders/search`) → picks the most general one (shortest path or root-level)

**Verification:**
- Profile a spec with a known search endpoint, confirm fields are populated
- Profile a spec without a search endpoint, confirm fields are empty

---

- [ ] **Unit 5: Update search template for live/local routing and provenance**

**Goal:** Modify the search command to try the API search endpoint (when available) before falling back to local FTS, and add provenance metadata to all search output.

**Requirements:** R4, R8, R9

**Dependencies:** Unit 1, Unit 4

**Files:**
- Modify: `internal/generator/templates/search.go.tmpl`
- Test: `internal/generator/generator_test.go`

**Approach:**
- When `SearchEndpointPath` is non-empty in template data AND `dataSource` is `auto` or `live`:
  - Call `c.Get(searchEndpointPath, map[string]string{searchQueryParam: query})`
  - Normalize the response: unwrap known envelope paths (`data`, `results`, `items`) to extract the result array — same heuristic `paginatedGet()` uses
  - On success: return results with `source: "live"`
  - On network error + `auto` mode: fall back to local FTS with `reason: "api_unreachable"`
  - On `live` mode + no network: return error
- When `SearchEndpointPath` is empty (API has no search endpoint):
  - `auto` and `live` modes both use local FTS with `reason: "no_search_endpoint"` and print explanation: "This API has no search endpoint. Searching local data."
  - `local` mode uses local FTS with `reason: "user_requested"`
- Add provenance to all output paths — both JSON (`meta` envelope) and human (stderr line)
- Remove the `Args: cobra.ExactArgs(1)` constraint and add `if len(args) == 0 { return cmd.Help() }` for verify compatibility (same pattern used to fix tail in the dub-pp-cli polish)

**Patterns to follow:**
- Existing search command structure in `search.go.tmpl`
- Existing `resolveRead()` pattern from Unit 2 for live/local dispatching
- Provenance helpers from Unit 1

**Test scenarios:**
- Happy path: API with search endpoint, `auto` mode → hits API, returns live results with `source: "live"`
- Happy path: API with search endpoint, `local` mode → uses local FTS, returns with `reason: "user_requested"`
- Happy path: API without search endpoint, `auto` mode → uses local FTS with `reason: "no_search_endpoint"`, stderr shows explanation
- Error path: API with search endpoint, `live` mode, network error → returns error
- Happy path: API with search endpoint, `auto` mode, network error → falls back to local FTS with `reason: "api_unreachable"`
- Happy path: `--json` output wraps results in `{"results": [...], "meta": {"source": "...", ...}}`
- Edge case: local FTS with no synced data → error: "No local data. Run 'sync' first."
- Happy path: human output shows "3 results (live)" or "3 results (cached, synced 2 hours ago)" on stderr

**Verification:**
- Generate a CLI from a spec with a search endpoint, confirm search tries API first
- Generate a CLI from a spec without a search endpoint, confirm search uses local FTS with explanation
- Confirm JSON output includes provenance metadata in all cases

---

**Deferred: Persistent data-source config default.** Adding `data_source` to the config file and env var override is a convenience feature that serves no stated requirement. The `--data-source` flag (Unit 1) delivers all required functionality. Config persistence can be added as a follow-up if users request it.

## System-Wide Impact

- **Interaction graph:** The `resolveRead()` helper sits between all GET command handlers and the client/store. It is the single point of control for data source decisions. The search command has its own resolution path (because it may use API search endpoints), but follows the same provenance pattern.
- **Error propagation:** Network errors in `auto` mode are caught and trigger fallback — they do NOT propagate as command errors. HTTP 4xx/5xx errors always propagate. In `live` mode, network errors propagate as errors.
- **State lifecycle risks:** No new mutable state. The local SQLite database is read-only from the perspective of `resolveRead()`. Only `sync` writes to it. No cache invalidation, no write-through.
- **API surface parity:** The `--data-source` flag applies to all read commands uniformly. Write commands are unaffected. The flag is a root-level persistent flag so it works identically across all subcommands.
- **Integration coverage:** The key integration scenario is: generate a CLI → run `sync` → verify `--data-source local` returns synced data → verify `--data-source auto` with network down falls back to local with provenance.
- **Unchanged invariants:** `sync` command behavior is unchanged. Write command behavior is unchanged. The HTTP client API is unchanged. The store schema is unchanged (only new query methods added). The profiler's `NeedsSearch` gate is unchanged.

## Risks & Dependencies

| Risk | Mitigation |
|------|------------|
| Store-less CLIs break if resolveRead() is in helpers.go | Resolved: `resolveRead()` lives in `data_source.go.tmpl`, only rendered when `VisionSet.Store` is true. CLIs without stores never see this code. |
| JSON output breaking change: wrapping in `{"results": [...], "meta": {...}}` | Accepted: always wrap, no existing users. Document in generated README. Envelope is always present for agent consistency. |
| Template complexity: GET branch becomes significantly more complex | Keep `resolveRead()` and `resolvePaginatedRead()` as single well-tested helpers in `data_source.go.tmpl`. The per-command template change is a one-line swap. |
| Local data shape differs from API response shape | `resolveRead()` marshals `[]json.RawMessage` from `store.List()` into a single JSON array. Local results omit API envelope fields (pagination metadata, totals). Documented as expected behavior. |
| DB path inconsistency across templates | Resolved: canonical path is `~/.local/share/<cli-name>/data.db`. All templates updated in Unit 1 to use `defaultDBPath()` from helpers. |
| Sync timestamp accuracy: `last_synced_at` may not reflect actual data freshness if sync was partial | Sync already tracks per-resource timestamps. `resolveRead()` uses the resource-specific timestamp, not a global one. |
| Resource type name mismatch between endpoint templates and sync | Add test verifying that `.ResourceName` from the endpoint template matches `SyncableResource.Name` from the profiler for the same resource. |

## Sources & References

- Related code: `internal/generator/templates/` (all template files listed above)
- Related code: `internal/profiler/profiler.go` (APIProfile, NeedsSearch, hasSearchEndpoint)
- Related code: `internal/generator/schema_builder.go` (TableDef, FTS5 decisions)
- Retro: `docs/retros/2026-03-30-postman-explore-retro.md` (DB path inconsistency finding)
- Retro: `docs/retros/2026-04-03-postman-explore-run2-retro.md` (verify gap for local-data commands)
