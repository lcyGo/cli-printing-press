---
title: "feat: Codex Mode Token Overhaul — Deep Delegation for Printing Press"
type: feat
status: active
date: 2026-03-30
origin: docs/plans/2026-03-27-feat-printing-press-codex-delegation-mode-plan.md
---

# feat: Codex Mode Token Overhaul — Deep Delegation for Printing Press

## Overview

Make codex mode actually work and then make it work *well*. The current SKILL.md detects `--codex` but never branches on it — Phase 3 and Phase 4 run identically with or without the flag. This plan fixes that, then goes further by moving prompt assembly and task decomposition into the Go binary so Claude's role in code generation shrinks from "write all the code" to "review diffs and make judgment calls."

**Recommendation: Phased hybrid approach.** Ship skill-level delegation first (Stage 1, ~2 days), then build Go binary intelligence (Stage 2, ~3-4 days). Stage 1 delivers immediate ~60% of Phase 3+4 token savings (~35-40% of total run). Stage 2 pushes it to ~75-80% of Phase 3+4 by eliminating Claude's prompt-assembly overhead.

## Problem Frame

A full printing-press run burns 45-85 minutes of Opus tokens. The breakdown:

| Phase | Opus Minutes | Work Type | Delegatable? |
|-------|-------------|-----------|-------------|
| Phase 0: Resolve & Reuse | <5 | Mechanical | No (already cheap) |
| Phase 1: Research Brief | 5-10 | Web search, synthesis | No (needs Opus reasoning) |
| Phase 1.5: Ecosystem Absorb | 5-10 | Web search, cataloging | No (needs Opus reasoning) |
| Phase 2: Generate | <2 | Go binary execution | No (already cheap) |
| **Phase 3: Build The GOAT** | **10-20** | **Code generation** | **Yes — 60% of total cost** |
| **Phase 4: Shipcheck fixes** | **10-20** | **Bug fixing** | **Yes — 20% of total cost** |
| Phase 5: Live Smoke | 2-5 | Test execution | No (needs interpretation) |

Running `/printing-press Discord codex` today does nothing different because SKILL.md lines 55-70 describe the mode but Phases 3/4 have zero codex branching.

## Requirements Trace

- R1. Codex mode must actually delegate code-writing tasks to Codex CLI
- R2. Quality cannot decrease — generated CLIs must score within 5 points of non-codex mode
- R3. Opus token usage must drop by at least 35% per full run (Stage 1), targeting 50%+ with Stage 2
- R4. Fallback to Claude must be automatic and invisible on any Codex failure
- R5. Codex prompts must include actual code context, not descriptions (proven pattern from osc-nightnight)
- R6. Verify-floor calibration and scorecard ordering invariants must be preserved (see origin: docs/solutions/logic-errors/scorecard-accuracy-broadened-pattern-matching-2026-03-27.md)
- R7. Output paths must respect the checkout-scoped layout contract (see origin: docs/solutions/best-practices/checkout-scoped-printing-press-output-layout-2026-03-28.md)

## Scope Boundaries

- This plan does NOT change Phase 1/1.5 research (those need Opus reasoning and web search)
- This plan does NOT change the quality gates or scoring system
- This plan does NOT add new features to generated CLIs
- Go binary changes (Phase 2) are additive commands, not refactors of existing generation logic

## Context & Research

### Where Tokens Are Actually Burned in Phase 3

Claude's work in Phase 3 breaks down into:

1. **Task decomposition** (~10% of Phase 3 tokens): Reading absorb manifest, deciding what to build in what order
2. **Context gathering** (~15%): Reading current generated code to understand what exists, what's missing, what patterns to follow
3. **Code writing** (~60%): Actually writing Go functions — store tables, workflow commands, transcendence features
4. **Inter-task review** (~15%): After each piece, reading the result, checking it compiles, deciding what's next

With naive Codex delegation (Claude assembles prompts, Codex writes code), you save the 60% code-writing but Claude still burns tokens on decomposition, context gathering, and review. The Go binary can eliminate most of the context gathering and decomposition overhead.

### Proven Codex Patterns (from osc-nightnight)

