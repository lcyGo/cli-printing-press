---
title: "fix: Sniff gate (Phase 1.7) skipped during skill execution"
type: fix
status: active
date: 2026-03-30
---

# fix: Sniff gate (Phase 1.7) skipped during skill execution

## Overview

Add a mandatory checkpoint after Phase 1 research that forces the LLM to evaluate the sniff gate before proceeding to the absorb gate. The sniff gate is being skipped entirely for APIs with no OpenAPI spec (like Redfin), which is exactly when it should fire.

## Problem Frame

When running `/printing-press Redfin Codex`, the skill executor went Phase 1 (research) -> Phase 1.5 (absorb gate) -> absorb approval, completely skipping Phase 1.7 (sniff gate) and Phase 1.8 (crowd sniff). Redfin has no OpenAPI spec and no catalog entry - the sniff gate decision matrix says "MUST offer sniff OR --docs" for this case. But the LLM never evaluated it.

Root cause: Unlike codex detection (which we fixed by merging into one bash block), the sniff gate can't be a bash block - it requires LLM judgment to evaluate a decision matrix. The LLM skips it because after writing the Phase 1 brief, it's eager to do the absorb searches and jumps ahead. The sniff gate section exists in the file but gets treated as optional.

Contributing factor: Phase ordering in the file is 1 -> 1.7 -> 1.8 -> 1.5, but the LLM doesn't follow file order - it follows what feels like the natural next step after research (absorb).

## Requirements Trace

- R1. The sniff gate decision matrix must be evaluated after Phase 1 research for every run that doesn't already have a spec
- R2. The LLM must not be able to skip from Phase 1 to Phase 1.5 without at least checking whether sniff applies

## Scope Boundaries

- Only modifying `skills/printing-press/SKILL.md`
- Not changing sniff gate behavior, just ensuring it gets evaluated
- Not reordering all phases (that's a bigger refactor)

## Key Technical Decisions

- **Add a mandatory checkpoint at the end of Phase 1**: After the brief is written, add a bolded instruction block that says "STOP. Before proceeding to Phase 1.5, you MUST evaluate Phase 1.7 (Sniff Gate)." This is the same pattern as "THIS IS A MANDATORY STOP GATE" that Phase 1.5 already uses successfully.
- **Add a pre-check to Phase 1.5**: At the top of Phase 1.5, add a guard: "If no spec source has been resolved (no --spec, no --har, no catalog spec, no sniff), STOP. You skipped Phase 1.7. Go back and evaluate the sniff gate."

## Implementation Units

- [ ] **Unit 1: Add mandatory sniff gate checkpoint after Phase 1 brief**

  **Goal:** Force the LLM to evaluate the sniff gate before moving on.

  **Requirements:** R1, R2

  **Files:**
  - Modify: `skills/printing-press/SKILL.md`

  **Approach:**
  - After the Phase 1 brief template (around line 420, after the closing code fence), add a bolded checkpoint block:
    ```
    **MANDATORY: Before proceeding to Phase 1.5 (Absorb Gate), evaluate Phase 1.7 (Sniff Gate) and Phase 1.8 (Crowd Sniff Gate).** If no spec source has been resolved yet (no --spec, no --har, no catalog spec URL), you MUST evaluate the sniff gate decision matrix. Do not skip to Phase 1.5.
    ```
  - At the top of Phase 1.5 (line 867), add a guard check:
    ```
    **Pre-check:** If no spec or HAR file has been resolved by this point and Phase 1.7 was not evaluated, STOP. Go back and run the sniff gate. The absorb manifest depends on knowing the API surface, which requires a spec.
    ```

  **Patterns to follow:**
  - Phase 1.5 already has "THIS IS A MANDATORY STOP GATE" - same energy
  - The codex detection fix used structural placement to prevent skipping

  **Verification:**
  - The sniff gate checkpoint text appears between Phase 1 brief and Phase 1.5
  - Phase 1.5 has a pre-check guard referencing spec resolution
  - No other phase behavior is changed
