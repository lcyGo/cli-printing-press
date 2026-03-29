---
title: "feat: Add .printing-press.json manifest to generated CLIs"
type: feat
status: completed
date: 2026-03-28
origin: docs/brainstorms/2026-03-28-cli-manifest-requirements.md
---

# feat: Add .printing-press.json manifest to generated CLIs

## Overview

Add a `.printing-press.json` manifest file to each generated CLI directory in `~/printing-press/library/`. The manifest captures provenance metadata — when the CLI was built, from what spec, by which version of printing-press — so the folder is self-describing even in isolation.

## Problem Frame

Generated CLIs carry no provenance metadata. The only way to determine when a CLI was created is to inspect filesystem timestamps, which are unreliable across copies, moves, and time. Metadata exists in `manuscripts/` and `.runstate/`, but those are machine-specific and separate from the published CLI. (see origin: docs/brainstorms/2026-03-28-cli-manifest-requirements.md)

## Requirements Trace

**Manifest Structure**
- R1. `.printing-press.json` manifest at root of each generated CLI directory
- R11. `schema_version: 1`

**Required Fields**
- R2. `generated_at` (RFC 3339 timestamp)
- R3. `printing_press_version`
- R4. `api_name`
- R5. `spec_url` and/or `spec_path`
- R6. `spec_format` (openapi3, internal, graphql)
- R7. `cli_name`
- R8. `run_id`
- R9. `catalog_entry` (optional, when spec came from catalog)
- R10. `spec_checksum` (SHA-256)

**Lifecycle**
- R12. Written during the publish phase, not intermediate generation steps

## Scope Boundaries

- No CLI command to read/query manifests
- No regeneration workflow triggered from the manifest
- No backfill for previously generated CLIs
- Write-once at publish time; no update-on-regeneration semantics
- Standalone `generate` command does not write a manifest (it has no publish phase)

## Context & Research

### Relevant Code and Patterns

- `internal/pipeline/publish.go` — `PublishWorkingCLI` is the publish entry point. After `CopyDir(workingDir, finalDir)`, it sets `state.PublishedDir`, calls `state.Save()`, then `WriteRunManifest(state)`. The new `WriteCLIManifest` call should go after `CopyDir` and before `state.Save()`.
- `internal/pipeline/publish.go` — `RunManifest` struct and `BuildRunManifest` show the existing pattern for metadata structs. `WriteCLIManifest` should follow the same `json.MarshalIndent("", "  ")` + `os.WriteFile(path, data, 0o644)` pattern.
- `internal/pipeline/state.go` — `PipelineState` carries `SpecURL`, `SpecPath`, `RunID`, `APIName`. Does not carry `SpecFormat` or `CatalogEntry`.
- `internal/pipeline/fullrun.go` — `copySpecToOutput` writes the spec to `<workingDir>/spec.json`. This file is available at publish time for checksum computation and format detection.
- `internal/cli/root.go:30` — `var version = "0.4.0"` is unexported. Cannot be imported by `internal/pipeline`.
- `internal/catalog/catalog.go` — `LookupFS` can resolve an API name to a catalog `Entry` with `SpecFormat`.
- `internal/naming/naming.go` — `CLI(apiName)` derives the CLI name.
- `internal/openapi/openapi.go` — `IsOpenAPI(data)` detects OpenAPI specs.
- Test patterns: table-driven with `testify/assert`, `t.TempDir()`, `setPressTestEnv` helper in `state_test.go`.

### Institutional Learnings

- Published CLIs in `library/` should be treated as immutable (see `docs/solutions/best-practices/checkout-scoped-printing-press-output-layout-2026-03-28.md`).
- Path logic is centralized in `internal/pipeline/paths.go` and naming in `internal/naming/`.

## Key Technical Decisions

- **Separate struct from RunManifest**: `CLIManifest` is a new struct, not an extension of `RunManifest`. RunManifest contains machine-specific fields (`GitRoot`, `WorkingDir`, `Scope`) that don't belong in a portable CLI manifest. Different purpose, different struct.
- **Compute everything at publish time**: Rather than adding fields to `PipelineState` (which would require a state version bump), compute `spec_format`, `catalog_entry`, and `spec_checksum` from available data at publish time. PipelineState already carries `APIName`, `SpecURL`, `SpecPath`, `RunID`.
- **Version via shared package**: Extract the version var to `internal/version/version.go` so both `internal/cli` and `internal/pipeline` can import it. Cleanest solution without parameter threading.
- **spec_path stored as-is (absolute)**: The `spec_url` is the portable identifier; `spec_path` is supplementary context for the generating machine. No path rewriting.
- **Spec checksum from raw file bytes**: Hash the spec file available in the working directory at publish time. Simple, verifiable with `shasum`, deterministic per run.

## Open Questions

### Resolved During Planning

- **spec_path absolute or relative?** → Store as-is (absolute). URL is the portable identifier. (see origin deferred question affecting R5)
- **Where in publish pipeline?** → After `CopyDir` in `PublishWorkingCLI`, before `state.Save()`. (see origin deferred question affecting R12)
- **Checksum from raw or parsed spec?** → Raw bytes of the spec file in the working directory. (see origin deferred question affecting R10)
- **How to detect spec format at publish time?** → Read `spec.json` from working dir, use `openapi.IsOpenAPI(data)` and `graphql.IsGraphQLSDL(data)` to detect format. Fallback to `"internal"`.

