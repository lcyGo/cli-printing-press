---
decision: 2026-05-25-003-espn-dogfood-cascade-findings
status: validated
created: 2026-05-25
plan: docs/plans/2026-05-25-003-fix-espn-dogfood-cascade-plan.md
feeds:
  - docs/plans/2026-05-25-001-feat-generic-recall-rerank-middleware-plan.md (U13/U14)
---

# ESPN dogfood cascade — findings + backport queue

The first ESPN dogfood session (2026-05-25, post-U21 install) surfaced four issues. This doc captures what each fix taught us, what's now validated end-to-end on ESPN, and what needs to flow back into cli-printing-press canonical templates + prediction-goat.

## What landed (espn-side, validated)

### U1 — Reflective schema migration

**File:** `library/media-and-entertainment/espn/internal/store/store.go` (modified) + `library/media-and-entertainment/espn/internal/store/migrations_test.go` (new).

Two-pass `migrate()`. Pass 1 runs CREATE TABLE IF NOT EXISTS statements (no-op on existing tables). Pass 2 spins up an in-memory canonical reference using the same migration list, enumerates canonical columns via PRAGMA table_info, diffs against the user's tables, and emits ALTER TABLE ADD COLUMN for each missing column. Pass 3 runs the remaining migrations (indexes, virtual tables) — now safe.

**Dogfood verification:** stale events table missing `season_year`/`season_type`/`week`/`notes` heals on next Open without manual `rm`. Legacy data survives. `idx_events_season` succeeds.

**Backport target (cli-printing-press):** `internal/generator/templates/store.go.tmpl`. The reflective migration pattern is generic — every learn-enabled CLI inherits it once the template gets the same shape. Not yet ported.

### U2 — SKILL.md teach-discipline

**File:** `library/media-and-entertainment/espn/SKILL.md` (modified).

Ports prediction-goat's "Automatic learning" section verbatim (four-branch decision tree, two-call protocol, envelope semantics) with espn-specific examples. Local mirror at `~/.claude/skills/pp-espn/SKILL.md` updated too.

**Dogfood verification:** Section present in installed skill. The next dogfood session against the patched binary should show whether the language actually changes agent behavior — initial signal positive (the first session that triggered this plan happened *without* the section).

**Backport target (cli-printing-press):** `internal/generator/templates/skill.md.tmpl`. Already had its own Automatic Learning section? Worth checking — if absent, port. If present but different shape, reconcile.

### U3 — Cross-alias canonical resolution

**Files:**
- `library/media-and-entertainment/espn/internal/learn/recall.go` (modified)
- `library/media-and-entertainment/espn/internal/learn/match.go` (modified — new warnings)
- `library/media-and-entertainment/espn/internal/learn/recall_canonical_test.go` (new, 6 tests)

Adds:
- `canonicalResolver` type with per-call entity → canonical(s) cache via entity_lookups
- Post-Normalize entity promotion (any non-entity token in entity_lookups gets promoted into normalized.Entities — fixes numeric-prefix aliases like "49ers")
- Cross-alias score fallback (any canonical overlap → score = jMin so ambiguous-alias cases pass)
- Cross-alias entity-classification promotion (Mismatch → Exact when canonicals overlap)
- `WarningCrossAliasMatch` (per-hit) + `WarningAmbiguousAlias` (top-level)

**Dogfood verification:** Teach "Niners game tonight" → `49ers`, `SF`, `Niners` all hit. `Cowboys`, `Lakers` correctly miss. Cross-alias hits carry the `cross_alias_match` warning. The seeded entity_lookups table (413 rows across NFL/NBA/MLB/MLS) is now load-bearing on the read side from a cold start.

**Backport target (cli-printing-press):** **Done in same session.** `internal/generator/templates/learn/recall.go.tmpl` + `internal/generator/templates/learn/match.go.tmpl` mirror the espn-side change. Golden fixture updated.

### U21 (from prior plan) — Entity-only Jaccard fallback

Already shipped before this plan's work began. Keeps recall functional when `NonEntityNormalized` is empty on both sides (canonical engine returns Jaccard=0 on empty token sets). Reconstructs stored non-entity tokens from `query_entities` column.

**Backport target (cli-printing-press):** Already done in PR #2200 (canonical foundation).

## What didn't land — U4 (deferred)

### U4 — `NormalizeQuery` write-path shape mismatch

**Status:** Deferred. Architectural cleanup, not bug fix.

**Rationale:** U3 + U21 together deliver the dogfood-working loop. U4 would route the write path through `learn.Normalize(cfg)` so `query_pattern` is stored already entity-aware, eliminating U21's reconstruction logic. But U21's reconstruction is cheap and the architectural argument ("one normalizer not two") doesn't deliver user-visible improvement now that the read side handles both shapes correctly.

