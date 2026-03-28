---
title: "Phase 0.6: Absorb & Transcend - Build the GOAT by Stealing Every Best Idea"
type: feat
status: active
date: 2026-03-28
origin: docs/plans/2026-03-28-001-research-claude-code-skills-landscape-plan.md
---

# Phase 0.6: Absorb & Transcend

## The Philosophy

The CLI Printing Press doesn't build "an alternative." It builds THE GOAT. That means:

1. **Absorb everything** - Every feature from every MCP tool, every skill command, every competing CLI, every community script gets cataloged and built INTO our CLI. If the Linear MCP has `search_issues` with full-text filters, our CLI has that AND offline search AND SQL queries AND it works without internet.

2. **Transcend with the NOI** - Once we've absorbed every known feature, we find the compounding use cases nobody thought of. The Non-Obvious Insight isn't just a marketing angle - it's the LAYER that sits on top of comprehensive feature coverage.

The GOAT CLI = everything everyone else does + everything nobody else thought of.

## Problem Statement

The previous version of this plan had it backwards. It said "find gaps, build around MCPs." Wrong. The right approach:

- If Stripe's MCP has `create_payment_intent` - our CLI has that too, PLUS it works offline, PLUS it pipes to jq, PLUS it has typed exit codes
- If PagerDuty's MCP has 60 tools covering the full incident lifecycle - our CLI has ALL 60 of those capabilities, PLUS SQLite persistence, PLUS historical MTTR analytics, PLUS on-call burden tracking
- If Cloudflare's Code Mode covers 2,500 endpoints - our CLI covers them too, PLUS you can sync config changes to SQLite and diff them over time

The MCP/skill landscape isn't a constraint. It's a requirements list. Every tool someone else built is a feature we need to match AND exceed.

## How Phase 0.6 Works

### Step 0.6a: Catalog EVERY Feature That Exists

Search for ALL Claude Code skills, MCP servers, plugins, competing CLIs, community scripts, and automation tools for this API.

**Search targets:**
1. Official Claude Code plugin (Anthropic marketplace)
2. Official MCP server (remote or local)
3. Community MCP servers (lobehub, mcpmarket, fastmcp)
4. Community Claude Code skills
5. Competing CLIs (npm, GitHub, Homebrew)
6. Automation scripts and tools (GitHub topics, awesome lists)
7. Popular wrapper libraries (what functions do the top SDKs expose?)

**For each tool found, extract the COMPLETE feature list:**

```
| Source | Tool/Command | What It Does | We Must Match | We Must Beat |
|--------|-------------|-------------|---------------|-------------|
| Linear MCP | search_issues | Full-text search with team/assignee/status/priority filters | Yes - same filters | + Offline search via FTS5, + regex, + SQL WHERE clauses |
| Linear MCP | create_issue | Create with title, desc, team, assignee, priority, labels | Yes - same fields | + --stdin for batch creation, + --dry-run preview |
| jira-cli (5.4k stars) | sprint view | Interactive sprint board | Yes - sprint visibility | + SQLite-backed velocity tracking over time |
```

**This is the requirements table.** Every row becomes a feature the CLI must have.

### Step 0.6b: Build the Comprehensive Feature Manifest

Merge all features from all sources into one master list, deduplicated:

```
| Feature | Sources That Have It | Our Implementation | Added Value |
|---------|---------------------|-------------------|-------------|
| Search issues by text | MCP, jira-cli, gh, github-to-sqlite | FTS5 offline search | Works without internet, regex support, SQL composable |
| Create issue | MCP, official CLI | --stdin batch, --dry-run, --json output | Agent-native, scriptable, idempotent |
| Sprint velocity | None (gap!) | `velocity --sprint current` from SQLite | Historical trends, team comparison |
```

**Column 4 (Added Value) is critical.** For EVERY feature, even the ones we're matching, we add printing-press value:
- `--json` output (agent-native)
- `--dry-run` (safe testing)
- `--stdin` (batch operations)
- `--select` (field filtering)
- Typed exit codes (agent self-correction)
- SQLite persistence (offline)
- FTS5 (instant search)

### Step 0.6c: Identify the Transcendence Layer

Now that we have EVERY feature anyone has ever built, ask: "What compound use cases become possible ONLY when you have ALL of these together?"

This is where the Non-Obvious Insight lives. Examples:

**Linear:** If you have issues + cycles + team members + comments all in SQLite...
- `bottleneck` - which issues block the most other issues? (requires relationship graph in local DB)
- `similar` - find duplicate issues across teams (requires FTS5 across all issue text)
- `velocity` - is the team getting faster or slower? (requires historical cycle data in SQLite)
- `silent` - which team members stopped contributing? (requires time-series activity data)

**Stripe:** If you have charges + subscriptions + disputes + payouts all in SQLite...
- `churn-risk` - subscriptions with failed charges in the last 30 days (requires local join)
- `revenue-trend` - MRR over time (requires historical subscription snapshots)
- `dispute-pattern` - which products generate the most disputes? (requires cross-entity analysis)

