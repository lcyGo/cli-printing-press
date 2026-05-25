---
decision: 2026-05-25-005-espn-playbooks-findings
status: validated-structural-real-dogfood-pending
created: 2026-05-25
plan: docs/plans/2026-05-25-005-feat-cli-playbooks-and-notes-plan.md
predecessor:
  - docs/decisions/2026-05-25-003-espn-dogfood-cascade-findings.md
  - docs/decisions/2026-05-25-004-recall-generalization-findings.md
---

# ESPN playbooks + notes — structural validation + backport queue

Plan 005 shipped the "smart local notes" primitive end-to-end on ESPN: schema, teach surfaces, recall envelope extension, SKILL.md decision-tree expansion, four hand-authored playbooks distilled from the 2026-05-25 dogfood transcripts. 254 tests pass across 13 packages.

This doc captures what's structurally validated, the one killer cross-entity replay demonstration, what's deferred to live agent dogfood for the final tool-call-count proof, and the backport queue for `cli-printing-press` canonical templates.

## What ships (espn-side, validated)

### U1 — Schema migration for learning_playbooks

New table keyed on the structural query family (entities stripped). Holds optional `playbook_json` + `notes_text`; either may be empty, both empty rejected. Reflective migration (plan 003 U1 pattern) picks it up automatically on legacy DBs. UNIQUE index on `query_family` so re-teach collapses cleanly. 8 store tests covering happy path, idempotency, concurrent writes, notes-only rows, empty-content rejection, family-empty rejection, get-not-found, table-exists-after-Open.

### U2 — Playbook type + slot resolution

`learn.Playbook` Go type matches the JSON shape: steps (each `cmd` or `client_side`), entity_slots, expected_tool_calls. `learn.ParsePlaybookFile` parses + validates. `learn.ResolveSlots` walks declared slots, resolves each to a canonical via the same `EntityResolver` interface `PromoteEntities` uses, and emits `{token, canonical}` per slot. Composes with the recall path without duplicating canonical resolution logic. 11 unit tests across happy/edge/error categories.

### U3 — Recall envelope surfaces playbook + notes

`learn.Result` and the CLI `recallEnvelope` JSON shape both gain `playbook` (with `slots_resolved` map) and `notes` top-level fields. Recall looks up `learning_playbooks` by `QueryFamily(normalized)` after PromoteEntities runs. Surfaces regardless of whether per-resource path hit. Errors swallowed so the legacy contract (recall never fails) is preserved. 5 new tests covering single-entity, multi-entity / different-entity family collision, no-match, notes-only, and playbook + resource-hit coexistence.

### U4 — teach-playbook + teach --playbook-file extension

Two write paths. Standalone `teach-playbook --query --playbook-file --notes-file` for recipe-only recording. Extended `teach --playbook-file --playbook-notes --playbook-notes-file` for the common end-of-session shape where resource learning AND playbook land in one call. Playbook-side failures are non-fatal: they log to teach.log and the resource learning still ships (graceful degrade). New `playbook list` for inspection. 9 CLI integration tests covering happy path, notes-only, missing file, malformed JSON, requires-content, both-surfaces, graceful-degrade, list-empty, list-populated.

### U5 — SKILL.md decision tree extended to six branches

Adds two new branches at the top of the tree: (1) playbook present -> read notes verbatim, replay steps with slot substitutions; (2) notes-only -> read verbatim before discovery. Existing four branches (exact hit / partial hit / mismatch / cold) follow. Adds a "Step 5: record a playbook when discovery took >5 calls" section with both write surfaces and the case-sensitivity gotcha (PPG vs ppg) documented. Skill verifier green; local mirror in sync.

### U6 — Four authored playbooks shipped

- `season_recap` (`internal/cli/playbooks/season_recap.json` + `_notes.md`): Warriors / Pistons / Lakers / any team. Captures byathlete `seasontype=2`, duplicate-label gotcha in categories payload, leaders --team silently dropped, PPG vs ppg normalization warning.
- `last_game_stats` (`internal/cli/playbooks/last_game_stats.json` + `_notes.md`): Carter Bryant style. Documents that `search` is unreliable for rookies, `compare` is the working athlete-resolution path, and the boxscore player-stats layout.
- `league_top_bottom` (`internal/cli/playbooks/league_top_bottom.json` + `_notes.md`): MLB / NBA / NFL top-3 / bottom-3 per division. Ships the team-abbr -> division mapping for MLB and NBA (NFL groups arrive pre-divided from ESPN).
- `team_today` (`internal/cli/playbooks/team_today.json` + `_notes.md`): Niners-style same-day game lookup. Documents nextEvent shape and the `scoreboard --dates` (plural) gotcha. NOTE: this family hits the empty-family edge case described below.