- **Does `copySpecToOutput` write raw bytes or re-marshal?** → It calls `ensureJSON()` (fullrun.go:274) which re-marshals YAML to JSON via `yaml.Unmarshal` + `json.Marshal`. The `spec_checksum` is a fingerprint of the re-marshaled JSON in `spec.json`, not the original source file. Still valid for change detection but will not match `shasum` of YAML source files.

### Deferred to Implementation

(None remaining — all resolved during planning or review.)

## Implementation Units

- [x] **Unit 1: Extract version to shared package**

**Goal:** Make the printing-press version importable from any internal package.

**Requirements:** R3

**Dependencies:** None

**Files:**
- Create: `internal/version/version.go` — with `x-release-please-version` annotation on the `var Version` line
- Modify: `internal/cli/root.go` — import `internal/version` and reference `version.Version`; remove `x-release-please-version` annotation
- Modify: `internal/cli/release_test.go` — update `TestVersionConsistencyAcrossFiles` (line 81 uses `version` directly, must change to `version.Version`), update `TestGoreleaserLdflagsTargetMatchesVersionVar` (asserts `internal/cli.version`, must change to `internal/version.version`), update `TestReleasePleaseAnnotationExists` (reads `root.go`, must change to `internal/version/version.go`)
- Modify: `.goreleaser.yaml` — update ldflags from `internal/cli.version` to `internal/version.version`
- Modify: `AGENTS.md` — update versioning section to list `internal/version/version.go` instead of `internal/cli/root.go`
- Test: `internal/version/version_test.go`

**Approach:**
- Move the `var version` declaration (with `x-release-please-version` annotation) and `debug.ReadBuildInfo()` init logic from `internal/cli/root.go` to a new `internal/version` package.
- Export as `version.Version` (the var) and `version.Get()` (function returning the version string, for callers that prefer a function call).
- `internal/cli/root.go` imports `version.Version` to set cobra's `Version` field — same runtime behavior, different import path.
- Update `.goreleaser.yaml` ldflags to target `internal/version.version` instead of `internal/cli.version`.
- Update `AGENTS.md` versioning section to reference the new file location.

**Patterns to follow:**
- The existing `init()` in `root.go` that reads `debug.ReadBuildInfo()` — replicate this logic in `internal/version/version.go`

**Test scenarios:**
- Happy path: `version.Version` returns the hardcoded version string when no build info is available
- Happy path: `version.Get()` returns the same value as `version.Version`
- Edge case: `TestVersionConsistencyAcrossFiles` still passes after the move (reads `internal/version/version.go` instead of `internal/cli/root.go`)
- Edge case: `TestGoreleaserLdflagsTargetMatchesVersionVar` passes with updated ldflags path (`internal/version.version`)

**Verification:**
- `go build ./...` succeeds
- `go test ./internal/version/ ./internal/cli/` passes
- The binary `--version` output is unchanged

---

- [x] **Unit 2: CLIManifest struct, writer, and helpers**

**Goal:** Define the manifest data model and a standalone function to write `.printing-press.json` to a directory.

**Requirements:** R1, R2, R3, R4, R5, R6, R7, R8, R9, R10, R11

**Dependencies:** Unit 1 (for `version.Version`)

**Files:**
- Create: `internal/pipeline/climanifest.go`
- Test: `internal/pipeline/climanifest_test.go`

**Approach:**
- Define `CLIManifest` struct with JSON tags matching the brainstorm field names: `schema_version`, `generated_at`, `printing_press_version`, `api_name`, `cli_name`, `spec_url`, `spec_path`, `spec_format`, `spec_checksum`, `run_id`, `catalog_entry`.
- `WriteCLIManifest(dir string, m CLIManifest) error` — marshals to indented JSON and writes to `filepath.Join(dir, ".printing-press.json")`. Follow the `WriteRunManifest` pattern: `json.MarshalIndent` with `""` prefix and `"  "` indent, `0o644` perms, `fmt.Errorf` wrapping.
- `specChecksum(path string) (string, error)` — reads file bytes, returns `"sha256:<hex>"`. Returns empty string (not error) if file doesn't exist.
- `detectSpecFormat(data []byte) string` — uses `openapi.IsOpenAPI` and `graphql.IsGraphQLSDL`, falls back to `"internal"`.
- `CLIManifestFilename` constant = `".printing-press.json"`.

**Patterns to follow:**
- `WriteRunManifest` / `WriteArchivedManifest` in `internal/pipeline/publish.go` for the JSON write pattern
- `openapi.IsOpenAPI` for format detection