**These compound commands are ONLY possible because the CLI has a local data layer.** No MCP can do them because MCPs are stateless. This is the moat.

### Step 0.6d: Write the Feature Manifest Artifact

Write to `docs/plans/<today>-feat-<api>-cli-feature-manifest.md`:

```markdown
## Feature Manifest: <API> CLI

### Absorbed Features (match or beat everything that exists)
| # | Feature | Best Current Source | Our Implementation | Added Value |
|---|---------|--------------------|--------------------|-------------|
| 1 | ... | ... | ... | ... |

### Transcendence Features (only possible with our data layer)
| # | Feature | Command | Why Only We Can Do This |
|---|---------|---------|------------------------|
| 1 | ... | ... | Requires local join across X and Y tables |

### Feature Count
- Features absorbed from existing tools: [N]
- Features added (transcendence layer): [M]
- Total CLI features: [N + M]
- This is [X]% more than the best existing tool
```

### Phase Gate 0.6

**STOP.** Verify:
1. Every feature from every MCP/skill/CLI/tool is cataloged
2. Every feature has an "Added Value" column showing how we beat the source
3. At least 5 transcendence features identified (compound use cases only we can do)
4. Feature manifest artifact written
5. Phase 0.7 data layer will be informed by which entities need SQLite persistence for transcendence

Tell the user: "Phase 0.6 complete: Absorbed [N] features from [X] tools. Added [M] transcendence features. Total: [N+M] features - [Z]% more than the best existing tool. Proceeding to data layer prediction."

## Audit of Current SKILL.md v2.0.0

The upstream merge brought a v2 rewrite (362 lines, 5 phases). Here's where the GOAT philosophy is missing:

### Phase 1 (Research Brief) - MISSING Absorb

Current brief asks:
- "What are the top 3-5 power-user workflows?" (too few - absorb ALL workflows from ALL tools)
- "What are the top table-stakes competitor features?" (passive framing - should be "steal every feature")
- "Find the top 1-2 competitors" (should be "find EVERY tool, MCP, skill, script")

**Fix:** Add an "Absorb" section to the brief that catalogs every feature from every tool and marks each as "must match" or "must beat."

### Phase 3 (Build) - MISSING Comprehensive Coverage

Current Phase 3 says "Build only the things most likely to change ship-readiness." This is a gap-filling mentality, not a GOAT mentality.

**Fix:** Phase 3 Priority 2 should be "ALL absorbed features from the brief's feature manifest, not just top 3-5." The transcendence layer (NOI commands) sits on top of comprehensive coverage.

### Rules - MISSING Anti-Gap-Thinking Rules

No rule says "build everything others have." The rules optimize for speed, which is good, but speed without comprehensiveness produces thin CLIs.

**Fix:** Add rules about absorbing before transcending.

### What v2 Gets Right

- Lean loop (brief -> generate -> build -> shipcheck) is faster than v1's 13 phases
- "Optimize for time-to-ship" is correct AS LONG AS comprehensiveness isn't sacrificed
- Single brief instead of 7 separate docs reduces busywork
- Emboss mode for second passes is preserved

## Implementation: What to Change in SKILL.md v2.0.0

### Change 1: Add Phase 1.5 as a separate gate in SKILL.md

Insert after Phase 1 (Research Brief), before Phase 2 (Generate):

```markdown
## Phase 1.5: Ecosystem Absorb Gate

THIS IS A MANDATORY STOP GATE. Do not generate until this is complete.

Research EVERY Claude Code plugin, MCP server, community skill, competing CLI,
and automation tool for this API. Catalog every feature. Build the absorb manifest.

### Step 1.5a: Search for every tool
1. WebSearch: "<API name>" Claude Code plugin site:github.com
2. WebSearch: "<API name>" MCP server model context protocol
3. WebSearch: "<API name>" Claude skill SKILL.md site:github.com
4. WebSearch: "<API name>" CLI tool site:github.com (competing CLIs)
5. WebSearch: "<API name>" automation script site:github.com
6. Check: Anthropic marketplace (claude-plugins-official repo)
7. Check: npm for @<api>/cli or <api>-cli packages
8. Check: MCP directories (lobehub, mcpmarket, fastmcp)

### Step 1.5b: Catalog every feature
For EACH tool found, list every feature/tool/command it provides.
Build the absorb manifest:

| Feature | Best Source | Our Implementation | Added Value |
|---------|-----------|-------------------|-------------|
| Search issues | Linear MCP (search_issues) | FTS5 offline search | Works without internet, regex, SQL |
| Create issue | Linear MCP (create_issue) | --stdin batch, --dry-run | Agent-native, scriptable |
| ... | ... | ... | ... |

Every row = a feature we MUST have. No exceptions.

### Step 1.5c: Identify transcendence features
What compound use cases become possible ONLY when all absorbed features
live in SQLite together? These are the NOI commands.

| Transcendence Feature | Command | Why Only We Can Do This |
|----------------------|---------|------------------------|
| ... | ... | Requires local join across X and Y |

Minimum 5 transcendence features.

### Phase Gate 1.5
STOP. Present the absorb manifest to the user. Tell them:
"Found [N] features across [X] tools. Our CLI will have all [N] plus [M]
transcendence features. Total: [N+M]. Approve to proceed to generation."

WAIT for approval before generating.
```