**When to revisit:** When the read-path reconstruction logic starts coupling tightly to specific schema column shapes (making future schema evolution harder), OR when the architectural inconsistency surfaces a real bug. Until then, the current shape is acceptable.

**Open question:** Whether prediction-goat's canonical engine should also adopt entity-aware write-path normalization. If yes, this becomes a generator template change. If no (PG's two-normalizer split is intentional), then the cleanup is purely cosmetic.

## What needs to backport to canonical templates

### Already mirrored to cli-printing-press in this session

- **U3 cross-alias resolution** — `feat/learn-canonical-restore` branch (PR #2200's stack) carries the same canonical resolver + entity promotion + cross-alias score fallback + WarningCrossAliasMatch/AmbiguousAlias constants in the canonical templates. Golden fixture updated.

### Still owed to cli-printing-press

- **U1 reflective schema migration** — needs to land in `internal/generator/templates/store.go.tmpl`. The implementation is generic; the port is mechanical. Adds 100-150 lines to the template + reuses ESPN's test pattern for `migrations_test.go.tmpl`. **Recommended next:** ship as part of plan 2026-05-25-001 U7 (sweep tool rewrite, currently deferred) OR as a focused PR before/after U7. Without this, every newly-printed learn-enabled CLI inherits the same in-place-column-add risk ESPN had.

- **U2 SKILL.md** — verify whether `internal/generator/templates/skill.md.tmpl` already has an Automatic Learning section gated by `{{if .Spec.Learn.Enabled}}`. If yes, audit its shape against espn's. If no, port the section.

## What needs to backport to prediction-goat (Plan A U14 territory)

Prediction-goat's canonical engine has the same architectural shape as the espn-installed package, so the bugs ESPN dogfood surfaced likely exist in prediction-goat too. Recommended audit:

### Confirmed needed

- **U3 cross-alias canonical resolution** — prediction-goat's recall.go does NOT have a cold-start cross-alias path. The only generalization path is the pattern engine (requires 3+ similar teaches before any cross-alias fires). Cold-start aliases (one teach → recall via different alias) silently miss. ESPN's fix is the canonical answer; PR against prediction-goat (or its publish workspace) would carry the same resolver + score fallback + WarningCrossAliasMatch.

### Probably needed

- **U1 reflective schema migration** — prediction-goat's store.go.tmpl also doesn't have ALTER TABLE migrations. If prediction-goat has added columns to any table in-place since release, the same bug exists. Audit and apply same fix.

### Not needed

- **U4 NormalizeQuery cleanup** — prediction-goat is the source of the two-normalizer split. Adopting the U4 cleanup there would be a deeper architectural refactor. Out of scope for this cascade.

## Surprises during execution

1. **The "49ers" numeric-prefix problem** — the entity extractor's capitalization-based detection doesn't catch alphanumeric-with-leading-digit tokens. I initially expected the seeded entity_lookups alone to be sufficient for cross-alias matching, but the extractor never produced `["49ers"]` as a query entity — it produced `[]` because `49ers` got classified as a non-entity content token. Required adding the post-Normalize entity promotion (any non-entity token that resolves in entity_lookups gets promoted) as a separate fix beyond canonical resolution.

2. **Ambiguous aliases need boolean overlap, not Jaccard** — "SF" resolves to both San Francisco 49ers (NFL) AND San Francisco Giants (MLB). With Jaccard scoring on canonical sets, `1/2 = 0.5 < 0.6` — gates out a legitimate match. Switched to boolean `setIntersects` at-threshold so any overlap counts; ambiguity surfaces via `WarningAmbiguousAlias` for the agent to handle.

3. **Reflective migration via in-memory canonical** — initial plan called for declaring canonical column lists alongside the CREATE TABLE statements. Pivoted to spinning up an in-memory SQLite with the migration list itself, then PRAGMA-comparing the user's DB against the canonical. Cleaner: the CREATE TABLE statement stays the single source of truth, and any future column addition flows into reconciliation automatically.

## Sequencing implication for plan 2026-05-25-001

This plan unblocks the previously-deferred work:

- **U12 (prediction-goat dogfood with real data)** can now proceed with concrete espn-validated findings to compare against. Specifically, U12 should validate whether prediction-goat exhibits the same cross-alias cold-start gap and stale-schema migration risk.
- **U13 (retrospective)** has the espn findings as input.
- **U14 (reconcile divergences)** has a concrete backport list: U1 + U3 (cross-alias) are confirmed needed; U2 (SKILL.md) needs audit; U4 (NormalizeQuery) is deferred.
- **U15-U18 (auto-rerank feature)** stays parked until backports land.
