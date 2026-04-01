---
title: "fix: Steam retro improvements — scorer bugs, cache poisoning, template defaults"
type: fix
status: active
date: 2026-03-31
origin: docs/retros/2026-03-31-steam-retro.md
---

# Fix: Steam Retro Improvements

## Overview

Fix 7 issues found during the Steam CLI generation retro. Two are scorer bugs (verify and dogfood report false failures), one is a runtime bug (dry-run cache poisoning), and four are generator template improvements. The scorer fixes are highest priority — they distort quality signals for every future CLI.

## Problem Frame

The Steam CLI scored 71/100 and 67% verify, but analysis showed ~30% of the score loss came from scoring tool bugs, not CLI defects. Verify derives command names from Go function names instead of actual cobra `Use:` fields (25 false failures). Dogfood skips `root.go` when checking for flag usage (6 false positives). A runtime bug causes dry-run responses to poison the response cache. Several generator templates emit suboptimal defaults.

(see origin: docs/retros/2026-03-31-steam-retro.md)

## Requirements Trace

- R1. Verify discovers command names from ground truth (cobra `Use:` field or `--help` output), not from Go function names
- R2. Dogfood dead-flag detection includes `root.go` flag reads while excluding declarations
- R3. Dry-run responses are never written to or read from the response cache
- R4. Root command `Short:` field defaults to a CLI-appropriate description, not raw spec text
- R5. Promoted commands show help when invoked with no positional args (exit 0, not exit 2)
- R6. Helper functions are emitted conditionally based on spec features

## Scope Boundaries

