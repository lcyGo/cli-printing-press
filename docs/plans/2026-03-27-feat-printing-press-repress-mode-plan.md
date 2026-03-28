---
title: "Emboss Mode: Second-Pass Improvement for Generated CLIs"
type: feat
status: active
date: 2026-03-27
---

# Emboss Mode: Second-Pass Improvement for Generated CLIs

## Overview

After the printing press generates a CLI (Phase 0-5, ~1 hour), you have a working tool. But it was built from spec analysis and research - not from using it. The `emboss` command takes an already-generated CLI and runs a fresh improvement cycle: re-research the competitive landscape (things change), run verify to find what's broken, compare against the original scorecard, identify the top 5 improvements, build them, and re-verify.

It's the printing press equivalent of compound-engineering's `plan -> deepen-plan -> work` loop. First pass builds it. Second pass makes it good.

```bash
# First pass: generate from scratch
/printing-press Discord

# Second pass: improve what exists
/printing-press emboss ./discord-cli --spec /tmp/discord-spec.json
```

## Problem Statement

The first press run produces a CLI that scores 65-85/100. The verify command catches what's broken. But there's no structured way to:

1. **Re-research** - the competitive landscape may have changed, new tools may have shipped, new pain points surfaced
2. **Re-evaluate** - now that the CLI exists, the product thesis from Phase 0.8 can be validated against reality
3. **Prioritize improvements** - which of the 50 things we could improve would actually move the needle?
4. **Execute the improvements** - apply fixes, add missing workflow commands, improve the data layer
5. **Re-verify** - prove the improvements actually worked

Currently you'd do this manually: read the old artifacts, re-run some searches, eyeball the code, make changes, run verify. Emboss automates the cycle.

## Proposed Solution

### The Emboss Cycle

```
Input: existing CLI directory + original spec

Step 1: AUDIT        (5 min)   Read the CLI as-is. Run verify + scorecard.
Step 2: RE-RESEARCH  (10 min)  Fresh competitive landscape. What changed since v1?
Step 3: GAP ANALYSIS (5 min)   Compare current state vs what's possible. Top 5 improvements.
Step 4: IMPROVE      (15 min)  Build the top 5. Commit each atomically.
Step 5: RE-VERIFY    (5 min)   Run verify again. Compare before/after.
Step 6: REPORT       (2 min)   Delta report. What improved, what didn't, what's next.
```

Total: ~30-40 minutes. Produces a delta report showing before/after scores.

### Step 1: AUDIT (What do we have?)

Read the existing CLI without changing anything:
- Run `printing-press verify --dir ./api-cli --spec spec.json` to get pass rate
- Run `printing-press scorecard --dir ./api-cli` to get quality score
- Read the README to understand what commands exist
- Read the original Phase 0 artifacts (if they exist in `docs/plans/`)
- Count commands, check data pipeline status, catalog what's there

Output: a baseline snapshot.

### Step 2: RE-RESEARCH (What's changed?)

Run the same Phase 0 searches but with a twist - you already know what exists. Focus on:
- **New competitors:** Has anyone shipped a new CLI for this API since v1?
- **New pain points:** Any new HN/Reddit threads about this API?
- **Spec changes:** Has the API released new endpoints since our spec was frozen?
- **Community signals:** Any social buzz about tools like ours?

Also check npm + GitHub for any CLI that didn't exist when v1 was generated (learning from the PostHog miss).

Output: a "what's new" briefing, not a full Phase 0 redo.

### Step 3: GAP ANALYSIS (What should we improve?)

Compare the audit + re-research against the quality bar. Identify the top 5 improvements by impact:

Scoring framework per improvement:
- **User impact** (1-5): How many users would notice this?
- **Score impact** (1-5): How much would verify/scorecard improve?
- **Effort** (1-5, inverted): How hard is it? (5 = easy, 1 = hard)
- **Freshness** (1-3): Is this based on new research or just cleaning up v1?

Pick the top 5 by composite score. Present to user for approval before building.

Common improvement categories:
- **Fix broken commands** (from verify failures)
- **Add missing workflow commands** (from re-research)
- **Improve data layer** (add tables, fix sync, add FTS5)
- **Polish README** (add cookbook, fix examples)
- **Add new endpoints** (from spec updates)

### Step 4: IMPROVE (Build the top 5)

For each approved improvement:
1. Create a targeted plan (1-2 sentences, not a full plan doc)
2. Implement it
3. Run `go build && go vet` to verify compilation
4. Commit atomically with a conventional message

