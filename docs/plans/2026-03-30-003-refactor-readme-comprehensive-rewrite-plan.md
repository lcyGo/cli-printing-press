---
title: "refactor: Comprehensive README rewrite + About + Library README"
type: refactor
status: active
date: 2026-03-30
---

# refactor: Comprehensive README rewrite + About + Library README

## Overview

Rewrite the CLI Printing Press README, GitHub About description, and printing-press-library README to reflect the current product after 32 merged PRs in 4 days. Keep the narrative voice and sales energy — this still needs to sell the "crazy hidden insights" concept — but make every section accurate to what the product actually does today.

## Problem Frame

The README was written during rapid early development. Since then, the product has gained: web sniffing (any website becomes a CLI), a publishing pipeline, codex delegation, smart-default output, proxy-envelope client pattern, adaptive rate limiting, URL auto-detection, novel feature auto-suggestions, 18 catalog entries, 2 shipped CLIs in the library, source credits in generated READMEs, discovery provenance, and more. The README still has a stale "What's New in v2" section that references internal iteration history — there is no v2, the product hasn't shipped v1 yet. Everything described is the current pre-v1 product being refined. The README doesn't mention half the current capabilities. The library repo README exists but only has 2 CLIs to showcase. The GitHub About description needs to match.

## Requirements Trace

- R1. README covers all major current capabilities (sniffing, publishing, codex, smart defaults, source credits, catalog of 18 APIs, library of 2 shipped CLIs, URL auto-detection, proxy-envelope, adaptive rate limiting, novel feature suggestions)
- R2. Keep the narrative structure and sales voice — hook, Steinberger story, NOI, absorb & transcend, creativity ladder, etc.
- R3. Remove stale "What's New in v2" section entirely — there is no v2, the product is pre-v1. Fold any still-relevant points into the body as "how it works" descriptions.
- R4. Add web sniffing story prominently — "any website becomes a CLI, no spec needed" is a major differentiator
- R5. Add publishing pipeline and library as proof of production
- R6. Update GitHub About description for cli-printing-press repo
- R7. Update printing-press-library README to reflect current state (2 CLIs, ESPN + Linear)
- R8. All version references, counts, code examples use current values (v0.4.0, 18 catalog entries, 2 library CLIs)
- R9. Update phase table to match current SKILL.md (sniff gate, absorb gate with novel features, discovery archiving)

## Scope Boundaries

- README.md, GitHub About, and library README only — no code changes
- Not restructuring the fundamental narrative arc
- Not writing tutorials or getting-started guides beyond what exists
- Not updating SKILL.md or catalog entries

## Key Technical Decisions

- **Keep narrative structure, update content**: The README story arc works. Don't restructure — update in place, section by section.
- **Delete "What's New in v2" entirely**: There is no v2 — this is all pre-v1 work. The scorecard anti-gaming, feature parity audit, etc. are just how the product works. Fold any still-useful points into the relevant sections as current behavior, not changelog.
- **Sniffing gets prominent placement**: Add to hero section examples and expand in "How It Works" as an alternative entry path. This is the biggest unlock — no OpenAPI spec needed.
- **Publishing + library = credibility**: Mention ESPN and Linear as shipped examples throughout. The library repo link adds proof.
- **Keep the gogcli/discrawl story but trim**: The anecdote is long. Condense to the essential insight — keep Steinberger as the benchmark figure.
- **Update hero code block**: Show the full range of invocations including sniffing, URLs, codex

## Open Questions

### Resolved During Planning

- **Should we add a "Library" section to the main README?** Yes, brief — link to library repo, mention ESPN + Linear as examples, show install command.
- **Keep or cut the "How I Knew This Was Real" section?** Keep but condense. The /last30days research angle is still unique and compelling.

### Deferred to Implementation

- **Exact wording for the sniffing pitch**: Must match existing voice. Short, punchy, "impossible" angle.
- **Whether to restructure the "What Gets Generated" section**: It's dense. Implementer decides if grouping changes help.

## Implementation Units