- Not changing how the generator names commands (the generator is correct)
- Not fixing sync path resolution (WU-5 from retro — needs separate design work)
- Not adding response envelope detection (retro finding #9 — deferred)
- Not adding query-param auth inference (retro finding #5 — deferred)
- Not changing the scorecard itself (only verify and dogfood have bugs)

## Context & Research

### Relevant Code and Patterns

**Verify command discovery:**
- `internal/pipeline/runtime.go:250-278` — `discoverCommands()` reads root.go, uses regex `rootCmd\.AddCommand\(new(\w+)Cmd\(` to extract function names, converts via `camelToKebab`
- `internal/pipeline/runtime.go:537-565` — `camelToKebab()` inserts hyphens before uppercase letters only, not before digits
- `internal/pipeline/runtime.go:287-322` — `inferPositionalArgs()` already runs `<binary> <cmd> --help` and parses the Usage line — so help output parsing is a proven pattern

**Dogfood dead-flag detection:**
- `internal/pipeline/dogfood.go:365-405` — `checkDeadFlags()` extracts flags from `&flags\.\w+` in root.go, searches other files for `flags.<name>`, skips root.go at line 385
- `internal/pipeline/dogfood.go:407-455` — `checkDeadFunctions()` uses similar pattern but for function definitions in helpers.go

**Generator templates:**
- `internal/generator/templates/client.go.tmpl:200-212` — `Get()` checks `!c.NoCache` but not `c.DryRun` before cache read/write
- `internal/generator/templates/root.go.tmpl:43` — `Short: "{{oneline .Description}}"` copies spec text verbatim
- `internal/generator/templates/command_endpoint.go.tmpl:39` — `Use: "{{.EndpointName}}{{positionalArgs .Endpoint}}"` sets positional args but Args: constraint is separate
- `internal/generator/templates/helpers.go.tmpl:250` — `replacePathParam()` emitted unconditionally

**Existing test patterns:**
- `internal/pipeline/runtime_test.go` — 3 tests, uses testify/assert with temp dirs and mock CLIs
- `internal/pipeline/dogfood_test.go` — 6 tests including mock root.go/helpers.go fixtures, uses `testify/assert`
- Test naming convention: `TestFunctionName_DescriptiveScenario`

### Institutional Learnings

- AGENTS.md: `go test ./...` before considering work done. Match package's existing style (table-driven with testify/assert)
- AGENTS.md: `gofmt -w ./...` after writing Go code
- Previous retros established the pattern of testing both positive and negative cases for scoring tools

## Key Technical Decisions

- **Verify: Parse `--help` output for command names (Option A from retro)** — `inferPositionalArgs` already parses `--help` output successfully. This is ground-truth and avoids the fragile Go-function-name-to-command-name derivation entirely. Option B (parsing `Use:` from source) would require reading every command file and regex-matching the cobra struct literal.

- **Dogfood: Filter declaration lines instead of skipping the file** — Include root.go in the search but exclude lines matching the declaration pattern `&flags\.<name>`. This is simpler than maintaining a separate "declarations" set and "usage" set.

- **Root description: Static template, not LLM-generated** — Use `"Manage {{.Name}} resources via the {{.Name}} API"` as the default. An LLM-generated description would be better but introduces unpredictability. The skill already tells Claude to rewrite it, so the template just needs a reasonable floor.

## Open Questions

### Resolved During Planning

- **Should verify fall back to camelToKebab if --help parsing fails?** Yes — if the binary can't be executed (broken build), falling back to the old method is better than no discovery at all. The old method works correctly for the majority of commands.

### Deferred to Implementation

- **Which specific helper functions should get conditional flags?** Needs reading the full helpers.go.tmpl to audit each function. The retro identified `replacePathParam` and `usageErr` but there may be others.
- **Does `HasDelete` flag computation have a bug?** The retro noted `classifyDeleteError` was emitted despite no DELETE endpoints. Need to trace the flag computation in `generator.go`.

## Implementation Units

- [ ] **Unit 1: Fix verify command name derivation**

**Goal:** Verify discovers command names from `--help` output instead of Go function names

**Requirements:** R1

**Dependencies:** None

**Files:**
- Modify: `internal/pipeline/runtime.go`
- Test: `internal/pipeline/runtime_test.go`

**Approach:**
- In `discoverCommands()`, after extracting the list of function names (keep this as fallback), run `<binary> --help` once
- Parse the top-level command list from help output. Cobra's `--help` outputs commands in an `Available Commands:` section, one per line, with the command name as the first word
- Build a map of discovered commands from help output
- If help parsing succeeds and finds commands, use those names. If it fails (binary doesn't exist or crashes), fall back to the existing `camelToKebab` derivation
- The binary path is already known — `runCommandTests` receives it, and `discoverCommands` can accept it as a parameter or the caller can do the help parse

**Patterns to follow:**
- `inferPositionalArgs()` at runtime.go:287 already runs `<binary> <cmd> --help` with exec.Command and 10-second timeout — reuse the same pattern
- Keep the existing `camelToKebab` path as fallback, not a replacement

**Test scenarios:**
- Happy path: Mock binary that outputs `Available Commands:\n  iecon-items-440  description\n  player  description` → discovers `iecon-items-440` (with hyphen before digit)
- Happy path: Standard commands like `isteam-user` → discovered correctly
- Edge case: Binary doesn't exist → falls back to camelToKebab derivation
- Edge case: Binary crashes on --help → falls back gracefully
- Edge case: Empty Available Commands section → falls back

**Verification:**
- `go test ./internal/pipeline/... -run TestDiscoverCommands` passes
- Generate from Steam spec, run verify → 25 previously-failing numeric-suffix commands now pass

---

- [ ] **Unit 2: Fix dogfood dead-flag false positives**

**Goal:** Dogfood includes root.go flag reads in dead-flag detection while excluding declarations

**Requirements:** R2

**Dependencies:** None

**Files:**
- Modify: `internal/pipeline/dogfood.go`
- Test: `internal/pipeline/dogfood_test.go`

**Approach:**
- In `checkDeadFlags()`, remove the `root.go` skip at line 385
- Instead, when scanning root.go, filter out lines that match the declaration pattern `&flags\.` — these are flag registrations, not usage
- Lines like `if flags.agent {` and `c.NoCache = f.noCache` should be counted as usage
- Lines like `BoolVar(&flags.agent, ...)` should not be counted as usage

**Patterns to follow:**
- Existing `checkDeadFunctions()` which uses regex to match call patterns — similar filtering approach

**Test scenarios:**
- Happy path: Mock root.go with `BoolVar(&flags.agent, ...)` and `if flags.agent {` → `agent` NOT reported dead
- Happy path: Mock root.go with `BoolVar(&flags.unused, ...)` but no read → `unused` IS reported dead
- Happy path: All 6 standard flags (agent, noCache, noInput, rateLimit, timeout, yes) with standard usage patterns → none reported dead
- Edge case: Flag name appears only in a comment → should still be reported dead (string match on non-comment lines)

**Verification:**
- `go test ./internal/pipeline/... -run TestRunDogfood` passes (existing test may need fixture updates)
- Run dogfood on any generated CLI → 0 false dead-flag warnings for the 6 standard flags

---

- [ ] **Unit 3: Fix dry-run cache poisoning**

**Goal:** Dry-run responses are never cached

**Requirements:** R3

**Dependencies:** None

**Files:**
- Modify: `internal/generator/templates/client.go.tmpl`

**Approach:**
- In the `Get()` method (lines 200-212), add `!c.DryRun` to both the cache-read and cache-write guards
- Before: `if !c.NoCache && c.cacheDir != ""`
- After: `if !c.NoCache && !c.DryRun && c.cacheDir != ""`
- Apply to both the cache read check (line 202) and the cache write check (line 209)

**Patterns to follow:**
- The `c.NoCache` guard pattern already exists — `c.DryRun` is the same shape

**Test scenarios:**
- Happy path: `Get()` with `DryRun=true` → cache is not read, cache is not written
- Happy path: `Get()` with `DryRun=false` → cache works normally (read and write)
- Integration: Run dry-run then real call → real call returns fresh API data, not the dry-run stub

**Test expectation: Template change — tested implicitly via generated CLI tests.** The template itself doesn't have unit tests; validation is that the generated `client.go` contains the `!c.DryRun` guard in both cache paths.

**Verification:**
- Generated `client.go` contains `!c.DryRun` in both cache guards
- Regenerate a test CLI → `go build`, `go vet` pass

---

- [ ] **Unit 4: Improve root command description template**

**Goal:** Root command `Short:` defaults to a CLI-appropriate description

**Requirements:** R4

**Dependencies:** None

**Files:**
- Modify: `internal/generator/templates/root.go.tmpl`

**Approach:**
- Replace `Short: "{{oneline .Description}}"` with `Short: "Manage {{.Name}} resources via the {{.Name}} API"`
- The `{{.Name}}` template variable is the API name (e.g., "steam-web", "stripe", "notion")
- This produces "Manage steam-web resources via the steam-web API" — generic but correct
- The `/printing-press` skill already instructs Claude to rewrite this, so the template is a floor, not a ceiling

**Patterns to follow:**
- Other template fields use `{{.Name}}` — consistent

**Test scenarios:**
- Happy path: Generate from any spec → root Short does not contain markdown links, raw API descriptions, or auth instructions
- Happy path: API name "stripe" → Short is "Manage stripe resources via the stripe API"
- Edge case: Empty spec description → Short is still the template default (no empty string)

**Verification:**
- Generate a test CLI → `--help` output shows the templated description
- No markdown links in the Short field

---

- [ ] **Unit 5: Help-guard pattern for promoted commands**

**Goal:** Promoted commands show help instead of erroring when invoked with no args

**Requirements:** R5

**Dependencies:** None

**Files:**
- Modify: `internal/generator/templates/command_endpoint.go.tmpl`
- Modify: `internal/generator/generator.go` (if the template needs a new data field for required arg count)

**Approach:**
- In the promoted command template, replace the `Args: cobra.ExactArgs(N)` line with a help-guard at the top of `RunE`
- Need to determine where `Args:` is set for promoted commands — it may be in the template itself or computed by the generator
- The guard pattern: `if len(args) < {{.RequiredArgCount}} { return cmd.Help() }`
- For non-promoted (raw) commands, keep `Args:` as-is — those are direct API wrappers where an error message is appropriate

**Patterns to follow:**
- The polish skill already applies this pattern manually — making the generator do it automatically

**Test scenarios:**
- Happy path: Generated promoted command with 1 required arg → `<cli> <cmd>` (no args) shows help, exits 0
- Happy path: Generated promoted command with 2 required args → `<cli> <cmd> arg1` (1 arg) shows help, exits 0
- Happy path: Generated promoted command with args provided → runs normally
- Edge case: Non-promoted (raw) command → still uses `Args:` pattern (no change)

**Verification:**
- Generate a test CLI with promoted commands → invoke with no args → help output, exit 0
- Verify still runs `--help` and `--dry-run` tests successfully on generated promoted commands

---

- [ ] **Unit 6: Conditional helper function emission**

**Goal:** Helper functions only emitted when the spec needs them

**Requirements:** R6

**Dependencies:** None

**Files:**
- Modify: `internal/generator/templates/helpers.go.tmpl`
- Modify: `internal/generator/generator.go` (HelperFlags struct)

**Approach:**
- Audit `helpers.go.tmpl` for unconditionally emitted functions
- Add flags to `HelperFlags` struct: `HasPathParams`, `HasPagination` (if not already present)
- Gate `replacePathParam()` behind `{{if .HasPathParams}}`
- Investigate the `HasDelete` flag computation — the retro reported `classifyDeleteError` was emitted despite no DELETE endpoints. Check if the flag is correctly computed in `generator.go`
- Leave truly universal helpers (truncate, newTabWriter, printOutputWithFlags, etc.) unconditional

**Patterns to follow:**
- Existing `{{if .HasDelete}}` conditional pattern for `classifyDeleteError`

**Test scenarios:**
- Happy path: Spec with no DELETE endpoints → `classifyDeleteError` not emitted
- Happy path: Spec with DELETE endpoints → `classifyDeleteError` emitted
- Happy path: Spec with no path params → `replacePathParam` not emitted
- Happy path: Spec with path params → `replacePathParam` emitted
- Edge case: Spec with only one path param endpoint → `replacePathParam` still emitted (threshold is any, not many)

**Verification:**
- Generate from Petstore spec → grep generated helpers.go for conditionally-emitted functions
- `go build`, `go vet` pass on generated CLI
- Dogfood reports fewer dead functions

## System-Wide Impact

- **Verify change (Unit 1):** Changes how all future CLIs are verified. The fallback to `camelToKebab` ensures no regression for CLIs where --help can't be parsed. No change to verify's JSON output schema — same `CommandResult` struct.
- **Dogfood change (Unit 2):** Changes dead-flag reporting for all future CLIs. Existing dogfood test fixtures may need updating if they relied on specific dead-flag counts.
- **Template changes (Units 3-6):** Only affect newly generated CLIs. Existing CLIs in the library are unaffected (they're already generated).
- **Error propagation:** No new error paths. Verify fallback is additive. Dogfood filter is narrowing (fewer false positives, not more).
- **Unchanged invariants:** Verify's `CommandResult` JSON schema, dogfood's `DeadCodeResult` schema, generator's `APISpec` struct, profiler's output — all unchanged.

## Risks & Dependencies

| Risk | Mitigation |
|------|------------|
| Verify's `--help` parsing breaks on unusual cobra configurations | Fall back to `camelToKebab` — the old behavior is preserved as fallback |
| Dogfood's declaration filter is too aggressive (filters real usage lines) | Test with fixtures that include both declarations and reads in root.go |
| Template changes break existing test fixtures | Run `go test ./...` before and after — test fixtures should still pass |
| `HasDelete` flag bug may be in the OpenAPI parser, not the generator | Trace the flag computation during implementation — may need a parser fix too |

## Sources & References

- **Origin document:** [docs/retros/2026-03-31-steam-retro.md](docs/retros/2026-03-31-steam-retro.md)
- Verify runtime: `internal/pipeline/runtime.go:250-565`
- Dogfood detection: `internal/pipeline/dogfood.go:365-455`
- Generator templates: `internal/generator/templates/`
- Profiler: `internal/profiler/profiler.go:70-229`
- Existing verify tests: `internal/pipeline/runtime_test.go`
- Existing dogfood tests: `internal/pipeline/dogfood_test.go`
