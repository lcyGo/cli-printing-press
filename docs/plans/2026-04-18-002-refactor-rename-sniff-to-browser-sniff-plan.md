---
title: Rename `sniff` to `browser-sniff`
type: refactor
status: active
date: 2026-04-18
deepened: 2026-04-18
---

# Rename `sniff` to `browser-sniff`

## Overview

Rename the `sniff` subcommand, its backing Go package, and all live references (skills, AGENTS.md glossary, README) to `browser-sniff`. The current name is misleading — the pair (`sniff` vs. `crowd-sniff`) hides the actual distinction: `sniff` analyzes a HAR file captured via a browser workflow, while `crowd-sniff` scrapes npm/GitHub. Renaming to `browser-sniff` makes the workflow explicit and keeps the pair parallel in tooling (browser-driven vs. community-sourced).

The rename is live-only. Historical docs (past retros, brainstorms, plans, CHANGELOG, solutions) keep their original wording — those describe what existed at the time.

## Problem Frame

`sniff` is overloaded. In AGENTS.md the glossary even conflates `crowd sniff / sniff` in one row ("scrapes npm, PyPI, and GitHub…"), which is wrong — that's only `crowd-sniff`. The SKILL.md uses "Sniff Gate", "Pre-Sniff Auth Intelligence", and `SNIFF_TARGET_URL` while the underlying mechanism is specifically a browser-driven HAR capture. A new contributor reading the glossary can't tell what `sniff` actually does without reading the code.

## Requirements Trace

- R1. `printing-press browser-sniff` is the sole subcommand for this workflow. `printing-press sniff` is removed entirely — no alias, no deprecation period.
- R2. AGENTS.md glossary accurately describes both discovery techniques and uses the new name.
- R3. The printing-press skill (SKILL.md + references) uses `browser-sniff` consistently, including phase names, marker files, env vars, and reference file names.
- R4. README and other live user-facing docs use the new name.
- R5. Internal Go package and symbol names reflect the new name where the rename is cheap; where the rename is expensive and purely internal, stay consistent within the file but don't churn unrelated code.
- R6. Historical documents (retros, brainstorms, past plans, CHANGELOG, solution writeups) are untouched.
- R7. `go build`, `go test ./...`, and `golangci-lint run ./...` pass after the rename.

## Scope Boundaries

- Not renaming `crowd-sniff` — its name already names its data source accurately.
- Not renaming the `--spec-source=sniffed` provenance value. That tag describes a spec's origin (live traffic vs. docs vs. official), not the command. Changing it would affect generated-CLI rate-limiting defaults and require a provenance-value migration that's out of scope for a naming cleanup.
- Not rewriting historical docs under `docs/retros/`, `docs/brainstorms/`, `docs/plans/`, `docs/solutions/`, or `CHANGELOG.md`. Those are immutable records — any dated filename signals "captured at this point in time" and stays as-is. Prose inside a live solution doc may be updated where the doc is forward-pointing reference material, but filenames never change.
- Not renaming git branches or prior commit messages.
- Not migrating in-flight runstate artifacts (`sniff-gate.json`, `sniff-report.md`) on disk. Any user with a mid-flight run across the upgrade boundary must re-run Phase 1.7 to produce the renamed artifacts. Runstate is transient and this is documented in release notes.
- Not preserving the old `sniff` command as an alias. Pre-1.0 + internal tool + all in-repo skill invocations updated in the same change set = clean break is safer than compatibility theater.

### Deferred to Separate Tasks

- Reconsidering `--spec-source=sniffed` → `--spec-source=browser-sniffed`: separate task, affects generated CLIs and provenance provenance metadata written into `.printing-press.json`.
- Renaming `internal/websniff` package if Unit 2 is deferred (see Key Technical Decisions).

## Context & Research

### Relevant Code and Patterns