The working pattern at `~/.claude/skills/osc-nightnight/SKILL.md:1260-1322`:
- Prompt includes ACTUAL CODE (not descriptions) via `$(head -20 file.go)`
- `echo "$PROMPT" | codex exec --yolo -c 'model_reasoning_effort="medium"' -m "gpt-5.4" -`
- Post-codex: format check → lint → diff check
- On any failure: fall back to Claude, increment counter
- 3 consecutive failures: disable codex for rest of session

### Institutional Learnings

- Verify-floor formula (`floor = (verifyPassRate * 80) / 100`) is load-bearing — any workflow change must preserve the caps-before-totals/floors-after-totals ordering (see origin: docs/solutions/best-practices/steinberger-scorecard-scoring-architecture-2026-03-27.md)
- Delegated agents must write to correct `.runstate/<scope>/` paths, not hardcoded paths

### Existing Generator Capabilities

The generator already knows:
- All resources and their CRUD operations (from `APISpec`)
- Domain archetype and recommended features (from profiler)
- Which vision templates to render (sync, search, store, etc.)
- Pagination patterns, auth protocol, data types

This information is currently used only at generation time. But it's exactly what Codex needs to write good code — and the Go binary can surface it as structured task data for free.

## Three Approaches Analyzed

### Approach A: Skill-Level Delegation Only

**What:** Add codex branching to SKILL.md Phase 3 and Phase 4. Claude reads absorb manifest, assembles Codex prompts with code context, delegates each task, reviews results.

**Token savings:** ~60% (code writing moves to Codex, but Claude still does prompt assembly and review)

**Effort:** ~2 days (one file change: SKILL.md)

**Pros:**
- Ships fast, one file change
- Proven pattern from osc-nightnight
- No Go code changes needed

**Cons:**
- Claude still spends significant tokens assembling each prompt (reading files, extracting context, formatting)
- N tasks × (read current state + extract context + format prompt + review result) = still substantial
- Each Codex call is sequential — Claude waits, reviews, then assembles next prompt
- Prompt quality depends entirely on Claude's in-context understanding of the generated codebase

**Realistic savings estimate:** 50-60% of Phase 3+4 tokens, ~35-40% of total run

### Approach B: Full Go Binary Token Overhaul

**What:** Add new Go commands that generate structured task manifests and ready-to-pipe Codex prompts. The Go binary does the work of reading the codebase, extracting context, and formatting prompts. Claude just orchestrates.

New commands:
- `printing-press tasks --dir <cli> --manifest <absorb-manifest>` — Reads the generated CLI, cross-references against the absorb manifest, outputs a JSON array of structured implementation tasks with pre-assembled code context
- `printing-press fix-tasks --dir <cli> --dogfood-output <json>` — Reads dogfood/verify findings and outputs structured fix tasks with file context
- Generator template improvements so Phase 3 has less to build

