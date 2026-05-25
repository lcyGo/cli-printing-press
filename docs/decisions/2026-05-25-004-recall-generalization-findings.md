---
decision: 2026-05-25-004-recall-generalization-findings
status: validated
created: 2026-05-25
plan: docs/plans/2026-05-25-004-fix-espn-recall-generalization-plan.md
predecessor: docs/decisions/2026-05-25-003-espn-dogfood-cascade-findings.md
---

# ESPN recall generalization — dogfood findings + backport queue

After the 003 cascade landed, four ESPN dogfood sessions in the same hour showed the loop still didn't generalize across entities. This plan addressed the layered failure surfaced in those transcripts. Every fix is validated against the real binary.

## Dogfood scenarios (post-patch behavior)

### Scenario 1 — Cold first ask

```
$ espn-pp-cli recall "Carter Bryant playoff stats" --agent
{
  "found": false,
  "warnings": ["no_learnings_for_query_family"]
}
```

Cold-start envelope still honest. No spurious warnings.

### Scenario 2 — Same-alias second ask

```
$ espn-pp-cli teach --query "next two weeks of home games for the Mariners" \
    --resource 12 --resource-type teams
$ espn-pp-cli recall "next two weeks of home games for the Mariners" --agent
{
  "found": true,
  "match_score": 1,
  "results": [{"resource_id": "12", "entity_match": "exact", ...}]
}
```

Literal-alias path unchanged; full-score hit.

### Scenario 3 — Cross-alias second ask (regression check from cascade U3)

```
$ espn-pp-cli teach --query "Niners game tonight" --resource 401547432 \
    --resource-type events
$ espn-pp-cli recall "49ers game tonight" --agent
{
  "found": true,
  "results": [{"resource_id": "401547432", "warnings": ["cross_alias_match"], ...}]
}
```

Cascade U3 cross-alias canonical resolution still works under the new U4 ratio-based gate (replacing the prior boolean at-threshold hack). No regression.

### Scenario 4 — Different-entity-same-shape ask (the killer case)

```
$ espn-pp-cli teach --query "How are the Mariners doing this season so far ..." \
    --resource 12 --resource-type teams
$ espn-pp-cli recall "how are the mets doing so far this year" --agent
{
  "found": false,
  "query_entities": ["mets"],
  "warnings": ["similar_shape_different_entity:Seattle Mariners"]
}
```

Before this plan: same query returned `no_learnings_for_query_family` even though the Mariners row was retrievable at match_score 0.8 behind `--debug-mismatches`. After: the agent sees the relevant alternative learning by name. The agent can now decide whether the user genuinely meant Mets (run discovery) or whether the question was ambiguous (ask the user).

## What landed (espn-side, validated)

### U1 — Symmetric entity promotion at teach time

**Files:**
- New: `library/media-and-entertainment/espn/internal/learn/promote.go`
- Modified: `library/media-and-entertainment/espn/internal/learn/recall.go`
- Modified: `library/media-and-entertainment/espn/internal/cli/teach.go`
- New tests in: `library/media-and-entertainment/espn/internal/learn/promote_test.go` and `internal/cli/teach_test.go`

Extracts the post-Normalize entity_lookups promotion from recall.go into a shared `PromoteEntities(NormalizedQuery, EntityResolver) NormalizedQuery` helper. Exports `CanonicalResolver` + `NewCanonicalResolver` so teach can build the same resolver shape recall uses. Teach now calls the helper before `UpsertLearning`, so the stored `query_entities` column is symmetric with what recall produces.

Validated end-to-end: teaching `"how are the mariners doing this year"` against the real binary stores `query_entities: ["mariners"]` instead of null. Previously the capitalization-based extractor missed lowercase entity tokens, leaving the cross-alias resolver with nothing to compare against on the stored side.

### U2 — Opportunistic backfill for legacy null-entity rows

**Files:**
- Modified: `library/media-and-entertainment/espn/internal/learn/recall.go`
- New tests in: `library/media-and-entertainment/espn/internal/learn/recall_canonical_test.go`

When recall encounters a stored row with `query_entities = null` (written before U1 landed), walk the lowercased `query_pattern` tokens through the canonical resolver to derive an effective entity slice for this call. Read-only — the stored column stays null, so user data isn't silently rewritten.

Lets pre-existing data.db files participate in cross-alias matching without forcing users to rebuild.

### U3 — Surface similar-shape mismatches in the main envelope

**Files:**
- Modified: `library/media-and-entertainment/espn/internal/learn/match.go` (new `WarningSimilarShapeDifferentEntity` constant)
- Modified: `library/media-and-entertainment/espn/internal/learn/recall.go` (track stored canonicals for routed mismatches; emit envelope warnings after row loop)
- New tests in: `library/media-and-entertainment/espn/internal/learn/recall_canonical_test.go`

When a row routes to `mismatches[]` because the entity differs but structural shape matches, emit a top-level warning of the form `similar_shape_different_entity:<canonical>`. Visible in the default envelope — not just behind `--debug-mismatches`. Agents reading `found: false` no longer see misleading `no_learnings_for_query_family` when a structurally-similar learning for a different entity exists.

The default envelope stops lying about cold-start when it isn't one.

### U4 — Multi-entity `queryStructural` + separate cross-alias Jaccard floor

**Files:**
- Modified: `library/media-and-entertainment/espn/internal/learn/patterns/extract.go`
- Modified: `library/media-and-entertainment/espn/internal/learn/recall.go` (added `Opts.CrossAliasJaccardMin`; replaced boolean at-threshold with ratio gate)
- New tests in: `library/media-and-entertainment/espn/internal/learn/patterns/extract_test.go` and `recall_canonical_test.go`