- `internal/cli/sniff.go` — the cobra command, `newSniffCmd()`, `Use: "sniff"`. Imports `internal/websniff`.
- `internal/cli/sniff_test.go` — tests against the command name and flags.
- `internal/cli/root.go:51` — `rootCmd.AddCommand(newSniffCmd())`.
- `internal/cli/crowd_sniff.go` — the sibling command; do not rename, but verify no cross-references break.
- `internal/websniff/` — package consumed by three live Go files: `internal/cli/sniff.go`, `internal/cli/crowd_sniff.go` (imports `websniff` and calls `websniff.WriteSpec` at line 150), and `internal/generator/generator.go` (uses `websniff.FixtureSet`). Plus comment-level references in `internal/crowdsniff/aggregate.go:10` and `internal/crowdsniff/specgen.go:119`.
- `internal/pipeline/research.go:288-292` — reads `sniff-report.md` from the discovery manuscript directory. This filename is a skill-produced artifact that crosses the Go-code / skill boundary.
- `internal/generator/templates/root.go.tmpl:115` — templated rate-limit default guarded on `{{- if eq .SpecSource "sniffed"}}`. Provenance value, out of scope.
- `skills/printing-press/SKILL.md` — 84 occurrences across phase names (Phase 1.6 Pre-Sniff Auth Intelligence, Phase 1.7 Sniff Gate, Phase 1.8 Crowd Sniff Gate), marker file (`sniff-gate.json`), env var (`SNIFF_TARGET_URL`), reference links.
- `skills/printing-press/references/sniff-capture.md` — 59 occurrences; the entire file is about the browser capture workflow.
- `skills/printing-press/references/crowd-sniff.md` — 8 occurrences; may link back to the capture reference.
- `skills/printing-press-retro/SKILL.md` — retro skill surfaces sniff as a retro topic.
- `AGENTS.md:62` — glossary row conflates the two and is factually wrong about `sniff`.
- `README.md` — lines 16, 165, 172, 176, 221, 223 describe sniff as a user-facing capability.

### Institutional Learnings

- `docs/solutions/best-practices/sniff-and-crowd-sniff-complementary-discovery-2026-03-30.md` — the canonical writeup of the two-technique discovery design. This one should be updated to reflect the new name since it's forward-pointing reference material, not a dated retro snapshot.
- `docs/solutions/best-practices/adaptive-rate-limiting-sniffed-apis.md` — describes runtime behavior for sniffed APIs. The filename uses "sniffed" as an adjective for the provenance tag (not the command), so leave it alone.

## Key Technical Decisions

- **D1: Clean break, no alias.** The `sniff` command is removed, not preserved as an alias. Justification: the only in-repo consumers (skill bash examples, retro skill references) are updated in the same change set (Unit 3). No external consumers have been identified. Pre-1.0 + internal tool = breakage is acceptable. An alias would create a contradiction with the `refactor(cli)!:` breaking-change commit and add ongoing maintenance cost for zero verified benefit.
- **D2: Rename `internal/websniff` → `internal/browsersniff`.** The package has three live consumers (`internal/cli/sniff.go`, `internal/cli/crowd_sniff.go`, `internal/generator/generator.go`) plus two comment-level references in `internal/crowdsniff/`. The rename is mechanical and keeps the codebase coherent with the user-facing name. Bounded friction handling: one retry round on build failure, then revert Unit 2 and defer.
- **D3: Rename skill-authored artifacts: `SNIFF_TARGET_URL` → `BROWSER_SNIFF_TARGET_URL`, `sniff-gate.json` → `browser-sniff-gate.json`, and `sniff-report.md` → `browser-sniff-report.md`.** The first two are read only by the skill. The third is written by the skill inside the discovery manuscript and *read by Go code* (`internal/pipeline/research.go` via `ParseDiscoveryPages`), so the rename must be coordinated across both sides in Unit 3. No read-side fallback for the old filename — consistent with the clean-break alias decision (D1). Marker file and report file renames are "forward only": existing published manuscripts containing `sniff-report.md` cannot be re-ingested by post-rename code; this is accepted, documented in release notes, and surfaces as a known edge case if users attempt to re-run emboss or retro flows against pre-rename manuscripts.
- **D4: Phase names in SKILL.md stay numbered (1.6, 1.7, 1.8) but get their prose relabeled.** "Sniff Gate" → "Browser-Sniff Gate", "Pre-Sniff Auth Intelligence" → "Pre-Browser-Sniff Auth Intelligence". Long, but matches the pattern we're setting.
- **D5: Leave `--spec-source=sniffed` alone for now, but acknowledge the drift.** The adjective there means "derived from captured traffic" and survives the command rename in meaning. However, after this plan lands, the only producer path that writes `sniffed` will be the renamed `browser-sniff` command — so the provenance value lags its producer. A future task (listed under Deferred to Separate Tasks) will migrate `sniffed` → `browser-sniffed` with provenance metadata compatibility. Near-term, add a one-line code comment near `internal/generator/templates/root.go.tmpl:115` and `internal/generator/generator.go:501` noting that `sniffed` is the legacy name for browser-captured provenance.
- **D6: Do not modify historical docs, and do not rename dated filenames.** `docs/retros/`, `docs/brainstorms/`, past `docs/plans/`, `CHANGELOG.md`, and dated solution-doc filenames describe the state of the system at the time they were written. Filenames never change. Prose inside forward-pointing solution docs may be updated where the doc still guides current behavior.

