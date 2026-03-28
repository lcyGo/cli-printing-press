---
title: "Audit: Does Phase 3 GOAT enforce agent principles? Does scoring need updating?"
type: research
status: active
date: 2026-03-28
---

# Audit: GOAT Phase + Agent Principles + Scoring Alignment

## Question 1: Does Phase 3 (Build The GOAT) enforce agent design principles?

### What the SKILL.md says

Phase 3 Priority 1 says "matched and beaten with agent-native output." But it doesn't spell out WHAT agent-native means for each absorbed feature. It's an adjective, not a checklist.

### What's missing

The absorb manifest table has columns: Feature | Best Source | Our Implementation | Added Value. But "Added Value" is freeform. There's no enforcement that every absorbed feature gets:

- `--json` output
- `--dry-run` (for mutation commands)
- `--stdin` (for batch input)
- `--select` field filtering
- Typed exit codes
- `--compact` token-efficient mode
- Auto-JSON when piped (no `--json` needed)
- No interactive prompts (fully scriptable)

**Recommendation:** Add an "Agent Checklist" to Phase 3 that runs AFTER building each absorbed feature:

```
For each command built, verify:
[ ] --json works and produces valid JSON
[ ] --dry-run works for any mutation command (POST/PUT/PATCH/DELETE)
[ ] --select filters fields correctly
[ ] Exit code is typed (0/2/3/4/5/7)
[ ] No interactive prompts (works in CI/agent context without TTY)
[ ] --compact returns only high-gravity fields
```

This is basically what `scoreAgentNative` checks, but applied during BUILD, not just during scoring after the fact.

### Verdict: Partial. The intent is there but the enforcement isn't.

---

## Question 2: Does the scoring system need to change?

### Current scorecard dimensions (18 total)

**Tier 1 - Infrastructure (12 dimensions, 0-120 raw -> 50 normalized):**
1. OutputModes (0-10) - --json, --csv, --select, --quiet, --compact
2. Auth (0-10)
3. ErrorHandling (0-10) - typed exits, retry, actionable messages
4. TerminalUX (0-10)
5. README (0-10)
6. Doctor (0-10)
7. **AgentNative (0-10)** - --json, --select, --dry-run, --stdin, --yes, no prompts
8. LocalCache (0-10)
9. Breadth (0-10)
10. Vision (0-10)
11. Workflows (0-10)
12. Insight (0-10)

**Tier 2 - Domain Correctness (6 dimensions, 0-50 raw -> 50 normalized):**
13. PathValidity (0-10)
14. AuthProtocol (0-10)
15. DataPipelineIntegrity (0-10)
16. SyncCorrectness (0-10)
17. TypeFidelity (0-5)
18. DeadCode (0-5)

### What the GOAT changes introduced

| Change | Scoring Impact |
|--------|---------------|
| Phase 1.5 Ecosystem Absorb Gate | No scoring impact - this is research, not generated code |
| Phase 3 "Build ALL absorbed features" | **Breadth score should increase** - more commands = higher breadth. Already handled. |
| Phase 3 "Build ALL transcendence features" | **Insight and Workflows scores should increase** - more workflow/insight commands. Already handled. |
| "Agent-native output on every feature" | **AgentNative dimension already checks this.** Scores --json, --select, --dry-run, --stdin, --yes, no-prompts. |

### What's NOT scored that should be

1. **Absorb coverage** - There's no scorecard dimension for "did you match every feature the top competitor has?" The current `Breadth` dimension just counts commands. It doesn't compare against a manifest.

2. **Transcendence depth** - `Insight` checks for file prefixes (health, trends, patterns, etc.) but doesn't verify the insight commands query REAL data from the data layer. A `health.go` that queries an empty table scores the same as one that computes a real composite score.

3. **Compound query verification** - The transcendence features require cross-entity joins (e.g., issues + cycles for velocity). No scorecard dimension checks that compound queries work.

### Recommendation: Scoring is MOSTLY fine. Two small additions.

**Addition 1: Add "Absorb Coverage" check to Tier 2**

If a Phase 1.5 absorb manifest exists (in `docs/plans/*absorb-manifest*`), the scorecard should:
- Count absorbed features in the manifest
- Count implemented commands in the CLI
- Score: implemented / absorbed * 10

This is a Tier 2 (domain correctness) dimension because it checks whether the CLI actually built what the research said to build.

**Addition 2: Verify insight commands query non-empty tables**

Currently `scoreInsight` checks if files like `health.go`, `trends.go` exist. It should ALSO grep those files for actual SQL queries or store method calls, not just file existence.

This is already partially covered by `DataPipelineIntegrity` (checks sync calls domain Upsert). But extending it to insight commands would close the gap.

### What does NOT need to change

- AgentNative (0-10) already covers the agent flags
- OutputModes (0-10) already covers --json, --csv, --select
- ErrorHandling (0-10) already covers typed exits
- The verify command already tests every command at runtime
- The two-tier weighted system (50% infra + 50% domain) is balanced

## The Safety Blanket: Protect Both Sides

The user's insight: don't just check agent principles AFTER building. Use them DURING building AND have an agent REVIEW the work after.

### What Phase 4.9 Does (the pattern to learn from)

