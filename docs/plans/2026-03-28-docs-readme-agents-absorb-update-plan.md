---
title: "Update README + GitHub About with Agents-First and Absorb & Transcend"
type: docs
status: active
date: 2026-03-28
---

# Update README + GitHub About

## What's Changed Since the README Was Last Updated

1. **Phase 1.5: Ecosystem Absorb Gate** - The press now catalogs every feature from every MCP, skill, and competing CLI, then builds ALL of them into the generated CLI before transcending with NOI commands. The README still says "top 3-5 workflows" in places.

2. **Agents are the primary power user** - The README mentions "agent-first flags" but doesn't lead with the thesis that AI agents (Claude Code, Codex, Gemini CLI, Cursor) are now the primary consumers of CLIs. Humans use CLIs; agents LIVE in them. Every design decision flows from this.

3. **Absorb & Transcend philosophy** - The README talks about "depth beats breadth" and the NOI, but doesn't explain the two-layer model: absorb everything competitors have, then transcend with compound use cases only possible with local data.

4. **Phase diagram is outdated** - Missing Phase 1.5 (Ecosystem Absorb Gate).

5. **Emboss mode** - Built but not in README.

6. **Verify command** - Built but not in README verification tools section.

7. **GitHub About description** - Currently: "Every API has a secret identity..." Good but doesn't mention agents or the absorb model.

## Changes to Make

### C1: Update the opening pitch (lines 1-11)

Current: "Just making a CLI is not hard. Making a CLI that understands the power user is extremely hard."

Add after: The power user in 2026 is an AI agent. Claude Code, Codex, Gemini CLI, Cursor - they call CLIs thousands of times a day. Every printing press CLI is designed for agents first: --json by default when piped, typed exit codes for self-correction, --compact for token efficiency, --dry-run for safe exploration. Humans get the same great experience, but agents are the primary design target.

### C2: Add "Absorb & Transcend" section (after NOI section)

New section explaining the two-layer model:
- Layer 1: Absorb every feature from every MCP, skill, competing CLI. Match and beat them all.
- Layer 2: Transcend with compound use cases that only work with local SQLite data.
- The GOAT CLI = everything everyone else does + everything nobody else thought of.

### C3: Update Phase diagram (line 119-125)

Add Phase 1.5 between Phase 1 and Phase 2:
```
Phase 1.5   Ecosystem Absorb Gate    (5-10 min)   Catalog every tool, build absorb manifest, get approval
```

### C4: Add Emboss mode to README

After the Codex Mode section, add Emboss:
```bash
/printing-press emboss ./library/notion-cli   # Second pass: improve an existing CLI
```
Brief explanation of the 6-step cycle and delta reporting.

### C5: Add Verify command to Verification Tools

```bash
# Runtime verification: tests every command against real API or mock server
printing-press verify --dir ./my-cli --spec ./openapi.json --api-key $TOKEN
```

### C6: Update GitHub About description

Current: "Every API has a secret identity - Stripe isn't payments, it's a business health monitor. This finds the Non-Obvious Insight and generates a Go CLI with SQLite sync, offline search, and the 12 commands that matter."

New: "Every API has a secret identity. This finds it, absorbs every feature from every competing tool, then builds the GOAT CLI on top - Go binary + MCP server, designed for AI agents first, with SQLite sync and offline search."

### C7: Fix "What Gets Generated" agent framing

The "Agent-first flags" bullet exists but is buried. Move it up and expand: explain WHY agents need these specific features (typed exit codes for self-correction, --compact for token budget, auto-JSON for pipe chains, --dry-run for safe exploration).

## Acceptance Criteria

- [ ] Opening pitch mentions agents as primary power user
- [ ] "Absorb & Transcend" section explains two-layer model
- [ ] Phase diagram includes Phase 1.5 Ecosystem Absorb Gate
- [ ] Emboss mode documented
- [ ] Verify command documented in verification tools
- [ ] GitHub About description updated
- [ ] "What Gets Generated" leads with agent design rationale
- [ ] No stale references to "top 3-5 workflows" (should be "full absorb manifest")

## Files

- `README.md`
- GitHub repo description (via `gh api`)
