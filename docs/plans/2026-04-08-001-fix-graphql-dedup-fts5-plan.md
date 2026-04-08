---
title: "fix: GraphQL type dedup, usageErr emission, and FTS5 manual sync"
type: fix
status: completed
date: 2026-04-08
origin: docs/retros/2026-04-08-linear-retro.md
---

# fix: GraphQL type dedup, usageErr emission, and FTS5 manual sync

## Overview

Three bugs surfaced during the Linear (GraphQL) CLI generation that affect every future generation. Two cause compile failures on GraphQL APIs, and one causes FTS5 search to fail silently on any API where FTS fields aren't extracted table columns.

## Problem Frame

When generating a CLI from Linear's 43k-line GraphQL schema:
1. `go vet` failed because the types.go template emitted duplicate struct fields (pagination args like `after`, `before`, `first` appeared multiple times on entity types)
2. `go build` failed because promoted commands reference `usageErr()` which is conditionally emitted in helpers.go based on `HasMultiPositional` - a flag that is false for most GraphQL APIs
3. During live testing, FTS5 search failed with "no such column: id" because the manual FTS sync fallback in store.go.tmpl uses `DELETE FROM fts WHERE id = ?`, which is incompatible with modernc.org/sqlite's FTS5 implementation

All three are generator-level bugs that will recur on every future generation unless fixed in the machine.

## Requirements Trace

- R1. GraphQL SDL parsing must produce Go types without duplicate struct fields
- R2. Generated CLIs must compile without manual intervention regardless of `HasMultiPositional` state
- R3. FTS5 search must work correctly in the manual (non-trigger) fallback path using modernc.org/sqlite

## Scope Boundaries

- Does not include the broader GraphQL template system (sync, client, promoted examples) - that is WU-3 in the retro
- Does not redesign the store schema or FTS architecture
- Does not add new GraphQL-specific templates

## Context & Research

### Relevant Code and Patterns

- `internal/graphql/parser.go:574-587` - `buildTypeDef()` collects fields without dedup
- `internal/generator/templates/helpers.go.tmpl:97-98` - `usageErr` behind `{{if .HasMultiPositional}}`
- `internal/generator/templates/command_promoted.go.tmpl` - calls `usageErr` unconditionally
- `internal/generator/generator.go` - `HasMultiPositional` set when positionalCount >= 2
- `internal/generator/templates/store.go.tmpl:312-324` - manual FTS fallback with broken DELETE
- `internal/generator/schema_builder.go:85-88` - `FTS5Triggers` only true when all FTS fields are extracted columns
- `internal/generator/templates/store.go.tmpl:91-113` - trigger-based FTS (already correct)

### Key Insight: FTS5 Trigger Path Already Works

The generator already has two FTS5 paths:
1. **Trigger-based** (`FTS5Triggers=true`): Content-linked FTS5 with AFTER INSERT/UPDATE/DELETE triggers. This path works correctly.
2. **Manual** (`FTS5Triggers=false`): Direct DELETE/INSERT in Upsert methods. This path has the modernc.org/sqlite bug.

The trigger path is used when all FTS fields are extracted columns. The manual path is used when FTS fields live inside the JSON data column. The bug only affects the manual path.

## Key Technical Decisions

- **Dedup in parser, not template**: Deduplication belongs in `buildTypeDef()` because the parser knows the semantic context. The template should not need to reason about duplicates.
- **Always emit usageErr**: The function is 1 line and harmless if unused. Removing the conditional is simpler and more robust than computing when it's needed.
- **Fix manual FTS delete syntax**: Use the FTS5-specific delete command (`INSERT INTO fts(fts, ...) VALUES('delete', ...)`) instead of the standard SQL DELETE which doesn't work on FTS5 virtual tables in modernc.org/sqlite.

## Open Questions

### Resolved During Planning

- **Q: Should we switch all FTS to trigger-based?** No. The manual path exists for a reason - when FTS fields aren't extracted columns (they're inside the JSON data blob). Both paths should work. Fix the manual path's SQL syntax.
- **Q: Is `HasMultiPositional` used elsewhere?** It gates `usageErr` only. Removing the guard has no other side effects.

### Deferred to Implementation

- Exact row-tracking mechanism for the FTS5 manual delete. The implementer needs to verify whether modernc.org/sqlite supports the `INSERT INTO fts(fts, rowid, ...) VALUES('delete', ...)` syntax or needs another approach.

## Implementation Units

- [ ] **Unit 1: Deduplicate fields in buildTypeDef**

**Goal:** Prevent duplicate struct fields in generated types.go for GraphQL APIs

**Requirements:** R1

**Dependencies:** None

**Files:**
- Modify: `internal/graphql/parser.go`
- Test: `internal/graphql/parser_test.go`

**Approach:**
- In `buildTypeDef()`, track seen field names with a `map[string]bool`
- Skip fields whose name has already been seen (first-wins)
- Also apply the same dedup to `addSupportTypes()` which recursively builds types through the same `buildTypeDef` function (already covered)

**Patterns to follow:**
- Existing `buildTypeDef` structure at line 574