## Open Questions

### Resolved During Planning

- Should `crowd-sniff` be renamed? No — its name names its source (community SDKs) accurately. (See Phase 0 conversation with user.)
- Should `--spec-source=sniffed` change? No — see D5.
- Should historical docs be rewritten? No — see D6.

### Deferred to Implementation

- Whether any printed CLI in the public library repo (`mvanhorn/printing-press-library`) embeds the old command name in its README or `.printing-press.json` manifest. Unit 5 adds a grep audit step. If findings are non-empty, either file a follow-up PR against that repo or document drift in release notes.

## Implementation Units

- [ ] **Unit 1: Rename cobra command and test file**

**Goal:** Change the subcommand name and update the test file in lockstep. Clean rename — no alias.

**Requirements:** R1, R7

**Dependencies:** None

**Files:**
- Rename: `internal/cli/sniff.go` → `internal/cli/browser_sniff.go`
- Rename: `internal/cli/sniff_test.go` → `internal/cli/browser_sniff_test.go`
- Modify: `internal/cli/root.go` (update `AddCommand` call — `newSniffCmd()` → `newBrowserSniffCmd()`)

**Approach:**
- `newSniffCmd()` → `newBrowserSniffCmd()`, `Use: "sniff"` → `Use: "browser-sniff"`.
- No alias registration. No deprecation plumbing. Invoking `printing-press sniff` after this change returns cobra's standard "unknown command" error.
- Test file updates: every test that constructs the command or asserts against `Use: "sniff"` switches to `"browser-sniff"`.

**Patterns to follow:**
- Existing `newCrowdSniffCmd()` in `internal/cli/crowd_sniff.go` for command structure.

**Test scenarios:**
- Happy path: `browser-sniff --har fixture.har --name foo --output out.yaml` produces the same OpenAPI YAML as the old `sniff` invocation produced against the same fixture (regression check via existing test fixtures).
- Happy path: the `browser-sniff` command appears in `--help` output.
- Error path: invoking `printing-press sniff` returns a cobra "unknown command" error (exit code non-zero). Assert on the error, not the absence of behavior — this pins the clean-break contract.
- Error path: invalid flags on `browser-sniff` surface the same error shape as before.
- Integration (sibling smoke test): `crowd-sniff --help` still works and its flag set is unchanged. Unit 1 modifies `root.go`, which registers both commands; this one-line check catches accidental breakage of the sibling command from shared changes.

**Verification:**
- `go test ./internal/cli/...` passes.
- `printing-press browser-sniff --help` works; `printing-press sniff` fails with "unknown command"; top-level `printing-press --help` lists `browser-sniff` (and does not list `sniff`).

- [ ] **Unit 2: Rename `internal/websniff` package to `internal/browsersniff`**

**Goal:** Align the Go package name with the command name.

**Requirements:** R5, R7

**Dependencies:** Unit 1 (so the only consumer — the renamed command file — can be updated in one step)