**Test scenarios:**
- Happy path: `WriteCLIManifest` creates `.printing-press.json` with all fields correctly serialized as indented JSON
- Happy path: `specChecksum` returns correct `sha256:<hex>` for a known input
- Happy path: `detectSpecFormat` returns `"openapi3"` for an OpenAPI spec, `"graphql"` for a GraphQL SDL, `"internal"` for an internal spec
- Edge case: `WriteCLIManifest` with optional fields (`catalog_entry`, `spec_path`) omitted — JSON uses `omitempty` correctly
- Edge case: `specChecksum` returns empty string for nonexistent file
- Edge case: `WriteCLIManifest` target directory doesn't exist — returns error
- Happy path: `WriteCLIManifest` always sets `schema_version` to 1

**Verification:**
- `go test ./internal/pipeline/ -run CLIManifest` passes
- Written JSON is valid and parseable
- Field names match the brainstorm spec exactly

---

- [x] **Unit 3: Wire manifest into PublishWorkingCLI**

**Goal:** Build and write the CLI manifest as part of the publish flow so every published CLI gets a `.printing-press.json`.

**Requirements:** R12, all R1-R11 (end-to-end)

**Dependencies:** Unit 2

**Files:**
- Modify: `internal/pipeline/publish.go` — add manifest build + write call inside `PublishWorkingCLI`
- Test: `internal/pipeline/climanifest_test.go` (add integration-style test)

**Approach:**
- In `PublishWorkingCLI`, after `CopyDir(workingDir, finalDir)` and setting `state.PublishedDir`, build a `CLIManifest`:
  - `schema_version`: 1
  - `generated_at`: `time.Now().UTC()`
  - `printing_press_version`: `version.Version`
  - `api_name`: `state.APIName`
  - `cli_name`: `naming.CLI(state.APIName)`
  - `spec_url`: `state.SpecURL`
  - `spec_path`: `state.SpecPath`
  - `spec_format`: detect from spec file in `state.EffectiveWorkingDir()` (note: `spec.json` only exists when `specFlag` is `--spec` per fullrun.go:267; for `--docs` runs, no spec.json exists and these fields will be empty)
  - `spec_checksum`: compute from spec file in `state.EffectiveWorkingDir()` (same caveat)
  - `run_id`: `state.RunID`
  - `catalog_entry`: look up `state.APIName` in `catalog.LookupFS` (empty string if not found)
- Call `WriteCLIManifest(finalDir, manifest)`
- If the manifest write fails, return the error (don't silently ignore — a missing manifest is a bug, not a recoverable condition)

**Patterns to follow:**
- The existing `WriteRunManifest(state)` call in `PublishWorkingCLI` — similar positioning, similar error handling

**Test scenarios:**
- Happy path: After `PublishWorkingCLI`, `finalDir/.printing-press.json` exists and contains correct `api_name`, `run_id`, `schema_version`, `generated_at`, `printing_press_version`, and `cli_name`
- Happy path: `spec_checksum` in manifest matches independently computed checksum of the spec file
- Happy path: When API name matches a catalog entry, `catalog_entry` is populated
- Edge case: When API name doesn't match a catalog entry, `catalog_entry` is omitted from JSON
- Edge case: When spec file is missing from working dir, `spec_checksum` and `spec_format` are empty but manifest is still written
- Integration: Published CLI directory is fully usable (existing publish behavior unchanged) with the addition of `.printing-press.json`

**Verification:**
- `go test ./internal/pipeline/` passes
- Running `MakeBestCLI` (via `FULL_RUN=1` integration test) produces a CLI directory containing `.printing-press.json`

## System-Wide Impact

- **Interaction graph:** Only `PublishWorkingCLI` is modified. No callbacks, middleware, or observers affected. The `generate` command (standalone) is explicitly out of scope — it does not publish to library/.
- **Error propagation:** Manifest write failure fails the publish operation. This is intentional — a publish without a manifest is incomplete.
- **State lifecycle risks:** None. No changes to `PipelineState` schema. The manifest is written to the output directory, not to runstate.
- **API surface parity:** The manifest is a new file in the output directory. No existing consumers read `.printing-press.json`, so no breakage risk.
- **Unchanged invariants:** `RunManifest` (in runstate and manuscripts) is untouched. The version string embedded in generated `root.go` via templates is untouched. The NOTICE file is untouched.

## Risks & Dependencies

| Risk | Mitigation |
|------|------------|
| `copySpecToOutput` may re-marshal the spec (checksum differs from original file) | Document which bytes are checksummed. The checksum is still a valid fingerprint for change detection. |
| `catalog.LookupFS` may not find entries for all API names | `catalog_entry` is optional (`omitempty`). Missing is fine. |
| Moving version to shared package could break build info injection via ldflags | Verify goreleaser ldflags path is updated to target `internal/version.version` |

## Sources & References

- **Origin document:** [docs/brainstorms/2026-03-28-cli-manifest-requirements.md](docs/brainstorms/2026-03-28-cli-manifest-requirements.md)
- Related code: `internal/pipeline/publish.go` (PublishWorkingCLI, RunManifest pattern)
- Related code: `internal/pipeline/fullrun.go` (MakeBestCLI, copySpecToOutput)
- Related code: `internal/cli/root.go` (version var, generate command)
- Institutional learning: `docs/solutions/best-practices/checkout-scoped-printing-press-output-layout-2026-03-28.md`