**Token savings:** ~75-80% (Claude orchestrates but doesn't read/format/assemble)

**Effort:** ~5-6 days (Go code + SKILL.md changes)

**Pros:**
- Maximum token savings — Claude barely touches generated code
- Task manifests are deterministic and testable (Go tests, not vibes)
- Prompts include perfect context (Go binary reads exact files, not Claude approximating)
**Cons:**
- More upfront work
- Go binary needs to understand absorb manifest format (new dependency)
- Risk of over-engineering — Go tasks might not capture the nuance Claude adds
- Delays shipping any savings by a week

**Realistic savings estimate:** 70-80% of Phase 3+4 tokens, ~50-55% of total run

### Approach C: Phased Hybrid (Recommended)

**What:** Ship Approach A first (immediate savings), then build Approach B on top (deeper savings). Each phase is independently valuable.

**Stage 1 (ship in ~2 days):** Skill-level delegation in SKILL.md
- Codex branching in Phase 3 and Phase 4
- Prompt templates for common task types (store, workflow command, fix)
- Circuit breaker and fallback logic
- Post-codex validation gates

**Stage 2 (ship in ~3-4 days after Stage 1):** Go binary intelligence
- `printing-press tasks` command that generates structured task manifests
- `printing-press fix-tasks` command for structured fix recommendations
- Generator template improvements to produce more complete code
- SKILL.md updated to use Go binary tasks instead of manual prompt assembly

**Token savings:** Stage 1: ~60%. Stage 2: ~75-80%.

**Why this is the right call:**
1. Stage 1 delivers real savings immediately — you stop burning tokens while Stage 2 is built
2. Stage 1 is a forcing function — you learn exactly which prompt patterns work well before encoding them in Go
3. Stage 2 builds on Stage 1 learnings rather than guessing upfront what Go should generate
4. Each stage is independently shippable and testable
5. If Stage 1 savings are "good enough," Stage 2 becomes optional optimization

## Key Technical Decisions

- **Codex invocation via stdin pipe** (not temp file): Simpler, proven in osc-nightnight. `echo "$PROMPT" | codex exec --yolo -`
- **One Codex call per task** (not batched): Codex handles focused tasks better. Batch mode risks one failure killing the whole batch. Sequential calls with circuit breaker is more robust.
- **Medium reasoning effort for Codex**: `model_reasoning_effort="medium"` — code generation is mechanical, doesn't need deep reasoning. Proven in osc-nightnight.
- **Go binary task manifests output JSON** (not YAML): JSON pipes to `jq`, is easier to parse in bash, and matches the existing `--json` convention across printing-press commands.
- **Generator improvements are additive**: New template logic that produces more complete store.go / workflow stubs. Existing templates stay untouched — new logic layers on top.

## Open Questions

### Resolved During Planning

- **Should codex mode be opt-in or default?** Opt-in via `codex` argument. The existing plan says opt-in, osc-nightnight auto-enables when codex binary exists. Recommendation: keep opt-in for printing-press since quality is the primary concern and users should consciously choose the tradeoff.
- **Which Codex model?** Use `-m "gpt-5.4"` (same as osc-nightnight). Omitting `-m` would use whatever default Codex has, which is less predictable.
- **How to handle Codex writing to wrong paths?** The SKILL.md prompt template must include `cd "$PRESS_LIBRARY/<api>-pp-cli"` as the first Codex instruction. Go binary task manifests will include absolute paths.

### Deferred to Implementation

- **Exact threshold for "good enough" prompt context**: How many lines of surrounding code does Codex need? Phase 1 will test with `head -50` / `grep -A 20` patterns; Phase 2 Go binary will use AST-aware extraction.
- **Whether parallel Codex calls are practical**: Sequential is safe; parallel would be faster but risks file conflicts. Test in Phase 1 and decide for Phase 2.

## High-Level Technical Design

> *This illustrates the intended approach and is directional guidance for review, not implementation specification. The implementing agent should treat it as context, not code to reproduce.*

### Stage 1: Skill-Level Delegation Flow

```
Claude reads absorb manifest
  │
  ├─ For each Priority 0/1/2 task:
  │    ├─ Read relevant source files (store.go, root.go, etc.)
  │    ├─ Assemble CODEX_PROMPT with actual code context
  │    ├─ echo "$CODEX_PROMPT" | codex exec --yolo ...
  │    ├─ Post-validation: go build + go vet + diff check
  │    ├─ On failure: fall back to Claude inline
  │    └─ On 3 consecutive failures: disable codex
  │
  └─ After all tasks: run Priority 1 Review Gate as normal
```

### Stage 2: Go Binary Intelligence Flow

```
printing-press tasks --dir <cli> --manifest <manifest.md> --json
  │
  ├─ Go binary reads generated CLI source tree
  ├─ Go binary reads absorb manifest (markdown → structured)
  ├─ Cross-references: what exists vs what's needed
  ├─ For each gap: extracts relevant code context from source
  └─ Outputs JSON array of tasks:
       [
         {
           "id": "store-messages-table",
           "priority": 0,
           "type": "store",
           "file": "internal/store/store.go",
           "description": "Add messages table with UpsertMessage/SearchMessages",
           "context": { "current_code": "...", "patterns": "...", "conventions": "..." },
           "constraints": ["<200 lines", "match existing table pattern"],
           "verify": "go build ./... && go vet ./..."
         }
       ]

Claude just iterates:
  for task in $(jq -c '.[]' tasks.json); do
    CODEX_PROMPT=$(jq -r '.prompt_template' <<< "$task")
    echo "$CODEX_PROMPT" | codex exec --yolo ...
    # validate, fallback, continue
  done
```

## Implementation Units

### Stage 1: Skill-Level Delegation (~2 days)

- [ ] **Unit 1: Codex mode detection and environment guards in SKILL.md**

**Goal:** Wire up the codex flag so it actually affects behavior, with proper environment detection and circuit breaker state.

**Requirements:** R1, R4

**Dependencies:** None

**Files:**
- Modify: `skills/printing-press/SKILL.md` (add codex detection block after setup contract, before Phase 0)

**Approach:**
- Add a `CODEX_MODE` detection block after the setup contract (line ~149)
- Check for `codex`/`--codex` in arguments
- Verify `codex` binary exists
- Check we're not already in a Codex sandbox (`$CODEX_SANDBOX`, `$CODEX_SESSION_ID`)
- Initialize `CODEX_CONSECUTIVE_FAILURES=0` counter
- Pattern: follow osc-nightnight lines 160-175 exactly

**Patterns to follow:**
- osc-nightnight SKILL.md lines 160-175 (mode detection)
- ce-work-beta SKILL.md (environment guard pattern)

**Test scenarios:**
- Happy path: `codex` in args + binary exists → CODEX_MODE=true
- Happy path: no `codex` in args → CODEX_MODE=false
- Edge case: `codex` in args but binary not installed → CODEX_MODE=false with warning
- Edge case: already inside Codex sandbox → CODEX_MODE=false silently
- Edge case: `--codex` flag syntax → same as bare `codex`

**Verification:**
- Running `/printing-press Discord codex` sets CODEX_MODE=true when codex binary exists
- Running `/printing-press Discord` leaves CODEX_MODE=false

---

- [ ] **Unit 2: Phase 3 codex delegation with prompt templates**

**Goal:** When CODEX_MODE is true, Phase 3 delegates each implementation task to Codex with a structured prompt containing actual code context.

**Requirements:** R1, R2, R3, R5

**Dependencies:** Unit 1

**Files:**
- Modify: `skills/printing-press/SKILL.md` (Phase 3 section, lines ~847-909)

**Approach:**
Add a codex branching block inside Phase 3 that:

1. After reading the absorb manifest, decomposes work into discrete tasks (one per command/feature)
2. For each task, reads the relevant source files from the generated CLI
3. **Pre-Codex snapshot**: `git add -A && git stash push -m "pre-codex-$TASK_ID"` to create a clean restore point
4. Assembles a CODEX_PROMPT using a standard template (see below)
5. Pipes to `codex exec --yolo -c 'model_reasoning_effort="medium"' -m "gpt-5.4" -`
6. Post-validation: `go build ./...`, `go vet ./...`, `git diff --stat` (non-empty)
7. On validation failure: `git stash pop` to revert Codex changes, increment CODEX_CONSECUTIVE_FAILURES, fall back to Claude for that task
8. On success: `git stash drop` (discard restore point), reset CODEX_CONSECUTIVE_FAILURES=0
9. On 3 consecutive failures: set CODEX_MODE=false, print warning, continue in Claude mode

**Prompt template structure** (from osc-nightnight, adapted for printing-press):
```
TASK: [specific task from manifest]
FILES TO MODIFY: [exact paths]
CURRENT CODE: [actual code from relevant files — head/grep/cat, not descriptions]
EXPECTED CHANGE: [plain English description of the code to write]
CONVENTIONS: [package name, cobra pattern, error handling, store patterns from existing code]
CONSTRAINTS: [no git ops, no files outside listed paths, <200 lines, run go build]
VERIFY: [cd <cli-dir> && go build ./... && go vet ./...]
```

**Task type templates** (specialize the generic template):
- **Store table**: reads existing store.go, extracts table creation pattern, asks for new table + CRUD methods
- **Workflow command**: reads root.go for cobra registration pattern, existing command for structure, asks for new command
- **Transcendence command**: reads store.go for available tables/queries, asks for cross-entity command
- **Fix/polish**: reads specific file + dogfood finding, asks for targeted fix

The Priority 1 Review Gate runs identically whether codex or not — it tests the generated commands, not how they were written.

**Patterns to follow:**
- osc-nightnight SKILL.md lines 1260-1322 (prompt assembly and codex exec)
- Existing Phase 3 priority ordering (P0 → P1 → review gate → P2 → P3)

**Test scenarios:**
- Happy path: Codex writes a store table that compiles and passes go vet
- Happy path: Codex writes a workflow command that passes Priority 1 Review Gate (--help, --dry-run, --json)
- Error path: Codex produces code that doesn't compile → falls back to Claude for that task, next task still tries Codex
- Error path: 3 consecutive Codex failures → disables codex mode, remaining tasks use Claude
- Edge case: Codex produces empty diff → treated as failure, falls back to Claude
- Integration: Full Phase 3 with codex produces a CLI that scores within 5 points of non-codex

**Verification:**
- Phase 3 with CODEX_MODE=true delegates code-writing tasks to Codex
- Phase 3 with CODEX_MODE=false runs identically to current behavior
- Generated CLI passes all 7 quality gates regardless of mode

---

- [ ] **Unit 3: Phase 4 codex delegation for fix cycles**

**Goal:** When CODEX_MODE is true, Phase 4 fix cycles delegate each bug fix to Codex.

**Requirements:** R1, R3, R4, R6

**Dependencies:** Unit 2

**Files:**
- Modify: `skills/printing-press/SKILL.md` (Phase 4 section, lines ~911-951)

**Approach:**
After running dogfood/verify/scorecard (these always run on Claude — they're Go binary executions):

1. Claude reads the output and identifies specific bugs to fix
2. For each bug, assembles a fix-focused Codex prompt:
   - The exact file and line range with the bug
   - The dogfood/verify finding describing what's wrong
   - The current code (actual content, not description)
   - The expected fix behavior
3. Delegates to Codex, validates, falls back as needed
4. Shares the same circuit breaker counter from Phase 3

Fix prompts are simpler than Phase 3 prompts — typically 5-50 lines per fix, one file per task.

**Critical:** The shipcheck tools themselves (dogfood, verify, scorecard) always run via the Go binary on Claude's side. Only the CODE FIXES are delegated. The verify-floor calibration and scorecard ordering invariants are untouched because those are Go code, not skill logic.

**Patterns to follow:**
- Phase 3 codex pattern (Unit 2)
- Existing Phase 4 fix ordering (generation blockers → invalid paths → dead flags → broken commands → polish)

**Test scenarios:**
- Happy path: Codex fixes a dead flag by wiring it to the correct handler
- Happy path: Codex fixes an invalid API path by correcting the URL template
- Error path: Codex "fix" breaks the build → revert, fall back to Claude
- Edge case: No bugs found by dogfood/verify → no Codex calls needed, skip delegation
- Integration: Scorecard after codex-fixed Phase 4 matches non-codex scores within 5 points

**Verification:**
- Shipcheck tools run identically regardless of codex mode
- Only code fixes are delegated
- Post-fix scorecard preserves verify-floor calibration

---

### Stage 2: Go Binary Intelligence (~3-4 days)

- [ ] **Unit 4: `printing-press tasks` command**

**Goal:** New CLI command that reads a generated CLI directory and an absorb manifest, then outputs a structured JSON array of implementation tasks with pre-assembled code context.

**Requirements:** R3, R5, R7

**Dependencies:** Units 1-3 (Stage 1 learnings inform what context Codex needs)

**Files:**
- Create: `internal/cli/tasks.go`
- Create: `internal/tasks/tasks.go` (task manifest logic)
- Create: `internal/tasks/tasks_test.go`
- Modify: `internal/cli/root.go` (register new command)

**Approach:**
The `tasks` command:
1. Reads the generated CLI directory structure (find all Go files, understand what commands exist)
2. Reads the absorb manifest markdown (parse feature tables)
3. Cross-references: which manifest features are already implemented vs missing
4. For each missing feature, generates a task object with:
   - Task ID, priority, type (store/workflow/transcendence/fix)
   - Target file path (where to create/modify)
   - Current code context (extracted from relevant files — store.go signatures, root.go command registration, existing command as pattern reference)
   - Expected behavior (from manifest row)
   - Conventions (extracted from existing code patterns)
   - Constraints and verify command
5. Outputs JSON to stdout

The key insight: the Go binary can do AST-aware context extraction. Instead of Claude doing `head -50 store.go`, the binary can extract the exact function signatures, table definitions, and patterns that Codex needs. This produces tighter, more relevant context.

**Patterns to follow:**
- Existing `internal/cli/` command structure (newXxxCmd pattern)
- `internal/pipeline/` for reading/analyzing generated CLI directories
- `internal/profiler/` for code analysis patterns

**Test scenarios:**
- Happy path: Given a generated CLI and an absorb manifest with 10 features, outputs 10 tasks with correct file paths and code context
- Happy path: Tasks are ordered by priority (P0 store → P1 absorbed → P2 transcendence)
- Edge case: Feature already implemented in generated code → excluded from task list
- Edge case: Absorb manifest has features that don't map to any spec endpoint → task includes a note about manual implementation
- Error path: Invalid directory path → clear error message
- Error path: Malformed manifest → parse what's possible, warn about unparseable rows

**Verification:**
- `printing-press tasks --dir <cli> --manifest <manifest.md> --json | jq length` returns task count matching unimplemented manifest features
- Each task's `context.current_code` field contains actual code from the CLI directory
- Tasks compile: all file paths in tasks reference valid locations in the CLI directory structure

---

- [ ] **Unit 5: `printing-press fix-tasks` command**

**Goal:** New CLI command that reads dogfood/verify output and the generated CLI, then outputs structured fix tasks with code context.

**Requirements:** R3, R5, R6

**Dependencies:** Unit 4

**Files:**
- Create: `internal/cli/fixtasks.go`
- Create: `internal/tasks/fixtasks.go`
- Create: `internal/tasks/fixtasks_test.go`
- Modify: `internal/cli/root.go` (register new command)

**Approach:**
The `fix-tasks` command:
1. Reads dogfood JSON output (dead flags, dead functions, invalid paths, etc.)
2. Reads verify JSON output (runtime failures, auto-fix suggestions)
3. For each finding, reads the relevant source file from the CLI directory
4. Generates a fix task with:
   - The finding description
   - The exact file and relevant code snippet
   - The suggested fix approach (from dogfood/verify's own recommendations)
   - Constraints and verify command
5. Outputs JSON to stdout, ordered by fix priority (build breaks → invalid paths → dead code → polish)

**Patterns to follow:**
- Unit 4 task structure
- Existing dogfood/verify output format in `internal/pipeline/`

**Test scenarios:**
- Happy path: Given dogfood output with 5 findings, outputs 5 fix tasks with correct file references and code context
- Happy path: Fix tasks are ordered by severity (build breaks first)
- Edge case: Finding references a file that doesn't exist → skip with warning
- Edge case: Dogfood output is empty (no issues) → empty task array
- Error path: Invalid dogfood JSON → clear parse error

**Verification:**
- `printing-press fix-tasks --dir <cli> --dogfood <json> --verify <json> --json | jq length` returns finding count
- Each fix task's code context matches the actual file content

---

- [ ] **Unit 6: Generator template improvements to reduce Phase 3 work**

**Goal:** Make the generator produce more complete code so Phase 3 has fewer gaps to fill.

**Requirements:** R3

**Dependencies:** None (can run in parallel with Units 4-5)

**Files:**
- Modify: `internal/generator/templates/store.go.tmpl` (more complete table definitions)
- Modify: `internal/generator/templates/` (workflow command stubs with fuller logic)
- Modify: `internal/generator/generator.go` (pass more profiler data to templates)
- Test: `internal/generator/generator_test.go`

**Approach:**
NOTE: `store.go.tmpl` already generates real CREATE TABLE DDL, typed columns from `schema_builder.go`, FTS5 indexes with triggers, and domain-specific Upsert/Search methods. The templates are NOT skeleton TODOs.

The remaining gap is in **workflow command stubs and vision templates** — these generate cobra registration and placeholder logic but leave the actual business logic (sync cursors, cross-entity queries, analytics aggregation) to Phase 3. Improvements should focus on:
- Fuller workflow command stubs that include pagination loop scaffolding
- Sync command templates with cursor tracking boilerplate
- Search command templates pre-wired to FTS tables from the store

This is about reducing Phase 3's mechanical work so Codex has less to invent.

**Patterns to follow:**
- Existing template data pipeline: `generator.go` → `templateData` struct → `.tmpl` files
- `internal/profiler/` output format for domain analysis
- Existing store.go.tmpl and schema_builder.go patterns (these are already good — don't duplicate)

**Test scenarios:**
- Happy path: Generator with a profiled spec produces sync command with pagination loop scaffolding
- Happy path: Search command template pre-wires to FTS tables matching profiler output
- Edge case: API with no syncable resources → sync template not rendered
- Integration: Generated workflow commands compile and pass go vet without Phase 3 edits

**Verification:**
- Generated workflow commands from petstore spec have fuller scaffolding than current
- Phase 3 task count is measurably reduced for a reference API
- `go build` passes on generated code

---

- [ ] **Unit 7: SKILL.md updated to use Go binary task manifests**

**Goal:** Update SKILL.md Phase 3 and Phase 4 to use `printing-press tasks` and `printing-press fix-tasks` instead of Claude-assembled prompts.

**Requirements:** R1, R3, R5

**Dependencies:** Units 4, 5, 6

**Files:**
- Modify: `skills/printing-press/SKILL.md` (Phase 3 and Phase 4 codex paths)

**Approach:**
Replace the Phase 1 Claude-assembled prompts with Go binary output:

Phase 3:
```bash
# Instead of Claude reading files and assembling prompts:
TASKS=$(printing-press tasks --dir "$PRESS_LIBRARY/<api>-pp-cli" \
  --manifest "$RESEARCH_DIR/<stamp>-feat-<api>-pp-cli-absorb-manifest.md" --json)

# Iterate and delegate each task (indexed to handle multi-line JSON values)
TASK_COUNT=$(echo "$TASKS" | jq length)
for i in $(seq 0 $((TASK_COUNT-1))); do
  task=$(echo "$TASKS" | jq -c ".[$i]")
  CODEX_PROMPT=$(echo "$task" | jq -r '.prompt')
  echo "$CODEX_PROMPT" | codex exec --yolo -c 'model_reasoning_effort="medium"' -m "gpt-5.4" -
  # validate, fallback, continue
done
```

Phase 4:
```bash
FIXES=$(printing-press fix-tasks --dir "$PRESS_LIBRARY/<api>-pp-cli" \
  --dogfood <dogfood-json> --verify <verify-json> --json)
# Same delegation loop
```

Claude's role shrinks to:
1. Running the Go commands
2. Iterating the task JSON
3. Reviewing failures and deciding fallbacks
4. Making judgment calls when Codex can't handle a task

**Patterns to follow:**
- Stage 1 codex delegation pattern (Unit 2) as the iteration/fallback framework
- Go binary `--json` output convention

**Test scenarios:**
- Happy path: `printing-press tasks` output feeds directly to Codex delegation loop
- Happy path: `printing-press fix-tasks` output feeds to fix delegation loop
- Error path: Go binary fails → falls back to Stage 1 style (Claude-assembled prompts)
- Integration: End-to-end run with Go binary tasks produces same-quality CLI as manual prompts

**Verification:**
- Phase 3 with Go binary tasks uses fewer Claude tokens than Stage 1 style prompt assembly
- Generated CLI quality is identical between Stage 1 and Stage 2 approaches

## System-Wide Impact

- **Interaction graph:** SKILL.md is the only consumer of codex delegation. The Go binary, quality gates, and scoring system are unaffected.
- **Error propagation:** Codex failures are caught per-task and fall back to Claude. The circuit breaker prevents cascading failures. The shipcheck block runs regardless.
- **State lifecycle:** No new state is introduced. `CODEX_CONSECUTIVE_FAILURES` is a shell variable, not persisted. The `tasks` command is stateless — reads files, outputs JSON.
- **Quality gate parity:** All 7 quality gates, dogfood, verify, and scorecard run identically regardless of codex mode. The verify-floor calibration is in Go code, not skill logic.
- **Unchanged invariants:** The research phases (0, 1, 1.5), generation (2), shipcheck tools (4), and live smoke (5) are completely unchanged. Only the code-writing portions of Phase 3 and the fix-writing portions of Phase 4 are affected.

## Risks & Dependencies

| Risk | Mitigation |
|------|------------|
| Codex writes code that compiles but is subtly wrong (wrong API paths, wrong field names) | Dogfood validator catches this — it cross-references generated code against the source spec. Proof of Behavior verification in shipcheck. |
| Codex prompt context is insufficient → empty diffs or wrong files | Phase 1 tests prompt patterns with real APIs. Phase 2 Go binary provides AST-aware context extraction. |
| Go binary manifest parser can't handle all absorb manifest formats | Parse defensively — extract what's structured, warn about unparseable rows, never block on parse errors. |
| Generator template improvements break existing generation | Additive changes only — new template logic supplements existing, doesn't replace. Generator tests catch regressions. |
| Codex model changes break delegation pattern | Omit `-m` flag as fallback option. Monitor Codex changelog. The circuit breaker auto-disables on failures. |

## Token Budget Projections

| Approach | Phase 3+4 Savings | Total Run Savings | Effort |
|----------|-------------------|-------------------|--------|
| Current (no codex) | 0% | 0% | 0 days |
| Stage 1: Skill delegation | ~60% of P3+4 | ~35-40% total | ~2 days |
| Stage 2: Go binary tasks | ~80% of P3+4 | ~50-55% total | ~3-4 days after Stage 1 |
| Combined (Stage 1 + 2) | ~80% of P3+4 | ~50-55% total | ~5-6 days total |

The remaining ~45% of tokens are research (Phase 1/1.5) which requires Opus reasoning and web search — these are not delegatable without quality loss.

## Recommendation

**Ship Stage 1 first (Units 1-3), then Stage 2 (Units 4-7).**

Stage 1 is a 2-day SKILL.md change that immediately cuts token usage by ~35-40%. You'll learn which prompt patterns work well for printing-press specifically (store tables vs workflow commands vs fixes have different context needs). Those learnings directly inform what the Stage 2 Go binary should generate.

Stage 2 is the real win — it eliminates the expensive Claude-side prompt assembly loop and replaces it with deterministic, testable Go code. But it's better built after Stage 1 because:
1. You'll know which context patterns Codex actually needs (not guessing)
2. You'll know which task types benefit most from AST-aware context vs simple head/grep
3. The Stage 1 fallback path means Stage 2 Go binary bugs don't block anything

## Sources & References

- **Origin plan:** [docs/plans/2026-03-27-feat-printing-press-codex-delegation-mode-plan.md](docs/plans/2026-03-27-feat-printing-press-codex-delegation-mode-plan.md)
- **Proven codex patterns:** `~/.claude/skills/osc-nightnight/SKILL.md` lines 1260-1322
- **Scorecard calibration:** `docs/solutions/logic-errors/scorecard-accuracy-broadened-pattern-matching-2026-03-27.md`
- **Scoring architecture:** `docs/solutions/best-practices/steinberger-scorecard-scoring-architecture-2026-03-27.md`
- **Output layout contract:** `docs/solutions/best-practices/checkout-scoped-printing-press-output-layout-2026-03-28.md`
- **SKILL.md codex mode description:** `skills/printing-press/SKILL.md` lines 55-70
- **Phase 3 Build The GOAT:** `skills/printing-press/SKILL.md` lines 847-909
- **Phase 4 Shipcheck:** `skills/printing-press/SKILL.md` lines 911-951