3 authored-playbook tests verify every JSON parses, every JSON has a matching notes file, and that the season-recap example queries across teams collapse to the same family `end led ppg rpg season spg`.

## The killer demonstration: cross-entity recipe transfer

Real binary, fresh DB, with only the Warriors-installed season_recap playbook:

```
$ espn-pp-cli recall "how did Lakers end the season who led in ppg rpg spg" --agent

found: False                                                  # no resource learning (Lakers never directly taught)
has_playbook: True                                            # but the family DOES match
playbook_family: "end led ppg rpg season spg"                 # same family the Warriors recall would compute
slots: {'$TEAM': {'canonical': 'Los Angeles Lakers', 'token': 'Lakers'}}
notes_excerpt: "# Season recap query family\n\nThe `espn-pp-cli leaders ...`"
```

The Warriors teach is being replayed against a Lakers query with the slot bound to Lakers. This is the "make every team's season recap fast even when only one was taught" behavior the plan promised, end-to-end working.

Cold/unrelated queries correctly miss:

```
$ espn-pp-cli recall "what time is the sun setting today" --agent
found: False, has_playbook: False, warnings: ["no_learnings_for_query_family"]
```

MLB top 3 fires for the MLB-specific family:

```
$ espn-pp-cli recall "top 3 mlb teams in each division" --agent
has_playbook: True, playbook_family: "3 division each mlb teams top"
notes_excerpt: "# League top/bottom per division query family\n\nESPN's `standings`..."
```

## What's deferred to live agent dogfood

The final tool-call-count proof per plan R9 requires firing the patched binary through an interactive Claude session against the 5 transcript shapes and counting calls before vs after. That can't run as a structural test inside this plan execution session — it requires the user (or an interactive agent session) reading the playbook + notes and choosing to replay them.

The structural foundation that an agent following SKILL.md's six-branch protocol would observe:
- Playbook + notes surface in the recall envelope (verified).
- Slot bindings carry the resolved canonical (verified).
- Notes surface verbatim (verified).
- SKILL.md prose tells the agent to read notes first, then replay steps (shipped).

What the next dogfood session will measure: did the agent actually follow the protocol; did tool-call count drop. The expected drops based on the transcripts:

| Family | Original transcript calls | Expected with playbook |
|---|---|---|
| Season recap (Warriors S1) | ~15 | ~4 (playbook explicit) |
| Season recap (Pistons) | ~15 | ~4 |
| Last-game stats (Carter Bryant) | ~12 | ~3 |
| MLB top 3 | ~10 | ~1 |
| MLB bottom 3 | ~10 | ~1 |

If the agent runs much more than this, the playbook is either being ignored (SKILL.md guidance not strong enough) or the family-key didn't collide as expected. Both are recoverable: tighten SKILL.md or adjust the playbook family-example queries.

## Surprises during execution

1. **Empty-family edge case for stopword-heavy queries.** "Niners game tonight" produces an empty query_family because Niners is an entity and both `game` + `tonight` are espn stopwords — nothing left for the structural key. `teach-playbook` rejects the empty-family upsert. Workaround: pick a teach-anchor query with at least one content token (e.g., "next $TEAM scheduled event"). Real fix: special-case empty families with a hash of `query_family_examples` so degenerate cases still have a key. Deferred to follow-up.

2. **Case-sensitive ALL-CAPS entity promotion changes family keys.** "PPG, RPG, SPG" (uppercase) auto-promote to entities via the all-caps rule and get stripped from the family, while "ppg rpg spg" (lowercase) stay as content. The two cases land in DIFFERENT families. Documented in season_recap_notes.md as an agent-side gotcha (always lowercase stat abbreviations in recall calls); a deeper fix would be to consult `entity_lookups` before promoting all-caps tokens.

3. **Athlete-slot resolution requires player coverage in `entity_lookups`.** "Stephen Curry last game" fires the last_game_stats playbook but `$ATHLETE` slot resolves empty because Curry isn't seeded — only team rosters are. The agent still gets the playbook + notes (still useful), but the slot binding requires player coverage. Deferred: extending entity_lookups seeds to include star players + active rosters per team.

4. **The CLI's stopword list shapes which queries can be "smart-noted".** Queries whose meaning lives entirely in entities + stopwords end up with empty families. Most useful queries have at least one content noun/verb that survives normalization (recap, leaders, schedule, score, history, etc.) — but the team-today / "tonight" shape is the canonical degenerate case.

## What needs to backport to canonical templates