This is where Codex delegation shines - each improvement is a scoped, independent task.

### Step 5: RE-VERIFY (Prove it worked)

Run the full verify suite again:
- `printing-press verify --dir ./api-cli --spec spec.json`
- `printing-press scorecard --dir ./api-cli`

Compare against the baseline from Step 1.

### Step 6: REPORT (The delta)

```
EMBOSS REPORT: discord-cli
==============================
           Before    After     Delta
Scorecard: 73/100    82/100    +9
Verify:    67%       92%       +25%
Commands:  24        28        +4
Pipeline:  FAIL      PASS      FIXED

Top Improvements Applied:
1. Fixed sync command (was 404ing on /repos endpoint)
2. Added "stale" workflow command
3. Added FTS5 search across issue titles
4. Fixed README cookbook with real examples
5. Wired --csv flag into workflow commands

Remaining Gaps:
- tail command still fails (Gateway not implemented)
- README could use a demo GIF
```

## Where This Lives

### As a SKILL.md Phase

Add to the printing press skill as an optional mode:

```
/printing-press emboss ./discord-cli           # Standard emboss
/printing-press emboss ./discord-cli codex     # Codex-delegated improvements
/printing-press emboss ./discord-cli --spec /tmp/spec.json  # With spec for verify
```

### As a Go Binary Command

Also add to the `printing-press` binary for the mechanical parts:

```bash
printing-press emboss --dir ./discord-cli --spec /tmp/spec.json [--fix] [--api-key TOKEN]
```

The binary handles: audit (verify + scorecard), re-verify, delta report.
The skill handles: re-research (web searches), gap analysis (reasoning), improvements (code changes).

## Implementation

### In the SKILL.md

Add after the Anti-Shortcut Rules section:

```markdown
## Emboss Mode (Second Pass)

When the user runs `/printing-press emboss <dir>`:

1. This is NOT a from-scratch run. The CLI already exists.
2. Read the existing CLI directory. Run verify + scorecard to get a baseline.
3. Read any existing Phase 0-5 artifacts in docs/plans/ for this API.
4. Run ONLY the research steps that look for what's NEW (not a full Phase 0 redo).
5. Identify top 5 improvements by impact. Present to user for approval.
6. Build each improvement atomically. Commit each.
7. Re-verify. Report the delta.

The emboss should take ~30 minutes, not ~1 hour. It's surgical, not generative.
```

### In the Go Binary

Add `printing-press emboss` command that wraps:
1. `verify` (baseline)
2. `scorecard` (baseline)
3. User does improvements (skill-driven)
4. `verify` (after)
5. `scorecard` (after)
6. Delta report

The delta report is the new artifact. It goes in `docs/plans/<today>-emboss-<api>-cli-delta.md`.

## The Analogy

| Compound Engineering | Printing Press |
|---------------------|---------------|
| `/ce:plan` | `/printing-press <API>` (first pass) |
| `/deepen-plan` | Phase 0-1 research enrichment |
| `/ce:work` | Phase 2-4 generation + build |
| `/ce:review` | Phase 5 scorecard + verify |
| **No equivalent** | **`/printing-press emboss` (second pass)** |

Emboss fills the gap. It's what you do after the first run when you look at it and say "this is good but not great." It's the deepen-plan + work cycle, applied to an already-generated CLI.

## Acceptance Criteria

- [ ] `/printing-press emboss <dir>` runs the 6-step cycle
- [ ] Step 1 produces a baseline (verify pass rate + scorecard score)
- [ ] Step 2 searches for new competitors/pain points (not full Phase 0)
- [ ] Step 3 identifies and ranks top 5 improvements
- [ ] Step 4 builds improvements with atomic commits
- [ ] Step 5 re-verifies and compares to baseline
- [ ] Step 6 produces a delta report in docs/plans/
- [ ] Total time: ~30-40 minutes (not another full hour)
- [ ] Codex delegation works for Step 4 improvements
- [ ] The cycle is repeatable - you can emboss multiple times

## The Name

**`emboss`** - run it through the press again. A printing press pun that communicates exactly what it does.

Alternatives considered: refine, polish, hone, temper, sharpen, elevate, reforge, proof, second-edition. All fine words but none have the press connection.

## Sources

- Compound engineering plan/deepen-plan/work cycle as the inspiration
- The GitHub CLI run from this session: first pass scored 73/100 with broken sync, proving the need for a structured second pass
- The verify command (built today) as the mechanical foundation for baseline/re-verify