Two fixes:

1. `queryStructural` signature changed from `(queryPattern, entity string)` to `(queryPattern string, entitiesToStrip []string)`. Strips every stored entity rather than just `members[0].queryEntities[0]`. Multi-entity teaches now group with their shape-peers (the inferPattern guard skips single-slot emission for multi-entity members until binding logic is upgraded).

2. New `Opts.CrossAliasJaccardMin` (default 0.3, separate from `JaccardMin`'s 0.6). Replaces the prior boolean-at-threshold hack with a ratio-based gate using `max(literalJaccard, canonicalJaccard)`. Cross-alias matches differ on literal entities, so non-entity Jaccard is naturally lower; a single tight floor was gating out legitimate paraphrase hits.

### Real-binary-only fix — similar-shape mismatches admitted to mismatches[]

**Files:**
- Modified: `library/media-and-entertainment/espn/internal/learn/recall.go`

Dogfood scenario 4 surfaced one final gap not caught by the test fixtures: the Mariners-vs-Mets case has score 0.43, below jMin=0.6, and canonicals don't overlap. The row was dropped at the row-loop gate before reaching mismatches[], so U3's warning had nothing to emit against.

Extended the cross-alias fallback into three branches:
- `canonicalOverlap` → cross-alias hit candidate (downstream entity-match promotion)
- no overlap, both sides have entities, score ≥ crossAliasMin → similar-shape mismatch candidate (routes to mismatches[], drives U3 envelope warning)
- otherwise → drop

Validated against the dogfood session 4 transcript: recall now returns `similar_shape_different_entity:Seattle Mariners` instead of `no_learnings_for_query_family`.

## What needs to backport to canonical templates

Every fix in this plan is engine shape (not espn-specific), so the canonical learn templates need parallel updates so future-printed CLIs inherit the corrected loop. Target files in `cli-printing-press`:

- `internal/generator/templates/learn/recall.go.tmpl` — port U1 promotion-helper call (replace inline promotion block with `PromoteEntities`), U2 opportunistic backfill, U3 mismatch-canonical tracking + envelope warning, U4 `CrossAliasJaccardMin` opt + ratio-based gate, similar-shape branch in the fallback switch.
- `internal/generator/templates/learn/promote.go.tmpl` — new file mirroring `library/.../promote.go` (helper + EntityResolver interface).
- `internal/generator/templates/learn/match.go.tmpl` — add `WarningSimilarShapeDifferentEntity` constant.
- `internal/generator/templates/learn/patterns/extract.go.tmpl` — `queryStructural` slice signature, multi-entity guard at inferPattern site.
- `internal/generator/templates/cli/teach.go.tmpl` — call the shared promotion helper before `UpsertLearning`.

These changes are mechanical; the espn-side code is the reference. A tracking issue in cli-printing-press is the right home; the sweep tool work (plan 003 U7, still deferred) is the path to retrofit already-published CLIs.

## What needs to backport to prediction-goat

Prediction-goat is the canonical lineage for the learn loop; the same architectural gaps that surfaced in espn likely exist there too. Audit checklist:

- **U1 symmetric promotion**: prediction-goat's teach.go probably stores raw `normalized.Entities` without entity_lookups promotion. Same fix applies.
- **U2 opportunistic backfill**: same — legacy rows from before symmetric teach would benefit.
- **U3 envelope-level mismatch warning**: prediction-goat's recall.go almost certainly doesn't surface mismatches outside `--debug-mismatches`. Port the constant + tracking logic + emission.
- **U4 cross-alias Jaccard floor**: prediction-goat's recall.go may or may not have a cross-alias path; if it does, the same separate-floor argument applies. If it doesn't, that's a deeper retrospective question for plan 2026-05-25-001 U12-U14.

## Surprises during execution

1. **The "so" token isn't a stopword anywhere.** Default `entities.Config` stopwords cover articles, prepositions, modal verbs, question words, conjunctions — but not "so". espn's CLI-specific stopwords are domain shape ("game", "vs", "stats", etc.). Scenario 4's normalized "doing far so year" wasn't stripped further. That's fine for matching (the Jaccard stays meaningful) but worth flagging — agents looking at the `normalized` field might wonder why "so" survived.

2. **Recall validateResource still classifies Mismatch correctly when the resource isn't in the local store.** The Mariners team row (resource_id=12) isn't in the test environment's `resources` table; validateResource falls back to `ClassifyEntityMatch(queryEntities, storedEntitySlice)` which returns Mismatch on the entity-set differences. U3's mismatch warning fires off this verdict cleanly without needing the resource present.

3. **The similar-shape fallback branch needed to land at runtime, not just in tests.** The plan's test scenarios used short query/stored pairs where Jaccard naturally cleared 0.6; the test suite passed before scenario 4 was dogfooded. The 12-word stored query vs the 7-word recall query in scenario 4 produced Jaccard 0.43 — below the floor that the test pairs hit. Always run real-binary scenarios; synthetic fixtures may flatter the implementation.

## Sequencing implication for plan 2026-05-25-001

This plan unblocks:
- **U12 prediction-goat dogfood**: now has concrete backport targets and the cross-alias / similar-shape / promotion-symmetry framework to test against.
- **U13 retrospective**: this doc plus the cascade findings doc are both inputs.
- **U14 reconcile divergences**: backport list above.
- **U7 sweep tool rewrite** (still deferred): becomes the path to retrofit existing published CLIs with all the cascade + generalization fixes.
- **U15-U18 auto-rerank**: stays parked until the generalization layer is stable across templates.