Phase 4.9 (`feat-agent-readiness-review-loop-plan.md`) invokes the `compound-engineering:cli-agent-readiness-reviewer` agent which evaluates **7 deep principles** - not just flag presence:

1. Non-interactive automation (no TTY prompts)
2. Structured output (JSON, typed fields)
3. Progressive help (usage examples, not just flag lists)
4. Actionable errors (specific fix suggestions, not generic messages)
5. Safe retries (idempotent operations, --dry-run)
6. Composability (pipe-friendly, exit codes, --select)
7. Bounded responses (--limit, --compact, token-conscious)

Each finding has a severity: **Blocker** (must fix), **Friction** (should fix), **Optimization** (nice to have).

The fix loop: implement all fixes, `go build && go vet`, re-review. Iterate until 0 Blockers + 0 Frictions, max 2 passes. Gracefully skips if compound-engineering plugin not available.

### The Two-Sided Protection Model

```
Phase 3 (BUILD)                    Phase 4.9 (REVIEW)
Agent principles guide what        Agent reviewer checks what
you build and HOW you build it     was actually built

"Every absorbed feature must       "Does search_issues actually
have --json, --dry-run, --select,  output valid JSON? Does
typed exit codes, no prompts"      --dry-run actually skip the
                                   API call? Is the error message
INPUT-SIDE PROTECTION              actionable or generic?"
(checklist during construction)
                                   OUTPUT-SIDE PROTECTION
                                   (agent review with fix loop)
```

### What to Change

**Change 1: Add agent build checklist to Phase 3**

After building each Priority 1 (absorbed) and Priority 2 (transcendence) command, verify:

```markdown
### Agent Build Checklist (per command)

After building each command, verify these 7 principles are met:

1. [ ] **Non-interactive**: No TTY prompts, no `bufio.Scanner(os.Stdin)`, works in CI
2. [ ] **Structured output**: `--json` produces valid JSON, `--select` filters fields
3. [ ] **Progressive help**: `--help` shows realistic examples with domain values
4. [ ] **Actionable errors**: Error messages include the specific flag/arg and correct usage
5. [ ] **Safe retries**: Mutation commands support `--dry-run`, idempotent where possible
6. [ ] **Composability**: Exit codes are typed (0/2/3/4/5/7), output pipes to jq cleanly
7. [ ] **Bounded responses**: `--compact` mode exists, list commands have `--limit`

This checklist maps 1:1 to the 7 principles that Phase 4.9's agent reviewer will check.
If you apply them during build, Phase 4.9 becomes a confirmation, not a catch-all.
```

**Change 2: Phase 3 gets a mini-review gate at Priority boundaries**

After completing Priority 1 (all absorbed features), BEFORE starting Priority 2 (transcendence):

```markdown
### Priority 1 Review Gate

After building all absorbed features, run a quick self-review:
- Pick 3 random commands. Run each with `--json`, `--dry-run`, `--help`.
- Do they all work? If any fail, fix before proceeding.
- This catches systemic issues (e.g., --dry-run not wired) early.
```

This is a lighter version of Phase 4.9's fix loop, applied during build.

**Change 3: No scoring changes needed**

Phase 4.9 already handles the deep review with the agent reviewer. The scorecard's `AgentNative` dimension (0-10) handles the string-matching check. The `verify` command handles runtime testing. Adding another scoring dimension would be redundant - the protection is already three layers deep:

1. Phase 3 build checklist (prevention)
2. Phase 4 shipcheck + verify (runtime testing)
3. Phase 4.9 agent readiness review (deep 7-principle review with fix loop)

## Summary

| Question | Answer | Action Needed? |
|----------|--------|---------------|
| Does Phase 3 enforce agent principles? | Partially - intent is there but no per-command checklist | **Add 7-principle checklist to Phase 3** |
| Does Phase 4.9 already catch agent issues? | Yes - deep 7-principle review with Blocker/Friction severity and fix loop | **Already working. Phase 3 checklist makes it a confirmation not a catch-all.** |
| Does scoring need to change? | No - AgentNative (scorecard) + verify (runtime) + Phase 4.9 (deep review) = three layers | **No changes needed** |
| Is the model right? | Yes - protect BOTH sides. Build with principles (Phase 3), review with agent (Phase 4.9) | **Add the build checklist and mini-review gate** |

## Acceptance Criteria

- [x] Phase 3 in SKILL.md gets the 7-principle agent build checklist (per command)
- [x] Phase 3 gets a mini-review gate between Priority 1 and Priority 2
- [x] Checklist maps 1:1 to Phase 4.9's 7 principles (so Phase 4.9 becomes confirmation)
- [x] No scoring changes (3 layers already sufficient)

## Sources

- `docs/plans/2026-03-27-feat-agent-readiness-review-loop-plan.md` - Phase 4.9 plan (completed)
- `docs/brainstorms/2026-03-27-agent-readiness-review-loop-requirements.md` - 7 principles definition
- `internal/pipeline/scorecard.go` - AgentNative dimension (0-10)
- `skills/printing-press/SKILL.md` - Phase 3 Build The GOAT
- `internal/pipeline/runtime.go` - verify tests every command at runtime
