---
title: "Phase 0.6: Skill & MCP Landscape Gate"
type: feat
status: active
date: 2026-03-28
origin: docs/plans/2026-03-28-001-research-claude-code-skills-landscape-plan.md
---

# Phase 0.6: Skill & MCP Landscape Gate

## Overview

Before the press generates a CLI, it needs to know what developers can ALREADY do with this API through Claude Code skills, MCP servers, and official plugins. If Cloudflare has a "Code Mode" MCP that covers 2,500 endpoints through 2 tools, building a 200-command CLI wrapper is pointless. If Plaid has zero official presence, the CLI owns everything.

This is a new mandatory phase between Phase 0.5 (Power User Workflows) and Phase 0.7 (Prediction Engine). It answers: "What exists in the AI tooling ecosystem for this API, and where should the CLI focus?"

## Problem Statement

The printing press currently generates CLIs without knowing what AI tooling already exists. We discovered this the hard way:
- PostHog has 27 MCP tools + 12 slash commands + an official plugin - our research initially said "ZERO CLI"
- PagerDuty has 60+ MCP tools - the most comprehensive of any company we analyzed
- Cloudflare's Code Mode covers 100% of their API through just 2 tools
- Linear has 21 MCP tools from an official server

If the CLI duplicates what an MCP already does (live CRUD), it's competing against native AI integration. The CLI's value is the data layer - SQLite, offline search, historical analytics. Phase 0.6 ensures the press knows this before writing a single line of code.

## The Insight

People build skills ON TOP of CLIs. A Stripe MCP skill calls `stripe-cli` under the hood. A GitHub plugin wraps `gh`. The relationship is:

```
CLI (foundation) -> MCP Server (AI interface) -> Skill/Plugin (workflow layer)
```

