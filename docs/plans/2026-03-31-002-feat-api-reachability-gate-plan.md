---
title: "feat: API reachability gate before generation"
type: feat
status: active
date: 2026-03-31
---

# feat: API reachability gate before generation

## Overview

Add a mandatory "can we actually reach this API?" check between the absorb gate (Phase 1.5) and generation (Phase 2). One real HTTP call to one real endpoint. If it fails, STOP and tell the user before burning tokens on 10,000 lines of dead code.

## Problem Frame

The Redfin CLI scored 85/100 on the scorecard and 96% on verify. Every API call returns HTTP 403. The CLI is structurally perfect and functionally zero.

The signals were everywhere:
- Sniff gate failed with bot detection (Redfin blocked automated browsers)
- The primary wrapper library (reteps/redfin) had 6+ open issues about 403 errors going back over a year
- Issue #22 was literally titled "Revive broken library: fix 403 errors"

None of this stopped the press from generating, building, polishing, and publishing a CLI that cannot make a single successful API call. The scorecard tests structure, not function. There is no gate that tests "does this API actually respond?"

## Requirements Trace

- R1. Before Phase 2 (generate), test one real API call against the resolved spec's base URL
- R2. If the call fails (403, 401, timeout, DNS error, connection refused), STOP and present the user with options - do not silently proceed
- R3. During Phase 1 research, check GitHub issues on the primary wrapper library for "403", "blocked", "broken", "deprecated" signals
- R4. If the sniff gate failed due to bot detection, treat that as a strong signal that the API itself may block programmatic access - escalate the reachability check to a hard gate, not a warning

## Scope Boundaries

- Only modifying `skills/printing-press/SKILL.md`
- Not changing the scorecard or verify tools (those test code quality, which is fine)
- Not adding a new Go binary command (this is a skill-level check using curl/fetch)
- Not blocking APIs that require auth keys the user hasn't provided yet (that's the existing API key gate)

## Key Technical Decisions

- **Placement: between Phase 1.5 and Phase 2**: After the absorb manifest is approved but before generation starts. This is the last cheap exit point. Once Phase 2 runs, you're committed to tokens.
- **One call, not a suite**: Pick the simplest GET endpoint from the spec (or the base URL with a health/version path). One 200 OK is enough to prove reachability. One 403/timeout is enough to flag the problem.
- **Hard gate for sniff-failed APIs**: If the sniff gate already failed with bot detection/403, the reachability check becomes a hard STOP, not a warning. The evidence is already there.
- **Research-phase issue scanning**: During Phase 1, when fetching competitor repos, also check the Issues tab for 403/blocked/deprecated signals. This catches the Redfin pattern where the wrapper library's own issues were screaming that the API is dead.

## Implementation Units

- [ ] **Unit 1: Add API reachability gate (Phase 1.9) to SKILL.md**

  **Goal:** Before generation, test one real API call. If it fails, present options to the user.

  **Requirements:** R1, R2, R4

  **Files:**
  - Modify: `skills/printing-press/SKILL.md`

  **Approach:**
  - Add a new "Phase 1.9: API Reachability Gate" section between Phase 1.5 (absorb gate) and Phase 2 (generate)
  - The gate runs a single curl/WebFetch against the API's base URL or simplest GET endpoint
  - For APIs with a resolved spec: pick the first GET endpoint with no required params, or fall back to the base URL
  - For sniffed/docs-based APIs: use the base URL directly
  - Success (HTTP 2xx or 3xx): proceed silently
  - Auth error (401/403 without a key): skip silently if the API key gate already noted "no key provided" - the user knows
  - Auth error (403 WITH bot detection signals like HTML error pages, "Are You a Robot", Cloudflare challenge): HARD STOP
  - If the sniff gate previously failed with bot detection: HARD STOP regardless of this call's result
  - On HARD STOP, present via AskUserQuestion:
    > "Warning: `<API>` appears to block programmatic access. [details of what failed]. Building a CLI that can't reach the API is a waste of time. What do you want to do?"
    > 1. **Try anyway** - proceed knowing the CLI may not work against the live API
    > 2. **Pick a different API** - start over with an API that has public access
    > 3. **Done** - stop here
  - Add a MANDATORY checkpoint marker so the LLM can't skip it

  **Verification:**
  - Phase 1.9 section exists between Phase 1.5 and Phase 2
  - Running against an API that returns 403 triggers the HARD STOP with user options
  - Running against a working API proceeds silently

- [ ] **Unit 2: Add issue scanning to Phase 1 research**

  **Goal:** During research, check competitor repo issues for "API is broken/blocked" signals.

  **Requirements:** R3

  **Dependencies:** None (independent of Unit 1)

  **Files:**
  - Modify: `skills/printing-press/SKILL.md` (Phase 1 research checklist)

  **Approach:**
  - In the Phase 1 research checklist, after "Find the top 1-2 competitors", add:
    "Check GitHub issues on the top wrapper library for '403', 'blocked', 'broken', 'deprecated', 'rate limit'. If multiple issues report the API is inaccessible, flag this in the research brief as a reachability risk."
  - This feeds into Phase 1.9 - if research already found 403 signals, the reachability gate should be stricter

  **Verification:**
  - Phase 1 research checklist includes issue scanning instruction
  - Research brief has a place for reachability risk signals

## Sources & References

- Redfin failure report: `docs/retros/2026-03-30-redfin-failure-report.md`
- reteps/redfin issues #7, #15, #19, #20, #21, #22 - all 403 errors
- Sniff gate: `skills/printing-press/SKILL.md` Phase 1.7
- Absorb gate: `skills/printing-press/SKILL.md` Phase 1.5