### Change 2: Update Phase 3 Build Priorities

In Phase 3, change the priority list. Currently it says "top 3-5 power-user workflows."
Replace with:

```markdown
Priority 1:
- data layer foundations for primary entities (unchanged)

Priority 2:
- ALL absorbed features from the Phase 1.5 manifest
- Every feature from every competing tool, matched and beaten with agent-native output
- This is NOT "top 3-5" - it's the FULL manifest

Priority 3:
- ALL transcendence features from Phase 1.5
- The NOI commands that only work because we have everything in SQLite
- This is where "12 commands beats 300" lives

Priority 4:
- skipped complex request bodies (unchanged)
- tests (unchanged)
```

### Change 3: Add anti-shortcut rules

Add to "What not to do":
```
- skip features because "the MCP already handles that" (absorb everything, beat it with offline + agent-native)
- build only "top 3-5 workflows" when the absorb manifest has 15+ (build them ALL)
- generate before Phase 1.5 Ecosystem Absorb Gate is complete
- call a CLI "GOAT" without matching every feature the best competitor has
```

## How This Changes the Printing Press

### The Two-Layer Model

```
Layer 1: ABSORB
  Every feature from every MCP, skill, CLI, script, SDK
  Matched AND beaten with agent-native output modes
  The "table stakes" - if anyone else has it, we have it better

Layer 2: TRANSCEND
  Compound use cases that require local data + cross-entity analysis
  The Non-Obvious Insight commands
  Only possible because Layer 1 put everything in SQLite
  This is why "12 commands beats 300" - the 12 are transcendence commands
```

### Where This Sits in the v2 Phase Sequence

```
Phase 0   - Resolve & Reuse
Phase 1   - Research Brief (API identity, competitors, product thesis)
Phase 1.5 - Ecosystem Absorb Gate (NEW - dedicated CC plugin/MCP/skill research -> absorb manifest)
Phase 2   - Generate
Phase 3   - Build (uses absorb manifest as the FULL feature list, not just "top 3-5")
Phase 4   - Shipcheck
Phase 5   - Live Smoke
```

Phase 1.5 is a SEPARATE GATE between the brief and generation. It:
1. Takes the brief's competitor findings as input
2. Deep-dives into every Claude Code plugin, MCP server, community skill, competing CLI
3. Catalogs every feature into the Absorb manifest
4. Identifies transcendence features on top
5. Presents the manifest to the user for approval before generating

Phase 0.6 directly feeds:
- **0.7:** The feature manifest tells you which entities need SQLite tables (anything in the transcendence layer requires local persistence)
- **0.8:** The product thesis becomes "everything X has, plus Y that only we can do"
- **Phase 4:** The feature manifest IS the build list for Phase 4

### Anti-Shortcut Rules to Add

```
- "The MCP already handles that, we don't need it" (WRONG. We need everything the MCP has
  AND more. The GOAT CLI doesn't skip features because someone else implemented them.
  It absorbs them and adds value: offline, --json, --dry-run, typed exit codes, SQLite.)
- "60 tools is too many to match" (PagerDuty's MCP has 60 tools. Our CLI should have
  60+ commands covering the same surface, PLUS the transcendence layer. Comprehensive
  coverage is the foundation that makes the NOI possible.)
- "We should focus on the gaps" (Gaps are where we TRANSCEND. But we can only transcend
  if we first ABSORB. You can't build 'velocity tracking' without first having the
  issue/cycle data that the MCP also has. Absorb first, transcend second.)
```

## Acceptance Criteria

- [ ] SKILL.md v2 has Phase 1.5 "Ecosystem Absorb Gate" between Phase 1 and Phase 2
- [ ] Phase 1.5 searches: CC plugins, MCP servers, community skills, competing CLIs, npm packages, MCP directories
- [ ] Phase 1.5 produces an absorb manifest table (feature | source | our impl | added value)
- [ ] At least 5 transcendence features identified per API
- [ ] Phase Gate 1.5 presents manifest to user and WAITS for approval before generating
- [ ] Phase 3 build priorities updated to use FULL manifest, not "top 3-5"
- [ ] 4 new anti-shortcut rules added
- [ ] Phase 3 data layer informed by transcendence requirements

## Sources

- Origin: `docs/plans/2026-03-28-001-research-claude-code-skills-landscape-plan.md`
- Linear MCP: 21 tools - all must be absorbed
- Stripe MCP: 25 tools - all must be absorbed
- PagerDuty MCP: 60+ tools - all must be absorbed
- PostHog: 27 tools + 12 slash commands - all must be absorbed
- Cloudflare Code Mode: 2,500 endpoints via 2 tools - coverage must be matched
- The Steinberger standard: discrawl absorbed Discord's data model AND transcended with sync+search+sql