**Test scenarios:**
- Happy path: Parse a schema with unique fields per type -> all fields present in output
- Edge case: Parse a schema where a type has `after` field in both entity fields and pagination args -> only one `After` field in the Go struct
- Edge case: Parse Linear's full schema.graphql -> verify no TypeDef has duplicate field names (use a loop over all types)
- Regression: Existing `TestParseSDLContent` still passes

**Verification:**
- `go test ./internal/graphql/...` passes
- Generate from Linear schema -> `go vet ./...` succeeds without types redeclaration errors

- [ ] **Unit 2: Always emit usageErr in helpers.go**

**Goal:** Ensure promoted commands compile regardless of HasMultiPositional state

**Requirements:** R2

**Dependencies:** None (can be done in parallel with Unit 1)

**Files:**
- Modify: `internal/generator/templates/helpers.go.tmpl`
- Test: `internal/generator/generator_test.go`

**Approach:**
- Remove the `{{if .HasMultiPositional}}` / `{{end}}` guard around the `usageErr` function definition
- The function becomes unconditionally emitted alongside `notFoundErr`, `authErr`, `apiErr`, etc.

**Patterns to follow:**
- `notFoundErr`, `authErr`, `apiErr`, `configErr`, `rateLimitErr` are all emitted unconditionally on the lines immediately below

**Test scenarios:**
- Happy path: Generate from a spec with no multi-positional endpoints -> helpers.go contains `usageErr` function
- Happy path: Generate from a spec with multi-positional endpoints -> helpers.go still contains `usageErr` (no regression)
- Integration: Generate from a spec with promoted commands -> `go build` succeeds

**Verification:**
- `go test ./internal/generator/...` passes
- Generate any CLI with promoted commands -> grep helpers.go for `func usageErr` -> present

- [ ] **Unit 3: Fix FTS5 manual sync for modernc.org/sqlite**

**Goal:** Make the manual FTS fallback path work correctly with the pure-Go SQLite driver

**Requirements:** R3

**Dependencies:** None (can be done in parallel with Units 1-2)

**Files:**
- Modify: `internal/generator/templates/store.go.tmpl`
- Modify: `internal/generator/templates/store.go.tmpl` (also the `upsertGenericResourceTx` section around line 147-160 which has the same pattern for `resources_fts`)

**Approach:**
- Replace `DELETE FROM {{.Name}}_fts WHERE id = ?` with the FTS5 delete command. Two options to evaluate during implementation:
  - Option A: `INSERT INTO {{.Name}}_fts({{.Name}}_fts, id, fields...) VALUES('delete', ?, fields...)` - requires knowing the old field values
  - Option B: Drop and rebuild the FTS entry using `DELETE FROM {{.Name}}_fts WHERE rowid = (SELECT rowid FROM {{.Name}} WHERE id = ?)` - only works if the content table has the same rowid
  - Option C: Switch the manual path to also use content-linked FTS (`content='tablename'`) with explicit rebuild commands rather than triggers. This avoids the DELETE problem entirely by using `INSERT INTO fts(fts) VALUES('rebuild')` after each upsert batch.
- Apply the same fix to `upsertGenericResourceTx` for the `resources_fts` table
- The trigger-based path (FTS5Triggers=true) is already correct and should not be changed

**Patterns to follow:**
- The trigger-based FTS path at lines 91-113 of store.go.tmpl shows the correct modernc.org/sqlite-compatible approach

**Test scenarios:**
- Happy path: Generate a CLI with FTS-enabled tables where FTS fields are NOT extracted columns (manual path) -> sync data -> `Search()` returns results
- Happy path: Generate a CLI with FTS-enabled tables where FTS fields ARE extracted columns (trigger path) -> sync data -> `Search()` still works (no regression)
- Error path: Verify no "no such column" or "SQL logic error" warnings during sync with FTS enabled
- Edge case: Upsert the same entity twice -> FTS index has exactly one entry (not duplicated)

**Verification:**
- Generate a test CLI -> sync sample data -> SearchIssues returns expected results
- No FTS-related warnings in stderr during sync

## System-Wide Impact

- **Interaction graph:** These fixes affect the generator output only. No changes to the printing-press binary's runtime behavior, CLI commands, or skill instructions.
- **Error propagation:** Unit 1 and 2 fix compile-time errors. Unit 3 fixes a runtime search failure. All are isolated to generated CLI code.
- **Unchanged invariants:** The trigger-based FTS path (FTS5Triggers=true) is not modified. REST API generation is not modified. The store schema structure is not modified.

## Risks & Dependencies

| Risk | Mitigation |
|------|------------|
| FTS5 delete syntax varies between SQLite implementations | Test with modernc.org/sqlite specifically; the trigger path is already proven |
| Removing usageErr guard might cause dead-code warnings | usageErr is referenced by all promoted commands, so it will never be dead code when promoted commands exist |
| Field dedup might drop a legitimately needed field | First-wins strategy preserves the primary field; pagination args (after, before, first, last) are always secondary |

## Sources & References

- **Origin document:** [docs/retros/2026-04-08-linear-retro.md](docs/retros/2026-04-08-linear-retro.md)
- Related code: `internal/graphql/parser.go`, `internal/generator/templates/helpers.go.tmpl`, `internal/generator/templates/store.go.tmpl`
- modernc.org/sqlite FTS5 behavior: content-linked tables with triggers are the recommended approach