Knowing what skills/MCPs exist tells you:
1. What the CLI DOESN'T need to duplicate (live CRUD that MCPs handle)
2. What the CLI SHOULD focus on (offline, analytics, data layer - things MCPs can't do)
3. Who has ALREADY thought about the power user workflows (skill authors)

## Proposed Solution

### New Phase: 0.6 - Skill & MCP Landscape

Runs after Phase 0.5 (Power User Workflows) and before Phase 0.7 (Prediction Engine).

### Step 0.6a: Search for Official Plugins

```
WebSearch: "<API name>" Claude Code plugin site:github.com
WebSearch: "<API name>" plugin site:claude.com OR site:anthropic.com
WebFetch: https://github.com/anthropics/claude-plugins-official/tree/main/external_plugins
```

Check if the API has an official Anthropic marketplace listing. Record: plugin name, install command, what it bundles.

### Step 0.6b: Search for MCP Servers

```
WebSearch: "<API name>" MCP server site:github.com
WebSearch: "<API name>" model context protocol
WebSearch: "<API name>" mcp tools site:lobehub.com OR site:mcpmarket.com OR site:fastmcp.me
WebFetch: https://mcp.<api-domain>.com (check if official remote MCP exists)
```

For each MCP found, catalog:
- Name and source (official vs community)
- Number of tools
- What each tool does (1-line description)
- Auth method (OAuth, API key, local)
- Coverage: what % of the API surface does it cover?

### Step 0.6c: Search for Community Skills

```
WebSearch: "<API name>" Claude Code skill site:github.com
WebSearch: "<API name>" skill SKILL.md site:github.com
WebSearch: "<API name>" slash command Claude
```

### Step 0.6d: Gap Analysis

Produce a table:

| What Already Exists | Coverage | What's Missing (CLI Opportunity) |
|---|---|---|
| Official MCP with 25 tools covering CRUD | 15% of API surface | 85% uncovered + zero data layer |
| Community skill with workflow patterns | Workflow guidance only | No execution, no persistence |

Then answer these questions:
1. **What can an MCP do that the CLI doesn't need to?** (live CRUD, OAuth flows, tool discovery)
2. **What can ONLY a CLI do?** (SQLite sync, offline search, raw SQL, batch operations, typed exit codes, pipe to jq)
3. **What should the CLI's workflow commands focus on?** (the gaps in MCP coverage)

### Step 0.6e: Write the Artifact

Write to `docs/plans/<today>-feat-<api>-cli-skill-landscape.md`:

```markdown
## Skill & MCP Landscape: <API>

### Official Presence
- Plugin: [yes/no] - [name, install]
- MCP Server: [yes/no] - [name, tool count]
- Skills: [count] community skills found

### Tool Inventory
| Tool | Source | Category | What It Does |
|------|--------|----------|-------------|

### Coverage Analysis
- MCP covers: [X]% of API surface
- MCP handles: [list of what's covered]
- MCP misses: [list of what's not covered]

### CLI Differentiation
The CLI should focus on:
1. [specific gap 1]
2. [specific gap 2]
3. [specific gap 3]

### Impact on Phase 0.7
[How this changes the data layer design]
[Which entities matter MORE because MCPs don't persist them]
[Which entities matter LESS because MCPs handle them live]
```

### Phase Gate 0.6

**STOP.** Verify:
1. Searched for official plugin, MCP server, and community skills
2. Tool inventory table is populated
3. Coverage % is estimated
4. CLI differentiation is articulated (at least 3 specific gaps)
5. Phase 0.7 impact notes written

Tell the user: "Phase 0.6 complete: [API] has [N] MCP tools covering [X]% of the API. CLI should focus on: [top 3 gaps]. Proceeding to data layer prediction."

## Implementation

### In SKILL.md

Add Phase 0.6 between Phase 0.5 and Phase 0.7. The phase diagram becomes:

```
PHASE 0 -> 0.5 -> 0.6 -> 0.7 -> 0.8 -> 0.9 -> 1 -> 2 -> 3 -> 4 -> 4.5 -> 4.6 -> 4.8 -> 5
```

### How It Feeds Into Later Phases

| Phase | How 0.6 Changes It |
|-------|-------------------|
| **0.7 (Data Layer)** | Focus SQLite tables on entities MCPs DON'T persist. If MCP handles live issue CRUD, the CLI's value is syncing issues for offline search + analytics. |
| **0.8 (Product Thesis)** | The "what's different" question is now informed by actual ecosystem research, not guesswork. |
| **Phase 4 (GOAT Build)** | Workflow commands should NOT duplicate MCP tools. If the MCP has `create_issue`, the CLI doesn't need `issue create` - it needs `stale --days 30`. |
| **README** | Can say "Works alongside the [API] MCP server" instead of competing with it. |

### Time Budget

5-10 minutes. Mostly WebSearch + WebFetch. The research you did in `2026-03-28-001-research-claude-code-skills-landscape-plan.md` took more time because it covered 11 APIs. Per-API it's fast.

## Acceptance Criteria

- [ ] SKILL.md has Phase 0.6 between Phase 0.5 and Phase 0.7
- [ ] Phase 0.6 searches for: official plugin, MCP server, community skills
- [ ] Phase 0.6 produces a tool inventory table
- [ ] Phase 0.6 produces a coverage % estimate
- [ ] Phase 0.6 produces 3+ CLI differentiation points
- [ ] Phase 0.6 writes an artifact to docs/plans/
- [ ] Phase Gate 0.6 is a mandatory stop gate
- [ ] Phase 0.7 and 0.8 reference Phase 0.6 findings

## Sources

- Origin: `docs/plans/2026-03-28-001-research-claude-code-skills-landscape-plan.md` - the example research that proved this phase's value
- Key finding: Cloudflare Code Mode covers 100% of API through 2 MCP tools - a CLI would have been wasted without knowing this
- Key finding: PagerDuty has 60+ MCP tools but zero data layer - the CLI angle is purely SQLite analytics
- Key finding: Plaid has zero official presence - the CLI owns everything
