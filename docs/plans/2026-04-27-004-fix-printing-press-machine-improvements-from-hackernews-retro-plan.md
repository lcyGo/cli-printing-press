---
title: 'Printing Press machine improvements from hackernews retro #350'
type: fix
status: active
date: 2026-04-27
origin: ~/printing-press/manuscripts/hackernews/20260427-120911/proofs/20260427-130958-retro-hackernews-pp-cli.md
---

# Printing Press machine improvements from hackernews retro #350

## Overview

Implement the eight in-scope work units from the hackernews retro ([issue #350](https://github.com/mvanhorn/cli-printing-press/issues/350)). The retro raised nine findings about the Printing Press; F3 (multi-base-URL spec change) was demoted to deferred and stays out of this plan. The remaining work spans three subsystems — generator templates, scoring tools, and skill instructions — and is mostly small mechanical fixes plus one meta-improvement to the retro skill itself.

The unifying thread: **stop emitting machine-internal data into user-facing surfaces, and stop classifying narrow findings as broad ones.** Phantom freshness paths render machine fallbacks into README/SKILL. Backspace bytes leak from JSON-parsed regex into rendered markdown. Silent zero-row sync logs report "500 synced" when nothing was stored. Each is a leak between layers that has a direct fix.

---

## Problem Frame

Hackernews regen on printing-press v2.3.9 produced a Grade A CLI but surfaced eight systemic Printing Press issues during the retro. Some affect every cache-enabled CLI generation (phantom freshness paths, dead `wrapResultsWithFreshness` helper). Some affect API subclasses (no-paginator APIs ignore `?limit=N`, sync of bare-ID arrays writes zero rows silently). Some are scoring tool false positives (reimplementation_check missing secondary clients, dogfood path-validity FAIL on internal-yaml). One — F9 — is a meta-finding about the retro skill itself producing over-broad findings, which was the catalyst for the audit pass that reshaped this plan.

The fixes are independent and can land in any order. Most are small enough to ship as individual commits.

---

## Requirements Trace

- R1. Trim `FreshnessCommands` rendered into README/SKILL to only paths whose subcommands actually exist (origin: F1).
- R2. Detect control bytes in rendered SKILL/README output and reject the render with file:offset and source field name. Add skill instruction warning agents to double-escape backslashes in `narrative.recipes[].command` and similar fields (origin: F2).
- R3. Reimplementation_check recognizes imports of sibling `internal/<name>` packages as legitimate API access (origin: F4).
- R4. Sync emits a stderr warning and a structured event with `stored: 0` when it consumes items but writes zero rows (origin: F5).
- R5. When the spec profiler detects no paginator on a list endpoint, generated commands truncate the response client-side via `truncateJSONArray` after the API call returns (origin: F6).
- R6. Dogfood reports `Path Validity: SKIPPED (internal-yaml spec)` instead of `0/0 valid (FAIL)` when the spec source is internal YAML (origin: F8).
- R7. Decide the canonical pattern for `wrapResultsWithFreshness` — either delete it from the template, or add SKILL Phase 3 instruction directing agents to use it (origin: F7).
- R8. Tighten the retro skill's blast-radius gate: require three concrete cross-API examples before classifying a finding as P1/P2; add a counter-check question and a recurrence-cost check (origin: F9).

**Origin trace:** the origin document is the retro itself (no requirements doc precedes it), so this section omits Actors / Key Flows / Acceptance Examples — the origin's WUs map 1:1 to this plan's units below.

---

## Scope Boundaries

- WU-3 from the retro (add `enrichment_apis:` section to spec format) is **out of scope**. The retro audit moved it to deferred P3 because the same finding has been raised in three retros without justifying the implementation cost. Re-evaluate when combo CLIs exceed ~40% of catalog.
- The hydration spec field for `response_format: id_list` is **out of scope** (originally retro F5b, now Skipped). Subclass too narrow (~3% of APIs). Hand-build hydration in printed CLIs that need it.
- This plan does not retroactively re-classify findings in existing retros under `~/printing-press/manuscripts/`. WU-9's stricter gate applies to **new** retros only.
- This plan does not touch the printed `hackernews` CLI in the library. Any printed-CLI-specific fixes were applied during the regen session; this plan is exclusively about the Printing Press.

### Deferred to Follow-Up Work

- **WU-3 (`enrichment_apis` spec format)** — separate plan when justification threshold is met. The current workaround (hand-build a 150-LOC client per multi-source CLI) is acceptable.

---

## Context & Research

### Relevant Code and Patterns

- `internal/generator/generator.go::freshnessCommandPaths()` (lines 636-664) — current source of phantom paths. Iterates `g.profile.SyncableResources` and unconditionally adds `prefix`, `prefix + " list"`, `prefix + " get"`, `prefix + " search"`. The same struct (`g.profile.SyncableResources[].Endpoints`) already tracks which endpoints actually exist — that's the data freshness paths should consult.
- `internal/generator/generator.go::Generator.generate()` (line 973-974) — maps `skill.md.tmpl` → `SKILL.md` and `readme.md.tmpl` → `README.md`. This is the natural injection point for a render-time control-byte sanitizer.
- `internal/pipeline/reimplementation_check.go::clientCallRe` (around line 100) — regex set: `flags\.newClient`, `http\.(Get|Post|NewRequest|Do)`, `c\.(Do|Get|Post)`. The fix extends this with sibling-internal-package import detection.
- `internal/pipeline/reimplementation_check.go::hasStoreSignal()` (around line 200) — existing pattern for "look at file content for an import line + call site". The same shape works for client signal: detect imports matching `"<module>/internal/<sibling-name>"` where sibling-name ≠ `client` and ≠ `store`.
- `internal/generator/templates/sync.go.tmpl::syncResource` (line 352, 406) — `db.UpsertBatch(resource, items)` is called, then `sync_complete` event is emitted with `total: %d` where `%d` is `totalCount`. `totalCount` increments by `len(items)` after each successful UpsertBatch — but UpsertBatch can silently no-op for items whose ID extraction fails. This is the gap.
- `internal/pipeline/dogfood.go::Run()` (lines 205-230) — already has a `spec.IsSynthetic()` branch that records `Skipped: true` for synthetic specs. Internal-yaml specs without `kind: synthetic` fall through to `checkPaths(...)` which runs but reports oddly when the spec has no recognized paths.
- `internal/generator/templates/helpers.go.tmpl` (line 1263) — defines `wrapResultsWithFreshness`. No call sites in any generated template.
- `skills/printing-press-retro/SKILL.md::Phase 3` — current blast-radius checklist (Step A "Cross-API stress test", Step B "Estimate frequency", Step C "Assess fallback cost", Step D "Make the tradeoff"). Steps B and C are open-ended — agents handwave concrete evidence.

### Institutional Learnings

- `docs/retros/2026-04-13-recipe-goat-retro.md` — case-history motivation for the agentic Phase 4.85 output review. Same shape as F9: a previously-acceptable bias toward "ship it" produced false negatives that an agent layer caught. The fix in that case was structural (new phase). The fix here is procedural (stricter gate question).
- `docs/retros/` (movie-goat retro F8, hackernews v1.3.3 retro F3) — both raised the spec-multi-base-URL finding before. Recurrence without implementation IS evidence the cost-benefit hasn't justified — F9 codifies recognizing this pattern.

### External References

None. All work is internal to this repo; no external API or framework decisions.

---

## Key Technical Decisions

- **Phantom-paths fix (R1) trims the rendered slice, not the runtime fallback map.** The map in generated `auto_refresh.go` keeps its `<resource> list/get/search` variants because Cobra's argument resolution can land on any of them at runtime — the no-op fallback is harmless. Only the `.FreshnessCommands` slice rendered into user-facing docs needs trimming.
- **Render-time sanitizer (R2) checks ALL rendered output, not just recipe fields.** A targeted check on `narrative.recipes[].command` would only catch this specific case. A general "no control bytes 0x00-0x08, 0x0B-0x0C, 0x0E-0x1F" check on all rendered SKILL/README/manifest output catches every future class of escape mistake. Tab (0x09), newline (0x0A), and carriage return (0x0D) are explicitly allowed.
- **Sibling-package detection (R3) is broad, not narrow.** Any import of `<module>/internal/<x>` where `<x>` is not `client` or `store` counts as client signal. Worst case: a non-client internal helper package is mistakenly recognized as a client. That's still a strictly-less-bad outcome than today's false positive (real Algolia client miscategorized as reimplementation).
- **Sync warn (R4) compares stored-vs-consumed.** Either thread a return value through `UpsertBatch` reporting rows-actually-written, or run a `SELECT count(*)` before/after. The first is more precise; the second is simpler and works without changing the upsert API. Choice deferred to implementation — both meet the acceptance criteria.
- **Limit truncation (R5) defaults to ON when no paginator detected.** This is the safer default than ON when explicitly opted-in: every API without a paginator silently breaks `--limit` today; turning truncation on by default matches user expectations and is harmless when the API already returns ≤ limit rows.
- **WU-8 (R7) decision: delete the helper.** Five months of generated CLIs and zero call sites. Polish-worker removes it on every run. Adding a SKILL instruction to use it would burden the hand-build path without clear benefit when `wrapWithProvenance` already covers the use case from spec-driven commands. Removing it from the template eliminates a dead-code finding that polish-worker has been masking.
- **Retro skill gate (R8) is procedural, not enforced.** The SKILL.md changes add three explicit gate questions to Phase 3 question 5. The skill cannot mechanically force three concrete API names — but a stricter prompt produces measurably more rigorous findings. Same approach as the existing "Cross-API stress test" prose.

---

## Open Questions

### Resolved During Planning

- **Should WU-1 also remove the runtime fallback variants from `auto_refresh.go`?** No. Keep the map's no-op variants for runtime resolution flexibility; trim only the rendered slice. (See Key Technical Decisions above.)
- **Should the render-time sanitizer be opt-in per template or applied globally?** Globally, on every rendered text file (SKILL.md, README.md, anything else). Opt-in defeats the purpose.
- **Do we keep `wrapResultsWithFreshness` and document it, or delete?** Delete. (See Key Technical Decisions above.)

### Deferred to Implementation

- **Exact mechanism for sync's stored-vs-consumed comparison.** Either thread row count back from UpsertBatch, or do a count query bracketed around the UpsertBatch call. Implementer picks based on whether modifying UpsertBatch's return type ripples beyond `sync.go.tmpl`.
- **Profiler signature for "no paginator detected".** Need to confirm the profiler exposes a single check or whether the generator inspects `endpoint.Pagination == nil` directly. Implementer reads the existing pagination-aware emit path and follows the same convention.
- **Names of the three concrete APIs the retro gate forces an agent to list.** N/A — the gate requires the AGENT to name three; the skill instructions don't prescribe which.

---

## Implementation Units

- U1. **Trim phantom freshness paths from rendered docs**

**Goal:** README and SKILL list only freshness paths whose subcommands actually exist for each resource.

**Requirements:** R1.

**Dependencies:** None.

**Files:**
- Modify: `internal/generator/generator.go` — refactor `freshnessCommandPaths()` to consult endpoint existence
- Modify: `internal/generator/generator_test.go` — extend existing test fixtures
- Test: `internal/generator/generator_test.go` (existing file)

**Approach:**
- Iterate `g.profile.SyncableResources` as today, but for each resource only emit `prefix + " " + endpointName` for endpoints actually declared on that resource. Always emit the bare `prefix` (resource-as-command).
- Leave the `auto_refresh.go.tmpl` runtime map untouched — it continues emitting its no-op fallback variants because those help Cobra arg resolution.
- Existing `Cache.Commands` block (custom command-path coverage from spec) continues unchanged.

**Patterns to follow:**
- The endpoint iteration shape at `generator.go::458` and `498` already walks `r.Endpoints` for emission; freshness paths should mirror it.

**Test scenarios:**
- Happy path: a resource with endpoints `top`, `new`, `best`, `get` (HN's stories) emits `["<cli> stories", "<cli> stories top", "<cli> stories new", "<cli> stories best", "<cli> stories get"]` in `.FreshnessCommands`.
- Happy path: a resource with a single endpoint `list` emits `["<cli> ask", "<cli> ask list"]` (the bare command stays because Cobra resolves `<cli> ask` to the promoted single endpoint).
- Edge case: a resource with zero endpoints (shouldn't exist in valid specs, but guard) emits only `["<cli> <resource>"]`.
- Negative: `<cli> ask get`, `<cli> ask search`, `<cli> stories list`, `<cli> stories search` do NOT appear unless those endpoints actually exist.
- Integration: a generated SKILL.md and README.md from the existing `golden-api.yaml` fixture render only real paths after the change. Verify via `scripts/golden.sh verify` — fixture update may be needed and should be part of the same commit.

**Verification:**
- All real freshness paths still resolve to actual subcommands in a generated CLI.
- `scripts/golden.sh verify` passes (after intentional fixture update for the changed paths).
- `internal/generator/auto_refresh.go.tmpl` is unchanged — runtime map still has no-op variants.

---

- U2. **Render-time control-byte sanitizer + skill warning**

**Goal:** Generator rejects any rendered SKILL.md/README.md/etc. that contains ASCII control bytes (0x00-0x08, 0x0B-0x0C, 0x0E-0x1F). Skill instructions for `narrative.recipes[].command` and similar fields warn agents to double-escape backslashes for embedded regex.

**Requirements:** R2.

**Dependencies:** None.

**Files:**
- Modify: `internal/generator/generator.go` — add post-render sanitization check around the `render(...)` call site (near line 970-980)
- Modify: `skills/printing-press/SKILL.md` — Phase 1.5e research.json instructions, narrative.recipes section, add the double-escape warning
- Test: `internal/generator/sanitizer_test.go` (new file)

**Approach:**
- After each template render, before writing the file to disk, scan the byte stream for disallowed control chars. Allow `\t` (0x09), `\n` (0x0A), `\r` (0x0D). Reject everything else in the 0x00-0x1F range.
- On rejection, return an error naming the file, the byte offset, and a hint about which source field is most likely responsible (best-effort: scan recent template variables that could carry user input).
- In SKILL.md research.json instructions, add one-liner under the `narrative.recipes[].command` description: "Regex literals in `recipes[].command`, `troubleshoots[].fix`, and `quickstart[].command` must double-escape backslashes (`\\b` not `\b`) so JSON parses to literal `\b` rather than backspace."

**Patterns to follow:**
- `internal/generator/generator.go` already has post-render passes for some files (e.g., gofmt on `.go` outputs). Add the sanitizer in the same pass shape.

**Test scenarios:**
- Happy path: rendered SKILL.md with normal text passes the sanitizer unchanged.
- Edge case: rendered SKILL.md containing tab/newline/CR passes (these are allowed).
- Error path: rendered SKILL.md containing a backspace byte (0x08) is rejected; error message names the file and the byte offset.
- Error path: rendered output containing form-feed (0x0C), bell (0x07), or other control chars is rejected.
- Integration: generate a CLI with a research.json containing `"command": "...\\bGo\\b..."` (2 backslashes — JSON parses as backspace). Generation fails with the sanitizer's error, not silently shipping mojibake.
- Integration: generate a CLI with a research.json containing `"command": "...\\\\bGo\\\\b..."` (4 backslashes — JSON parses as `\bGo\b`). Renders correctly.

**Verification:**
- Existing CLIs in `~/printing-press/library/` regenerate cleanly (no false-positive sanitizer rejections).
- A deliberately-broken research.json fails generation with a clear error pointing at the file and offset.
- SKILL.md research.json instructions surface the double-escape rule before agents author recipes.

---

- U3. **Reimplementation_check recognizes sibling internal packages**

**Goal:** Hand-built API client packages in `internal/<name>/` (where `<name>` is not `client` or `store`) are recognized as legitimate API access; commands that import and call into them no longer trip the reimplementation false positive.

**Requirements:** R3.

**Dependencies:** None.

**Files:**
- Modify: `internal/pipeline/reimplementation_check.go` — add sibling-internal-import detection alongside existing `clientCallRe`
- Test: `internal/pipeline/reimplementation_check_test.go` — extend existing table-driven tests

**Approach:**
- Add a regex `siblingInternalImportRe` matching `"[^"]*/internal/(?!client\b|store\b|cliutil\b|cache\b|config\b|mcp\b|types\b)[a-z][a-z0-9]*"` — any internal package not in the generator's reserved set.
- In `classifyReimplementation`, set `hasClient = true` when the file content matches `siblingInternalImportRe`. This treats "imports a non-reserved sibling internal package" as evidence of legitimate API access.
- Reserved package names are the ones the generator emits unconditionally. List them in a const so the test verifies the list is consistent.

**Patterns to follow:**
- The existing `hasStoreSignal` function (around line 200) shows the pattern: regex match on file content → boolean signal → drives the classification result.

**Test scenarios:**
- Happy path: a command file importing `"hackernews-pp-cli/internal/algolia"` and calling any method on a typed value does NOT trip reimplementation. `hasClient = true` is set by the new regex.
- Happy path: a command file importing `"food52-pp-cli/internal/recipescraper"` (hypothetical) does NOT trip reimplementation.
- Negative: a command file importing only `"<module>/internal/store"` does NOT match the sibling-internal regex (store is reserved), but is still exempted by the existing store carve-out — net behavior unchanged.
- Negative: a command file importing only stdlib and returning a hardcoded JSON literal still trips reimplementation (`hasClient = false`, `hasTrivialBody = true`).
- Negative: a command file importing nothing from `<module>/internal/` and not calling any of the canonical client patterns still trips reimplementation.
- Edge case: a command file importing `<module>/internal/cliutil` does NOT trip the new signal (cliutil is reserved). The existing cliutil-aware logic governs.

**Verification:**
- The hackernews CLI (which imports `internal/algolia`) regenerates with 0 reimplementation findings instead of 5.
- Existing CLIs without secondary clients keep their current scoring (no regressions).
- `internal/pipeline/reimplementation_check_test.go` table grows by ≥3 cases covering the new signal.

---

- U4. **Sync warns when consumed-vs-stored counts diverge**

**Goal:** Sync emits an honest count of stored rows. When `UpsertBatch` consumed N items but stored zero (e.g., bare-ID arrays from Firebase-shaped APIs), users see a clear warning and the structured event reports `stored: 0`.

**Requirements:** R4.

**Dependencies:** None.

**Files:**
- Modify: `internal/generator/templates/sync.go.tmpl` — replace the `totalCount += len(items)` post-UpsertBatch with a stored-count read
- Modify: `internal/generator/templates/store.go.tmpl::UpsertBatch` — return the count of rows actually written
- Test: `internal/generator/templates_test.go` (existing) or new `internal/generator/sync_template_test.go`

**Approach:**
- Change `UpsertBatch(resourceType string, items []json.RawMessage) error` to `UpsertBatch(resourceType string, items []json.RawMessage) (int, error)` returning the count of rows actually upserted. Bump the rows-written counter inside the inner upsert helper that already runs per-item.
- In sync.go.tmpl: after `stored, err := db.UpsertBatch(...)`, set `totalCount += stored` (not `+= len(items)`). When `len(items) > 0 && stored == 0`, emit:
  - Stderr warning: `<resource> returned scalar items; consumed N, stored 0. The store will be empty for this resource. Likely cause: API returns ID lists rather than objects.`
  - Structured event: `{"event":"sync_anomaly","resource":"%s","consumed":%d,"stored":0,"reason":"all_items_failed_id_extraction"}`
- Existing structured `sync_complete` event still emits but now with the honest `stored` count.

**Patterns to follow:**
- The existing `sync_warning` event shape (around line 280 in sync.go.tmpl) for stderr messaging during sync.

**Test scenarios:**
- Happy path: a sync receives `[{id:1, ...}, {id:2, ...}]` → both upserted, `stored: 2`, no warning, `sync_complete` shows `total: 2`.
- Edge case: an empty array `[]` → `stored: 0`, no warning (consumed was also 0).
- Error path: a sync receives `[1, 2, 3]` (bare IDs) → `stored: 0`, warning emitted, `sync_anomaly` event emitted, `sync_complete` shows `total: 0`.
- Error path: a sync receives `[{id:1}, "garbage"]` → `stored: 1`, no warning (some rows landed). Mixed shapes are not flagged.
- Integration: regenerate hackernews and run `sync` against /topstories.json. Stderr now shows the warning, JSON events show `stored: 0`, log no longer claims "500 synced".

**Verification:**
- Hackernews's sync log changes from `"500 synced"` to a clear "0 stored, 500 consumed" message with the warning.
- CLIs whose sync writes object arrays show zero behavior change.
- `golden.sh verify` passes after the template change (golden may need update if any fixture exercises sync; check first).

---

- U5. **Client-side limit truncation when no paginator detected**

**Goal:** Generated list commands honor `--limit N` even against APIs that ignore the `?limit=` query param, by truncating the response client-side after the API call returns.

**Requirements:** R5.

**Dependencies:** None.

**Files:**
- Modify: `internal/generator/templates/` — list-endpoint command templates (the ones currently emitting `params["limit"] = ...`); the same templates that produced `stories_top.go`-style files
- Modify: `internal/generator/templates/helpers.go.tmpl` — add `truncateJSONArray` helper to the generated CLI's helpers package
- Test: existing template tests; add a fixture spec with no pagination to exercise the truncation path

**Approach:**
- Detect "no paginator" in the generator: an endpoint where `endpoint.Pagination == nil` AND no recognized paginator query param (look at existing pagination detection at `generator.go::684` for the established convention).
- When the endpoint has no paginator AND the command accepts a `--limit` flag, emit `data = truncateJSONArray(data, flagLimit)` after `resolveRead` returns.
- Generate `truncateJSONArray` into the printed CLI's helpers package (it's already present in hackernews's hand-written `limit_helper.go` — promote the same shape into the template).
- For endpoints WITH a paginator, emit nothing new — the API already honors the limit.

**Patterns to follow:**
- Hackernews's hand-written `internal/cli/limit_helper.go::truncateJSONArray` is the canonical implementation. Same logic, generated into every CLI.
- Existing `extractPageItems` in generated sync.go shows the JSON array detection pattern.

**Test scenarios:**
- Happy path: a spec with `endpoints.list { params: [{name: limit, ...}] }` and no `pagination:` block emits a command that calls `truncateJSONArray(data, flagLimit)` after `resolveRead`.
- Happy path: `<cli> resource list --limit 3 --json | jq '.results | length'` returns 3 even when the API returns 500 items.
- Edge case: `--limit 0` (default) → truncateJSONArray is a no-op (returns input unchanged).
- Negative: a spec with explicit `pagination: {limit_param: per_page}` emits no client-side truncation (defers to the API's paginator).
- Negative: a spec with cursor-based pagination (`pagination: {cursor_param: page_token}`) emits no client-side truncation.
- Integration: regenerate one cache-enabled CLI with a paginated API (e.g., a Stripe-style spec) and verify the generated list command does NOT call truncateJSONArray.
- Integration: regenerate hackernews and verify the 6 commands previously hand-patched (`stories_top.go` etc.) now have generator-emitted `truncateJSONArray` instead of needing the printed-CLI helper.

**Verification:**
- Hackernews's `internal/cli/limit_helper.go` becomes redundant with generator output (printed CLI can drop the hand-built file on its next regen).
- A no-paginator API correctly honors `--limit` end-to-end.
- A paginated API behaves identically to today (no double-truncation).

---

- U6. **Dogfood path-validity reports SKIPPED for internal-yaml specs**

**Goal:** Dogfood emits a clear `Path Validity: SKIPPED (internal-yaml spec)` line instead of the contradictory `0/0 valid (FAIL)` users see today on internal-yaml CLIs.

**Requirements:** R6.

**Dependencies:** None.

**Files:**
- Modify: `internal/pipeline/dogfood.go` — extend the existing `IsSynthetic()` skip branch to cover all internal-yaml specs, OR add a parallel branch
- Test: `internal/pipeline/dogfood_test.go` — add a case for non-synthetic internal-yaml

**Approach:**
- Today's code (around line 213) sets `Skipped: true` only when `spec.IsSynthetic()`. Extend this to also skip when the spec source is internal-yaml (regardless of synthetic flag), since path-validity assumes OpenAPI-style path patterns that internal-yaml expresses differently.
- Detail message: `"internal-yaml spec — paths validated at parse time"` (per acceptance criteria).
- Verify the report's overall verdict logic doesn't FAIL on a SKIPPED path-validity check — should already be the case since synthetic specs don't fail today.

**Patterns to follow:**
- The existing `spec.IsSynthetic()` branch at `dogfood.go:212-218` — same shape, broader condition.

**Test scenarios:**
- Happy path: dogfood on an internal-yaml CLI without `kind: synthetic` reports `Path Validity: SKIPPED (internal-yaml spec — paths validated at parse time)`.
- Happy path: dogfood on a synthetic internal-yaml CLI keeps the existing skip message (don't regress that case).
- Happy path: dogfood on an OpenAPI CLI keeps the `N/M valid` count (don't regress the common case).
- Negative: overall dogfood verdict on an internal-yaml CLI with all other checks passing is PASS or WARN, not FAIL, attributable to path-validity (the existing scorecard already excludes this dimension; verify alignment).

**Verification:**
- Hackernews dogfood report no longer shows `0/0 valid (FAIL)`. Shows `SKIPPED` instead.
- Scorecard's existing internal-yaml exclusion logic is unchanged (no double-skip).
- An OpenAPI CLI's dogfood is byte-for-byte identical pre/post change.

---

- U7. **Delete dead `wrapResultsWithFreshness` helper**

**Goal:** Remove the unused `wrapResultsWithFreshness` helper from the generator template so future CLIs don't ship dead code that polish-worker has to scrub.

**Requirements:** R7.

**Dependencies:** None.

**Files:**
- Modify: `internal/generator/templates/helpers.go.tmpl` — remove `wrapResultsWithFreshness` function
- Modify: `internal/generator/templates/auto_refresh.go.tmpl` — remove the comment referring to `wrapResultsWithFreshness` at line 150
- Test: `internal/generator/generator_test.go` — verify no test asserts on its presence (likely none, but check)

**Approach:**
- Decision rationale (Key Technical Decisions above): zero call sites in five months of generated CLIs, polish-worker removes it on every run, `wrapWithProvenance` already covers the use case from spec-driven commands. Delete and move on.
- If implementation discovers any unexpected call site (it shouldn't), pause and reassess — could indicate the helper was load-bearing for an obscure path.

**Patterns to follow:**
- N/A — pure deletion.

**Test scenarios:**
- Happy path: a regenerated CLI compiles cleanly without the helper.
- Happy path: dogfood does NOT report a dead-helper finding for `wrapResultsWithFreshness` after this change (since the helper no longer exists).
- Negative: existing tests still pass (no test should be referencing the helper).

**Verification:**
- `grep -r wrapResultsWithFreshness internal/generator/` returns zero results.
- A fresh `printing-press generate` run produces a CLI whose `internal/cli/helpers.go` does not contain the helper.
- Polish-worker, on the next polish pass for any CLI, no longer reports the dead-helper finding.

**Test expectation: none — pure deletion of an unused symbol. The verification is that nothing breaks; no new behavior to test.**

---

- U8. **Tighten retro skill's machine-vs-CLI gate**

**Goal:** Future retros require concrete cross-API evidence before classifying findings as P1/P2. The skill instructions add three explicit gates: name three concrete APIs, counter-check question, recurrence-cost check.

**Requirements:** R8.

**Dependencies:** None.

**Files:**
- Modify: `skills/printing-press-retro/SKILL.md` — Phase 3 question 5 ("Blast radius and fallback cost") and the cardinal rules at the top

**Approach:**
- In Phase 3 question 5, replace Step B ("Estimate frequency") with a stricter form requiring three concrete API names. Add bullet wording like:

  > **Step B (revised): Name three concrete APIs that would benefit.** List three specific APIs by name (e.g., "Stripe, Notion, GitHub") that would benefit from this fix beyond the one that surfaced it. If you can only name two — or one plus handwaving "many APIs in theory" — the finding is capped at P3 with a `subclass:<name>` annotation. Concrete evidence is the burden of proof, not optimism.

- After Step D, add a new Step E:

  > **Step E: Counter-check question.** Ask explicitly: "If I implemented this fix, would it actively hurt any API that doesn't have this pattern?" If yes, the fix needs a guard or condition before being P1/P2 — not a default change.

- Add a new Step F:

  > **Step F: Recurrence-cost check.** Search prior retros under `~/printing-press/manuscripts/*/proofs/*-retro-*.md` for the same finding. If the same finding has been raised in 2+ prior retros without being implemented, the prior cost-benefit math has been "no" twice. Don't re-raise it at the same priority — either move to P3 with a "raised N times, still not justified" annotation, or reframe the finding into a smaller incremental fix.

- In the cardinal rules at the top of SKILL.md, add a counterweight to "Bias toward fixing":

  > **Bias toward fixing only when the fix would help APIs you can name.** "20% of catalog" without naming three concrete APIs is not evidence — it's optimism. The retro is a triage tool, not a wishlist.

**Patterns to follow:**
- The existing prose-style rule wording in `skills/printing-press-retro/SKILL.md::Cardinal Rules`. Match the tone of the existing five rules.

**Test scenarios:**
- Happy path: an agent retro-ing a CLI tries to classify a finding as P2 with frequency "API subclass: ~20% of catalog". The Step B gate forces them to name three concrete APIs. If they can only name two, the finding lands as Skipped or P3.
- Happy path: a finding with clear cross-API evidence (e.g., F1's "every cache-enabled CLI") sails through the gate unchanged.
- Edge case: a finding raised previously (2+ prior retros) cannot be re-raised at P2; agent must reframe or downgrade.
- Negative: existing well-supported findings in the retro template aren't disrupted.

**Verification:**
- The next retro run after this change produces a findings list where every P1/P2 finding names three concrete APIs in its "Frequency" or "Cross-API check" field.
- Findings that fail the gate appear in Skipped with a `subclass:<name>` tag rather than ghosted into the P-buckets.
- A re-retroed CLI doesn't re-raise the same finding at the same priority without a recurrence-cost annotation.

**Test expectation: none — skill instruction changes are not unit-testable. The verification is observational on the next retro session.**

---

## System-Wide Impact

- **Interaction graph:** U1 (freshness paths) and U7 (helper deletion) both touch generator templates that affect every CLI generation. U2 (sanitizer) intercepts every render. U3 (reimplementation_check) runs on every dogfood. U4 (sync warn) modifies behavior in every printed CLI's sync command. U5 (limit truncation) modifies generation for every list-endpoint command. U6 (path-validity) affects every dogfood report. U8 (retro skill) affects every retro session.
- **Error propagation:** U2 introduces a new error class (render-time control-byte rejection) that aborts generation. Generators exit non-zero with a clear file:offset:reason message. No silent swallowing.
- **State lifecycle risks:** None. No persistent state changes.
- **API surface parity:** U4 changes `UpsertBatch`'s return signature (`error` → `(int, error)`). All call sites in the generated `sync.go` need updating in the same template change. No external callers to `UpsertBatch` exist (it's per-CLI-generated code). Verify by grep across `internal/generator/templates/`.
- **Integration coverage:** U1 + U2 + U5 all need to be exercised against the existing `golden-api.yaml` fixture. Run `scripts/golden.sh verify` after each unit; expect intentional fixture updates for U1 and U5.
- **Unchanged invariants:**
  - Cobra runtime command resolution is unchanged. The `auto_refresh.go` runtime map keeps its no-op fallback variants.
  - Existing scorecard scoring rules are unchanged. U3 only changes the boolean signal feeding into reimplementation classification; the score weights stay the same.
  - The synthetic-spec carve-out in dogfood (U6) is unchanged — internal-yaml is added as a peer condition, not a replacement.

---

## Risks & Dependencies

| Risk | Mitigation |
|------|------------|
| U2 sanitizer false-positives on legitimate template output (rare control bytes in spec content) | Test against every CLI in `~/printing-press/library/` during implementation. Allow `\t`, `\n`, `\r` explicitly. If a real output legitimately needs a control byte, error message names it precisely so the spec author can fix the input. |
| U4's `UpsertBatch` signature change ripples to other template files | Grep all uses across `internal/generator/templates/*.tmpl` before changing the signature. Update every caller in the same commit. Pre-existing CLIs in the library don't matter (each CLI's generated code is internally consistent). |
| U5 inadvertently double-truncates on APIs that appear unpaginated but secretly honor limit | The truncation is idempotent — if the API returns `limit` items already, `truncateJSONArray` is a no-op. False positives are harmless. False negatives (paginated API mistakenly considered unpaginated) would result in client-side truncation when not strictly needed; still correct, just slightly less efficient. |
| U8 retro-skill changes don't actually change agent behavior because instructions are advisory | Mitigate by including concrete-language requirements ("name three", "search prior retros") rather than vague guidance. Calibrate by retroing the next 2-3 CLIs and observing whether the rigor improved. If not, escalate to a structural fix (e.g., a script that grep-searches prior retros for duplicate findings). |
| Golden fixtures need updating in U1 and U5; mistaking those updates for genuine drift | Run `scripts/golden.sh verify` BEFORE making the code change, then again AFTER. Compare diffs to confirm only the expected fixtures changed. Document the expected diff in the commit message. Only after manual diff inspection, run `scripts/golden.sh update`. |

---

## Documentation / Operational Notes

- **AGENTS.md guidance check:** AGENTS.md says "When you change code, check for a `_test.go` file in the same package." Each unit lists the corresponding test file.
- **Pre-commit / pre-push hooks:** repo has lefthook hooks (`go fmt -w`, `golangci-lint`). Each unit's implementation should pass these locally before committing. AGENTS.md warns against `gofmt -w .` from repo root (rewrites golden fixtures); use `go fmt ./...` per package patterns.
- **Skill workflow parity (AGENTS.md):** U2 changes generator behavior in a way that affects what the skill should warn about. Update `skills/printing-press/SKILL.md` Phase 1.5e in the same change as U2's generator code.
- **Commit hygiene:** Use the repo's commit-style convention from AGENTS.md. Each unit lands as its own commit with a `fix(cli):` or `fix(skills):` scope. Multi-subsystem units (U2 touches both `internal/generator/` and `skills/`) use `fix(cli):` since the generator change is the load-bearing half. U8 is `fix(skills):` since it's skill-only.
- **Release impact:** None of these changes are user-visible API breaks. They're behavior fixes and bug-elimination. Standard release-please flow applies.

---

## Sources & References

- **Origin document:** `~/printing-press/manuscripts/hackernews/20260427-120911/proofs/20260427-130958-retro-hackernews-pp-cli.md`
- **GitHub issue:** [mvanhorn/cli-printing-press#350](https://github.com/mvanhorn/cli-printing-press/issues/350)
- **Related code:**
  - `internal/generator/generator.go::freshnessCommandPaths` (U1)
  - `internal/generator/generator.go::generate` rendering pipeline (U2)
  - `internal/pipeline/reimplementation_check.go` (U3)
  - `internal/generator/templates/sync.go.tmpl` and `store.go.tmpl` (U4)
  - `internal/generator/templates/` list-endpoint command templates (U5)
  - `internal/pipeline/dogfood.go` (U6)
  - `internal/generator/templates/helpers.go.tmpl` (U7)
  - `skills/printing-press-retro/SKILL.md` (U8)
- **Related published artifact:** [Hackernews regen PR #139](https://github.com/mvanhorn/printing-press-library/pull/139) — the printed CLI whose generation surfaced these findings.
- **Prior recurring findings:** `docs/retros/2026-04-13-recipe-goat-retro.md`, movie-goat retro F8, hackernews v1.3.3 retro F3 (all surfaced the multi-base-URL spec gap that U8's recurrence-cost check is designed to catch).
