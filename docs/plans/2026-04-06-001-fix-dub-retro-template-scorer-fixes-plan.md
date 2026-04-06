---
title: "fix: Dub retro — FTS indexing, retry cap, dogfood auth, dedup, dead code"
type: fix
status: active
date: 2026-04-06
origin: docs/retros/2026-04-06-dub-retro.md
---

# fix: Dub retro — FTS indexing, retry cap, dogfood auth, dedup, dead code

## Overview

Five generator template and scorer fixes surfaced during the Dub CLI generation retro (issue #142). The most critical: `UpsertBatch` doesn't populate FTS indexes, making search silently return empty results after sync — every generated CLI is affected. The remaining four are smaller fixes that each save manual intervention on every or most future generations.

## Problem Frame

During the Dub run, five systemic Printing Press issues required manual code edits that would recur on every future CLI:
1. Search returns empty after sync (FTS not populated in batch path)
2. CLI hangs indefinitely on rate-limited APIs (no retry cap)
3. Dogfood falsely reports auth mismatch (checks wrong file)
4. Duplicate command registration panics (resource name = vision name)
5. Dead `formatCompact` function in every CLI (template emits unused code)

## Requirements Trace

- R1. After `sync --full` + `search <term>`, results must be returned (from retro F2, F3)
- R2. Rate-limited responses never cause waits longer than 60 seconds (from retro F4)
- R3. Dogfood correctly detects Bearer auth for all generated CLIs (from retro F5)
- R4. No duplicate AddCommand panics when API has analytics/search/events endpoints (from retro F1)
- R5. No dead functions emitted by default templates (from retro F6)

## Scope Boundaries

- Does not change the FTS schema, add new FTS tables, or change FTS trigger mode
- Does not change retry count, backoff strategy, or adaptive rate limiter logic
- Does not change scorecard auth scoring (only dogfood)
- Does not change promoted command logic or how resources are classified
- Does not add new template features — strictly bug fixes

## Context & Research

### Relevant Code and Patterns

- `internal/generator/templates/store.go.tmpl`: `Upsert()` (line 165) calls `upsertGenericResourceTx` which handles both resources table + resources_fts. `UpsertBatch()` (line 332) only does `INSERT OR REPLACE INTO resources` — no FTS
- `internal/generator/templates/sync.go.tmpl`: Line 237 calls `db.UpsertBatch()` for paginated sync. `upsertSingleObject()` (line 392) dispatches to per-table methods but is only used for non-array responses
- `internal/generator/templates/client.go.tmpl`: `retryAfter()` at line 547 parses Retry-After header with no maximum bound
- `internal/pipeline/dogfood.go`: `checkAuth()` at line 459 reads only `client.go`, but Bearer prefix is constructed in `config.go` via `AuthHeader()` method
- `internal/generator/templates/root.go.tmpl`: Resources loop (line 100) and VisionSet.Analytics (line 122) can both emit `newAnalyticsCmd`
- `internal/generator/templates/helpers.go.tmpl`: `formatCompact` at line 693 — no call sites in any template

### Test Patterns

- Tests use `t.TempDir()` with `writeTestFile()` helper
- Testify: `require` for setup, `assert` for validation
- `dogfood_test.go` creates mock CLI directory structures and runs `RunDogfood()`
- `generator_test.go` runs full spec→generate→compile→verify cycles

## Key Technical Decisions

- **FTS fix in UpsertBatch, not sync**: Adding FTS indexing to `UpsertBatch` is simpler and lower-risk than changing sync to dispatch to per-table methods. The per-table FTS tables remain unpopulated (that's a separate, larger change) but `resources_fts` — which is what the search command uses — will be populated
- **60-second cap**: Matches the fix applied during the Dub run. Generous enough for real rate limits, short enough to prevent hangs
- **Dogfood: check both files**: Adding `config.go` to the auth check is the minimal fix. The alternative (moving Bearer string to client.go template) would be a workaround for a scorer bug
- **Root dedup via VisionSet exclusion**: Adding VisionSet names to the Resources loop guard is the cleanest approach — it keeps the two registration paths independent

## Implementation Units

- [ ] **Unit 1: Add FTS indexing to UpsertBatch**

**Goal:** `resources_fts` is populated during batch sync so search returns results

**Requirements:** R1

**Dependencies:** None

**Files:**
- Modify: `internal/generator/templates/store.go.tmpl`
- Test: `internal/generator/generator_test.go`

**Approach:**
- In `UpsertBatch()` (line 332-362), add FTS `DELETE` + `INSERT` into `resources_fts` inside the transaction loop, after the `INSERT OR REPLACE INTO resources` statement
- Follow the same pattern as `upsertGenericResourceTx` (lines 147-160): delete by id, then insert with (id, resourceType, jsonContent)
- Use non-fatal error handling for FTS ops (match existing pattern: `fmt.Fprintf(os.Stderr, "warning: ...")`)

**Patterns to follow:**
- `upsertGenericResourceTx` in store.go.tmpl lines 147-160

**Test scenarios:**
- Happy path: Generate a CLI from testdata spec, verify `UpsertBatch` function body contains `resources_fts` INSERT statement
- Happy path: Full generate→compile cycle still passes with the template change

**Verification:**
- Generated store.go contains FTS indexing in UpsertBatch
- `go build ./...` passes on generated CLI
- Search returns results after sync (verified via generated code inspection)

---

- [ ] **Unit 2: Cap retryAfter at 60 seconds**

**Goal:** Rate-limited responses never cause waits longer than 60 seconds

**Requirements:** R2

**Dependencies:** None

**Files:**
- Modify: `internal/generator/templates/client.go.tmpl`
- Test: `internal/generator/generator_test.go`

**Approach:**
- Add `const maxRetryWait = 60 * time.Second` before the function
- After both integer parse and HTTP date parse, cap: `if d > maxRetryWait { return maxRetryWait }`

**Patterns to follow:**
- Existing `retryAfter` structure at client.go.tmpl line 547

**Test scenarios:**
- Happy path: Generated client.go contains `maxRetryWait` constant and cap logic
- Happy path: Full generate→compile cycle passes

**Verification:**
- Generated client.go contains the 60-second cap
- `go build ./...` passes

---

- [ ] **Unit 3: Fix dogfood Bearer detection to check config.go**

**Goal:** Dogfood correctly detects Bearer auth when the string is in config.go

**Requirements:** R3

**Dependencies:** None

**Files:**
- Modify: `internal/pipeline/dogfood.go`
- Test: `internal/pipeline/dogfood_test.go`

**Approach:**
- In `checkAuth()` (line 459), after reading `client.go`, also read `config.go` from `filepath.Join(dir, "internal", "config", "config.go")`
- Combine both sources for the string search: `combinedSource := clientSource + configSource`
- Search `combinedSource` for `"Bearer "` / `"Bot "` patterns

**Patterns to follow:**
- Existing `os.ReadFile` + `strings.Contains` pattern in dogfood.go lines 459-474

**Test scenarios:**
- Happy path: CLI with Bearer auth in config.go (not client.go) → dogfood reports MATCH
- Happy path: CLI with Bearer auth in client.go → still reports MATCH (no regression)
- Edge case: CLI with no auth in either file → reports "unknown" (no regression)
- Edge case: CLI with Bot auth in config.go → reports MATCH for Bot

**Verification:**
- `go test ./internal/pipeline/ -run TestRunDogfood` passes
- Existing dogfood tests continue to pass

---

- [ ] **Unit 4: Deduplicate VisionSet vs Resource AddCommand in root.go**

**Goal:** No duplicate AddCommand when an API resource name matches a VisionSet command

**Requirements:** R4

**Dependencies:** None

**Files:**
- Modify: `internal/generator/templates/root.go.tmpl`
- Test: `internal/generator/generator_test.go`

**Approach:**
- The collision names are the VisionSet command names: analytics, export, import, search, sync, tail
- In the Resources loop (line 100-104), add a check against these names. Two options:
  - Option A: Check VisionSet flags inline: `(not (and $.VisionSet.Analytics (eq $name "analytics")))`
  - Option B: Build a combined exclusion set in generator.go before template execution
- Option A is simpler and self-contained in the template. The guard at line 101 becomes:
  ```
  {{- if and (not (index $.PromotedResourceNames $name)) (ne $name "auth") (not (index $.VisionCmdNames $name))}}
  ```
  where `VisionCmdNames` is a `map[string]bool` passed from generator.go containing all enabled vision command names

**Patterns to follow:**
- `PromotedResourceNames` exclusion pattern at root.go.tmpl line 101
- Generator data construction in generator.go

**Test scenarios:**
- Happy path: Generate from a spec with `/analytics` endpoint + VisionSet.Analytics=true → root.go has exactly one `newAnalyticsCmd` registration
- Happy path: Generate from a spec without `/analytics` endpoint + VisionSet.Analytics=true → root.go has one `newAnalyticsCmd` from VisionSet
- Edge case: Generate from a spec with `/search` endpoint + VisionSet.Search=true → no duplicate
- Happy path: Full generate→compile cycle passes

**Verification:**
- `grep -c 'newAnalyticsCmd' generated/root.go` returns 1
- `go build ./...` passes

---

- [ ] **Unit 5: Remove dead formatCompact from helpers template**

**Goal:** No unused functions emitted by templates

**Requirements:** R5

**Dependencies:** None

**Files:**
- Modify: `internal/generator/templates/helpers.go.tmpl`
- Test: `internal/generator/generator_test.go`

**Approach:**
- Remove the `formatCompact` function (lines 693-705) from helpers.go.tmpl entirely
- Verify no template references it: `grep -r "formatCompact" internal/generator/templates/` should return nothing after removal

**Patterns to follow:**
- Other conditional helpers gated by `{{if}}` blocks in helpers.go.tmpl

**Test scenarios:**
- Happy path: Generated helpers.go does not contain `formatCompact`
- Happy path: `go vet ./...` reports no dead code in generated CLI
- Happy path: Full generate→compile cycle passes

**Verification:**
- `grep formatCompact generated/helpers.go` returns nothing
- `go build ./...` passes

## System-Wide Impact

- **FTS fix (Units 1)**: Affects every generated CLI's search behavior. No API surface change — internal store method only
- **Retry cap (Unit 2)**: Affects rate-limited API interactions. Lower-risk — only caps extreme values
- **Dogfood fix (Unit 3)**: Affects dogfood scoring for all bearer-auth CLIs. May improve auth_protocol scores by 2-5 points
- **Root dedup (Unit 4)**: Affects generated root.go for APIs with endpoint names matching vision commands. Narrow blast radius
- **Dead code (Unit 5)**: Affects every generated CLI's helpers.go. Trivial removal

## Risks & Dependencies

| Risk | Mitigation |
|------|------------|
| FTS indexing in UpsertBatch slows sync | Non-fatal FTS ops + same pattern as existing Upsert. FTS inserts are fast within a transaction |
| Dogfood change breaks existing tests | Run full `go test ./internal/pipeline/` before merging |
| VisionCmdNames map not populated correctly | Build from the same VisionSet flags already used in root.go.tmpl |

## Sources & References

- **Origin document:** [docs/retros/2026-04-06-dub-retro.md](docs/retros/2026-04-06-dub-retro.md)
- Related issue: #142
- Template files: `internal/generator/templates/store.go.tmpl`, `client.go.tmpl`, `root.go.tmpl`, `helpers.go.tmpl`
- Scorer: `internal/pipeline/dogfood.go`