- [ ] **Unit 1: Rewrite cli-printing-press README.md**

  **Goal:** Comprehensive rewrite of README.md reflecting all current capabilities while preserving narrative voice.

  **Requirements:** R1-R5, R8-R9

  **Dependencies:** None

  **Files:**
  - Modify: `README.md`

  **Approach:**

  Section-by-section rewrite plan:

  **Hero (lines 1-14):** Update code examples to show full range — sniffing (`--har`), URL auto-detection, codex mode, emboss. Add mention of "no spec needed" angle. Keep the agent-first framing.

  **"Get it" (lines 16-28):** Keep as-is, verify install commands are current.

  **"Why These CLIs Win" (lines 30-43):** Add smart-default output (auto-table for humans, auto-JSON when piped). Mention source credits in generated READMEs. Keep agent-native, local-first, dual-interface, verified points.

  **"Every Endpoint. Every Insight. One Command." (lines 44-84):** Keep Steinberger/discrawl story as the anchor. This section is the soul of the README. Trim lightly if needed.

  **"Absorb & Transcend" (lines 52-61):** Add novel feature auto-suggestions (PR #50) — the system now suggests features before the absorb gate. Mention the 18-API catalog as context.

  **"The Non-Obvious Insight" (lines 62-84):** Keep the NOI table and explanation. This is timeless.

  **"How I Knew This Was Real" (lines 86-96):** Condense the gogcli vs Google Workspace story to ~50% of current length. Keep the /last30days hook.

  **"The Creativity Ladder" (lines 98-112):** Keep as-is — the rung concept is clear and stable.

  **"Why Not Just CLIs - CLIs + MCP" (lines 114-133):** Keep. Maybe add note about MCP being auto-discovered by Claude Desktop.

  **"Domain Archetypes" (lines 135-147):** Keep. Stable concept.

  **"How It Works" (lines 149-170):** Major update needed:
  - Update phase table to match current SKILL.md phases
  - Add sniff gate (Phase 1.7) — browser-use capture, HAR import, discovery provenance
  - Add absorb gate with novel feature suggestions
  - Add discovery/ archiving
  - Add publishing pipeline as the final step
  - Update Codex mode description with 3-strike fallback
  - Keep Emboss mode

  **"What Gets Generated" (lines 182-208):** Update:
  - Add smart-default output (human tables / auto-JSON when piped)
  - Add source credits section in generated READMEs
  - Add `.printing-press.json` manifest
  - Add proxy-envelope client pattern mention
  - Add adaptive rate limiting for sniffed APIs
  - Keep all existing agent-native flags

  **"Quality Scoring" (lines 212-258):** Keep structure. Remove all "v2" framing — there is no v2, this is the product. No "v1 did X, now we do Y" comparisons. Just describe how scoring works. Verify dimension lists are current.

  **"Quick Start" (lines 261-298):** Update examples to show:
  - Sniffing: `/printing-press --har ./capture.har --name ESPN`
  - URL: `/printing-press https://postman.com/explore`
  - Publishing: `/printing-press-publish linear`
  - Library install: `go install github.com/mvanhorn/printing-press-library/library/project-management/linear-pp-cli@latest`

  **"Verification Tools" (lines 300-344):** Keep. Verify commands are current.

  **DELETE "What's New in v2" (lines 346-364):** Remove entirely. There is no v2 — this is pre-v1. Any still-useful points (scorecard anti-gaming, feature parity audit, command naming) should already be described in the relevant body sections. Don't reference internal iteration history in a public README.

  **ADD "Library" section (new):** Brief section with:
  - Link to printing-press-library repo
  - ESPN (media-and-entertainment) and Linear (project-management) as shipped examples
  - Install command for library CLIs
  - Link to full catalog (18 APIs)

  **"Development" (lines 368-387):** Keep as-is.

  **"Credits" (lines 389-394):** Update. Verify all contributors are current. Add any new inspirations.

  **Reference materials for implementer:**
  - Current SKILL.md for phase descriptions
  - PRs #43-#74 for feature details
  - `catalog/*.yaml` for full catalog list (18 entries)
  - `~/printing-press/library/` for shipped CLIs
  - `internal/websniff/` for sniffing capabilities
  - `internal/cli/root.go` for URL auto-detection
  - `internal/generator/templates/readme.md.tmpl` for source credits

  **Patterns to follow:**
  - Existing README voice: short sentences, bold claims, specifics over generalities
  - Steinberger as recurring benchmark figure
  - Code blocks showing real commands with real APIs (Discord, Linear, ESPN, Notion)

  **Critical: Purge ALL v1/v2 references.** The current README has v1/v2 in at least 6 places:
  - Line 212: "Quality Scoring (v2 - Three Benchmarks)" → just "Quality Scoring"
  - Line 245: "The v2 scorecard catches that" → rephrase without version reference
  - Line 314: "The v1 scorecard checked..." → rephrase as "Earlier iterations checked..."  or remove the comparison entirely
  - Lines 346-364: entire "What's New in v2" section → delete
  - Line 350: "v1 did excellent competitive research..." → delete with section
  - Line 354: v1/v2 comparison table → delete with section

  There is no v1 or v2. This is the product, pre-launch. Describe what it does, not what it used to do.

  **Verification:**
  - All R1 capabilities mentioned
  - Zero references to "v1" or "v2" anywhere in the README
  - No "What's New" changelog section
  - Phase table matches SKILL.md
  - All code examples work with current command syntax
  - Narrative energy matches existing pitch style

- [ ] **Unit 2: Update GitHub About description**

  **Goal:** Set the cli-printing-press repo's About description to a concise, compelling one-liner.

  **Requirements:** R6

  **Dependencies:** None

  **Files:**
  - GitHub repo settings (via `gh` CLI)

  **Approach:**
  - Use `gh repo edit` to set the description
  - The user provided the description: "Every API has a secret identity. This finds it, absorbs every feature from every competing tool, then builds the GOAT CLI on top — designed for AI agents first, with SQLite sync, offline search, and compound insight commands."
  - Trim to fit GitHub's 350-char limit if needed

  **Verification:**
  - `gh repo view --json description` shows updated description

- [ ] **Unit 3: Update printing-press-library README**

  **Goal:** Update the library repo README to showcase ESPN and Linear, reflect current state.

  **Requirements:** R7

  **Dependencies:** None

  **Files:**
  - Modify: `~/printing-press-library/README.md`

  **Approach:**
  - Read current README
  - Update to highlight ESPN (media-and-entertainment) and Linear (project-management) as shipped examples
  - Add brief descriptions of each CLI with install commands
  - Ensure category list and directory structure reflect current state
  - Keep existing Contributing section and registry.json documentation
  - Update any stale descriptions or counts
  - Link back to cli-printing-press repo as the generator

  **Verification:**
  - README mentions both ESPN and Linear CLIs
  - Install commands are correct
  - Category list matches actual directories
  - Link to cli-printing-press repo is present

## Risks & Dependencies

| Risk | Mitigation |
|------|------------|
| Losing the sales energy in rewrite | Keep paragraphs that work, only update stale content |
| README becomes too long | Integrate new features into existing sections; condense "How I Knew" |
| Inaccurate feature descriptions | Cross-reference SKILL.md, PR bodies, and actual Go code |
| GitHub description too long | 350-char limit — trim if needed |

## Sources & References

- Current README: `README.md` (394 lines)
- Current library README: `~/printing-press-library/README.md`
- Skill definition: `skills/printing-press/SKILL.md`
- Recent PRs: #43-#74 on mvanhorn/cli-printing-press
- Catalog: `catalog/*.yaml` (18 entries: asana, digitalocean, discord, front, github, hubspot, launchdarkly, petstore, pipedrive, plaid, postman-explore, sendgrid, sentry, square, stripe, stytch, telegram, twilio)
- Library: ESPN (media-and-entertainment), Linear (project-management)
- Web sniffing: `internal/websniff/` (~2400 lines)
- URL detection: `internal/cli/root.go`
- Publishing: publish skill + `internal/cli/publish.go`
- Plugin version: v0.4.0