Every fix in plan 005 is engine-shape (not espn-specific). Cross-CLI port queue targeting `mvanhorn/cli-printing-press/internal/generator/templates/`:

- **`store.go.tmpl`** — add `learning_playbooks` CREATE TABLE + UNIQUE INDEX to the migration list. Reflective migration handles the rest. ~25 LOC.
- **`learn/playbooks.go.tmpl`** — new file. Port `learn.Playbook`, `learn.PlaybookStep`, `learn.ResolvedPlaybook` types + `ParsePlaybookFile`, `MarshalPlaybook`, `QueryFamily`, `ResolveSlots`. Domain-neutral; mirrors espn-side promote.go shape.
- **`learn/recall.go.tmpl`** — append the playbook lookup block (5 lines + envelope attachment) after the existing similar-shape mismatch surfacing block. Mirror the espn-side change.
- **`cli/teach.go.tmpl`** — add `--playbook-file`, `--playbook-notes`, `--playbook-notes-file` flags; call `upsertPlaybookFromTeach` after `UpsertLearning`. Mirror espn-side.
- **`cli/teach_playbook.go.tmpl`** — new file. Port standalone `teach-playbook` + `playbook list` commands.
- **`cli/root.go.tmpl`** — register `newTeachPlaybookCmd` + `newPlaybookCmd` in the AddCommand chain.
- **`skill.md.tmpl`** — extend the Automatic Learning section to six-branch tree (currently four). Mirror espn-side SKILL.md changes.
- **`internal/store/playbooks.go.tmpl`** — new file. Port `UpsertPlaybook`, `GetPlaybookByFamily`, `ListPlaybooks`, `PlaybookRow`. Domain-neutral; safe to template directly.

Also needed: a `playbooks/` directory convention per CLI. Each printed CLI's authoring agent ships its own JSON + MD files distilled from its CLI's known gotchas. The generator's job is the engine; the per-CLI playbooks are authored content (like `learn_init.go` seeds today).

Sequencing recommendation: land all template changes in one PR against `cli-printing-press`, regenerate ESPN from templates as a sanity check (should match the hand-applied espn-side changes), then sweep-tool over the remaining published library CLIs in a follow-up.

## Sequencing implication for plan 2026-05-25-001

This unblocks:
- **U12 prediction-goat dogfood**: now has a concrete primitive to validate against. Does prediction-goat's recall benefit similarly from notes about Polymarket payload gotchas? Same primitive should ship there.
- **U13 retrospective**: this doc plus the 003 and 004 findings docs form the cross-plan evidence base.
- **U14 reconcile divergences**: the template backport queue above is the concrete work.
- **U7 sweep tool rewrite** (still deferred): becomes more valuable now — retrofitting existing published CLIs with playbook support is the highest-leverage post-merge move.

If the live agent dogfood shows the expected tool-call drops, every PP CLI inheriting this primitive becomes structurally faster for repeated query families. If it doesn't (agent ignores the playbook, family-keys don't collide as expected, SKILL.md guidance is too weak), the retrospective for the cross-CLI port covers what to tighten before propagating.

## Files added in espn-side (PR #851)

**Library (`mvanhorn/printing-press-library`):**
- New: `library/media-and-entertainment/espn/internal/store/playbooks.go` + `_test.go`
- New: `library/media-and-entertainment/espn/internal/learn/playbooks.go` + `_test.go`
- New: `library/media-and-entertainment/espn/internal/cli/teach_playbook.go` + `_test.go`
- New: `library/media-and-entertainment/espn/internal/cli/playbooks_authored_test.go`
- New: `library/media-and-entertainment/espn/internal/cli/playbooks/season_recap.json` + `_notes.md`
- New: `library/media-and-entertainment/espn/internal/cli/playbooks/last_game_stats.json` + `_notes.md`
- New: `library/media-and-entertainment/espn/internal/cli/playbooks/league_top_bottom.json` + `_notes.md`
- New: `library/media-and-entertainment/espn/internal/cli/playbooks/team_today.json` + `_notes.md`
- Modified: `library/media-and-entertainment/espn/internal/store/store.go` (migration entry)
- Modified: `library/media-and-entertainment/espn/internal/learn/recall.go` (envelope attachment)
- Modified: `library/media-and-entertainment/espn/internal/cli/teach.go` (--playbook-file flags)
- Modified: `library/media-and-entertainment/espn/internal/cli/root.go` (register new commands)
- Modified: `library/media-and-entertainment/espn/SKILL.md` (six-branch decision tree + Step 5)

**Generator (`mvanhorn/cli-printing-press`):**
- New: this retrospective doc.
- Pending: template ports (queued in next plan).