**Files:**
- Rename: `internal/websniff/` → `internal/browsersniff/` (12 `.go` files move; every file's `package websniff` declaration — including the 6 `_test.go` files which are in-package tests — updates to `package browsersniff`)
- Modify: `internal/cli/browser_sniff.go` (update import path and symbol qualifier from `websniff.X` to `browsersniff.X`)
- Modify: `internal/cli/crowd_sniff.go` (update import and `websniff.WriteSpec` call at line 150)
- Modify: `internal/generator/generator.go` (update import at line 20, `websniff.FixtureSet` type on the `Generator` struct at line 123, and consumption sites around lines 704-705; a full `grep websniff internal/generator/` lists every hit)
- Modify: `internal/crowdsniff/specgen.go` (comment reference at line 119 — "adapted from websniff/specgen.go")
- Modify: `internal/crowdsniff/aggregate.go` (comment reference at line 10 — "from websniff/classifier.go")
- Modify: any other file that imports or comment-references `internal/websniff` discovered in the pre-change grep (expected: the five above)

**Approach:**
- Change package declaration in every file under the directory from `package websniff` to `package browsersniff`.
- `go mod tidy` and `go build ./...` validate import resolution.
- Friction decision gate: if `go build ./...` fails after renaming the directory and updating the listed import/comment sites in one pass, attempt no more than one round of follow-up fixes (e.g., `go:embed` path adjustments, test-helper path changes). If it still fails, revert Unit 2 and defer the package rename to a separate refactor. Record the failure mode so the follow-up has a starting point. The package rename is valuable but not load-bearing — Unit 1 (command rename) is the high-value change and stands alone.

**Test scenarios:**
- Test expectation: none — this is a pure rename with no behavior change. Existing test suite (`go test ./internal/browsersniff/... ./internal/cli/...`) must pass unchanged as the correctness check.

**Verification:**
- `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run ./...` all pass.
- `grep -r "websniff" internal/` returns zero matches (except possibly `.gocache/`, which is machine-local and disposable).

- [ ] **Unit 3: Update skill artifacts, prose, and Go-side artifact reader (single PR)**

**Goal:** Coherently update skill artifact names, all prose references in the printing-press skills, and the Go code that reads skill-produced artifacts. Land as one commit/PR so the skill and its consumers stay in lockstep.

**Requirements:** R3

**Dependencies:** Unit 1 (skill bash examples invoke `browser-sniff`, which must exist).

**Files:**
Skill artifacts:
- Rename: `skills/printing-press/references/sniff-capture.md` → `skills/printing-press/references/browser-sniff-capture.md`
- Modify: `skills/printing-press/references/crowd-sniff.md` (cross-links to the renamed reference)
- Modify: `skills/printing-press/references/browser-sniff-capture.md` (title, headings, self-references)
- Modify: `skills/printing-press/references/deepwiki-research.md` (single mention — verify wording fits)
- Modify: `skills/printing-press/references/voice.md` (if any mentions — verify)

Skill prose:
- Modify: `skills/printing-press/SKILL.md` (84 occurrences across phase names, artifact references, bash examples, cross-links)
- Modify: `skills/printing-press-retro/SKILL.md`
- Modify: `skills/printing-press-retro/references/issue-template.md`

Go-side artifact reader:
- Modify: `internal/pipeline/research.go` (update `ParseDiscoveryPages` to read `browser-sniff-report.md` instead of `sniff-report.md`)
- Modify: `internal/pipeline/research_test.go` (update fixture filenames)
- Modify: `internal/pipeline/climanifest_test.go` (lines 495 and 505 write/read the literal `sniff-report.md` filename in an archive round-trip test — update to `browser-sniff-report.md` in lockstep with the reader)

**Approach:**
Artifact renames (skill write-side + Go read-side, single atomic change):
- `SNIFF_TARGET_URL` → `BROWSER_SNIFF_TARGET_URL` in every SKILL.md and reference file mention.
- `sniff-gate.json` → `browser-sniff-gate.json` in SKILL.md phase contracts, reference docs, and `$PRESS_RUNSTATE/runs/$RUN_ID/` paths.
- `sniff-report.md` → `browser-sniff-report.md` in skill prose AND in the three `internal/pipeline/` files. No fallback read for the old filename (D3 clean break).

Prose sweep:
- "Sniff Gate" → "Browser-Sniff Gate"
- "Pre-Sniff Auth Intelligence" → "Pre-Browser-Sniff Auth Intelligence"
- "sniff gate" (lowercase prose) → "browser-sniff gate"
- "the sniff" (as a noun) → "the browser-sniff" or rephrase for readability
- "sniffing" (as a verb) → "browser-sniffing" OR rephrase (e.g., "capturing browser traffic") — choose whichever reads better per sentence
- "sniff capture" → "browser-sniff capture"
- Inline bash example `printing-press sniff --har ...` → `printing-press browser-sniff --har ...`
- `crowd-sniff` prose is unchanged.
- Reading pass after each file: if "Pre-Browser-Sniff Auth Intelligence" makes a sentence unreadable, rewrite rather than mechanically substitute.

**Test scenarios:**
- Happy path (Go reader): `ParseDiscoveryPages` reads `browser-sniff-report.md` from a fixture discovery dir and returns the expected URL list (update `internal/pipeline/research_test.go` fixtures).
- Error path: `ParseDiscoveryPages` returns the standard "file not found" error shape when `browser-sniff-report.md` is missing. This is the new contract — pre-rename manuscripts are not read.
- Error path (clean-break pin): when only `sniff-report.md` exists in the discovery dir (simulating a pre-rename manuscript) and `browser-sniff-report.md` is absent, `ParseDiscoveryPages` returns not-found — it does NOT silently read the legacy filename. This test pins D3's no-fallback contract and guards against future regression if anyone adds silent fallback-read logic.
- Integration: `internal/pipeline/climanifest_test.go` archive round-trip passes with `browser-sniff-report.md` as the filename on both write and read sides.

**Verification:**
- `go test ./internal/pipeline/...` passes.
- All markdown cross-references resolve (no broken links pointing to `sniff-capture.md`).
- `grep -rn "\bsniff\b" skills/ | grep -v "browser-sniff\|crowd-sniff"` returns zero hits.
- `grep -rn "sniff-capture\.md\|SNIFF_TARGET_URL\|sniff-gate\.json\|sniff-report\.md" skills/ internal/ docs/solutions/` returns zero live-doc hits.

- [ ] **Unit 4: Update AGENTS.md glossary and other live docs (prose only — no filename renames)**

**Goal:** Fix the incorrect AGENTS.md glossary entry and update user-facing docs.

**Requirements:** R2, R4, R6

**Dependencies:** None (can run in parallel with Units 2–3)

**Files:**
- Modify: `AGENTS.md` (glossary line 62 + 3 total occurrences)
- Modify: `README.md` (lines 16, 165, 172, 176, 221, 223)
- Modify: `docs/solutions/best-practices/sniff-and-crowd-sniff-complementary-discovery-2026-03-30.md` (**prose only** — filename unchanged per D6)
- Modify: `docs/solutions/best-practices/multi-source-api-discovery-design-2026-03-30.md` (**prose only** — 7 mentions)

**Approach:**
- AGENTS.md fix: the current row `**crowd sniff** / **sniff**` is factually wrong — `sniff` does not scrape npm/PyPI/GitHub. Split into two rows:
  - `**browser-sniff**` — "Browser-driven API discovery. User captures live traffic via browser automation (browser-use, agent-browser) or DevTools as a HAR; the `browser-sniff` subcommand analyzes the HAR and produces an OpenAPI-compatible spec. Use when no official spec exists or to supplement one with endpoints the docs miss."
  - `**crowd-sniff**` — existing definition, unchanged except for hyphenation.
- README: substitute `sniff` → `browser-sniff` in user-facing prose. The line `browser capture, HAR import, discovery provenance` stays as-is.
- Solution doc content update: update prose inside `sniff-and-crowd-sniff-complementary-discovery-2026-03-30.md` to reference `browser-sniff`. Filename stays (dated file = historical record per D6). Optionally add a one-line note at the top: "Command names updated from `sniff` → `browser-sniff` after 2026-04-18; captured here for continuity."
- `adaptive-rate-limiting-sniffed-apis.md` stays unchanged (the "sniffed" adjective refers to provenance, not the command — see D5).

**Test scenarios:**
- Test expectation: none — documentation.

**Verification:**
- AGENTS.md glossary now has accurate, split rows for the two techniques.
- `grep -n "\bsniff\b" AGENTS.md README.md` returns only intentional uses within `browser-sniff` or `crowd-sniff`.
- No solution doc filenames changed — inbound links preserved.

- [ ] **Unit 5: Sweep live references, audit public library repo, verify historical docs untouched**

**Goal:** Final sweep. Catch stragglers, confirm the rename is complete on live surfaces, audit the public library repo for stale references, and verify historical docs are untouched.

**Requirements:** R2, R3, R4, R6

**Dependencies:** Units 1–4

**Files:**
- Read-only: `docs/plans/`, `docs/retros/`, `docs/brainstorms/`, `CHANGELOG.md` (confirm untouched)
- Modify: any live file discovered by the sweep

**Approach:**
- Run `grep -rn "\bsniff\b" .` excluding `.git/`, `.gocache/`, `docs/retros/`, `docs/brainstorms/`, `docs/plans/` (except plans created on/after 2026-04-18), `CHANGELOG.md`, and `node_modules/`.
- Classify each hit: (a) already-correct `browser-sniff`/`crowd-sniff` compound → ignore, (b) historical snapshot → ignore, (c) live reference that slipped through → fix.
- **Public library repo audit:** Clone or fetch `mvanhorn/printing-press-library` (read-only check) and grep its contents for `printing-press sniff` or standalone `sniff` references in published CLI READMEs and `.printing-press.json` manifests. If findings are non-empty, either file a follow-up PR in that repo or document the drift in this release's notes. Surface the result (count + sample paths) in the PR description.
- **Release-please dry-run:** Before creating the PR, run release-please locally (or in a branch) to confirm whether `refactor(cli)!:` triggers a minor or major bump under the current repo config. Record the result in the PR description. If the bump is unexpected (major pre-1.0), pause and discuss before merging.

**Test scenarios:**
- Test expectation: none — verification-only pass.

**Verification:**
- `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run ./...` all pass.
- `gofmt -w ./...` produces no changes.
- Manual smoke test: build the binary, run `./printing-press browser-sniff --help`, `./printing-press sniff` (returns "unknown command" error), `./printing-press --help` (shows `browser-sniff`, does not show `sniff`).
- Final grep sweep shows zero bare `sniff` tokens in live files.
- Public library audit result recorded in PR description.
- Release-please dry-run result recorded in PR description.
- D5 code comments present: `grep -A1 "sniffed" internal/generator/generator.go internal/generator/templates/root.go.tmpl | grep -q "legacy name"` (or equivalent) confirms the legacy-name explanatory comment was added near the `SpecSource == "sniffed"` branch.

## System-Wide Impact

- **Interaction graph:** `printing-press sniff` stops existing. Every in-repo skill invocation (bash examples in SKILL.md, retro skill references) is updated in Unit 3 in the same change set. After merge, no internal path invokes the old name.
- **Error propagation:** No change to the `browser-sniff` command's behavior. The old `sniff` invocation returns cobra's standard "unknown command" error.
- **State lifecycle risks:** Runstate artifacts (`sniff-gate.json`, `sniff-report.md`) are renamed forward-only. A user mid-pipeline across the upgrade boundary re-enters Phase 1.7 and writes the new-name artifacts. Pre-rename published manuscripts in `~/printing-press/manuscripts/` cannot be re-consumed by emboss/retro flows without manual filename migration — accepted, documented in release notes.
- **API surface parity:** The `crowd-sniff` command's CLI surface is unchanged.
- **Integration coverage:** The `internal/pipeline/climanifest_test.go` archive round-trip test covers the Go reader + skill writer contract for `browser-sniff-report.md`. Unit 3 updates both sides in lockstep.
- **Public library repo:** Previously-published CLIs in `mvanhorn/printing-press-library` may contain stale `sniff` references in their generated READMEs or manifests. Unit 5 grep-audits this surface; findings drive either a follow-up PR or release-note disclosure.
- **Unchanged invariants:** `crowd-sniff` command name, behavior, and flags. The `--spec-source=sniffed` provenance value and its generator-template branch (migration deferred — D5). The `internal/crowdsniff/` package. The adaptive-rate-limiting solution doc name. Locally-generated printed CLIs in `~/printing-press/library/` (they regenerate from templates, which don't reference the command name).

## Risks & Dependencies

| Risk | Mitigation |
|------|------------|
| A historical doc in `docs/plans/` references `sniff` in a way that becomes misleading after the rename | Leave historical docs untouched per D6. Plans describe the system as it was when written; readers understand plans are dated. |
| `internal/websniff` package rename has hidden consumers beyond the three known live imports | Unit 2 starts with a fresh `grep -r "websniff" .` (excluding `.git/`, `.gocache/`). If additional consumers exist, include them in Unit 2. If `go build ./...` fails, try one round of follow-up fixes then revert Unit 2 and defer — the command rename (Unit 1) is the high-value change and stands alone. |
| Skill prose becomes unreadable with long compound names like "Pre-Browser-Sniff Auth Intelligence" | Unit 3 includes an explicit reading pass; rewrite rather than mechanically substitute when prose breaks. |
| Downstream scripts or external consumers shell to `printing-press sniff` | Accepted breakage. Pre-1.0 + internal tool; no verified external consumers. Release notes call out the removal prominently. If a consumer surfaces post-release, they pin to the previous version or update their invocation. |
| Pre-rename manuscripts in `~/printing-press/manuscripts/` contain `sniff-report.md`, making them unreadable by post-rename emboss/retro flows | Accepted. Document in release notes. Users with mission-critical pre-rename manuscripts can rename the file manually (single `mv`) or re-run Phase 1.7 to regenerate. |
| `refactor(cli)!:` commit triggers unexpected release-please version bump | Unit 5 includes a release-please dry-run step before merge. If the bump is unexpected (major pre-1.0), pause and discuss before merging. |
| Public library repo CLIs contain stale `sniff` references | Unit 5 audits this surface. Findings drive a follow-up PR in the library repo or a release-note disclosure. Not blocking this PR. |

## Documentation / Operational Notes

- Release notes entry (next release):
  > **BREAKING:** `printing-press sniff` has been renamed to `printing-press browser-sniff` to clarify the distinction vs. `crowd-sniff`. The old `sniff` command has been removed — no alias. Update any scripts or invocations. Skill-authored artifacts also renamed: `sniff-gate.json` → `browser-sniff-gate.json`, `sniff-report.md` → `browser-sniff-report.md`, `SNIFF_TARGET_URL` → `BROWSER_SNIFF_TARGET_URL`. In-flight runs re-generate the new artifacts at Phase 1.7; pre-rename published manuscripts cannot be read by post-rename emboss/retro flows without a manual `mv`.
- The CHANGELOG is auto-generated by release-please from conventional commits. The commit message (`refactor(cli)!: rename sniff to browser-sniff`) carries the breaking-change signal. Unit 5 runs a release-please dry-run to confirm the actual version bump under the current repo config before merge.
- Skills documentation surfaces the new name via the printing-press plugin manifest; no manual plugin-catalog update needed.

## Sources & References

- Plan originated from conversation: user observed that "sniff" is a misleading name vs. "crowd sniff" since the former is fundamentally browser-driven.
- Related code: `internal/cli/sniff.go`, `internal/cli/root.go`, `internal/websniff/`, `skills/printing-press/SKILL.md`, `AGENTS.md`.
- Prior plan context: `docs/plans/2026-03-29-003-feat-crowd-sniff-plan.md` (established crowd-sniff as the sibling command).
- Prior solution doc: `docs/solutions/best-practices/sniff-and-crowd-sniff-complementary-discovery-2026-03-30.md` (prose updated per Unit 4; filename preserved per D6).
