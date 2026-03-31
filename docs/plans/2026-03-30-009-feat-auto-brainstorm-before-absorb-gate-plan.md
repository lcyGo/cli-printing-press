---
title: "feat: Auto-brainstorm features before absorb gate"
type: feat
status: active
date: 2026-03-30
---

# feat: Auto-brainstorm features before absorb gate

## Overview

Run the brainstorm questions internally (LLM answers them using research context) and pre-populate the transcendence table with brainstormed features BEFORE presenting the absorb gate. The user sees the full feature set including brainstormed ideas at first glance. "Brainstorm more features" remains as an option for users who want to add their own ideas on top.

## Problem Frame

The absorb gate currently presents the manifest, then offers "Brainstorm more features" as option 2. When the user picks it, the skill asks 3 smart questions:
1. "What workflows do you personally use `<API>` for that aren't covered?"
2. "What's annoying about existing tools that you wish someone would fix?"
3. "If this CLI could do one magical thing, what would make you say 'I need this'?"

These questions are great - but the LLM already has enough research context to answer them itself. In the Redfin run, the auto-suggested features (Step 1.5c.5) scored 7-9/10 and the brainstorm-generated features (portfolio tracker, smart scoring, export engine) scored 8-10/10. The user just wanted to say YES to everything. The brainstorm dialogue was friction that delayed approval without adding insight the LLM didn't already have.

The fix: run the brainstorm questions as an internal LLM exercise during Step 1.5c.5 (which already runs automatically), merge the results into the transcendence table, and present the combined set at the gate. The user still sees "Brainstorm more features" for adding their own ideas - but the default path already includes smart brainstormed features.

## Requirements Trace

- R1. The 3 brainstorm questions are answered by the LLM using research context during Step 1.5c.5
- R2. Brainstormed features are scored with the same 4-dimension scoring system and added to the transcendence table
- R3. The absorb gate presents the COMBINED set (auto-suggested + auto-brainstormed) as the default manifest
- R4. "Brainstorm more features" remains available for users who want to add their own ideas on top
- R5. No new user-facing prompts are added - the brainstorm happens silently as part of the existing auto-suggest step

## Scope Boundaries

- Only modifying `skills/printing-press/SKILL.md`
- Not changing the scoring system or thresholds
- Not removing the "Brainstorm more features" option - just making it additive rather than the first pass
- Not changing Phase 1 research or Phase 2 generation

## Key Technical Decisions

- **Merge into Step 1.5c.5, not a new step**: The auto-suggest step already analyzes research and generates scored features. Adding the brainstorm lens (workflows, pain points, one magical thing) is a natural extension, not a separate step.
- **LLM self-brainstorm, not user Q&A**: The 3 questions become internal prompts the LLM answers using the research brief and absorb manifest as context. This works because the research already covers user workflows, pain points, and competitor gaps.
- **Label brainstormed features distinctly**: In the transcendence table, mark auto-brainstormed features with a different evidence source (e.g., "brainstorm: domain workflow analysis") so the user can distinguish them from data-driven suggestions if they want.

## Implementation Units

- [ ] **Unit 1: Expand Step 1.5c.5 to include self-brainstorm**

  **Goal:** Add a self-brainstorm section to the auto-suggest step that answers the 3 brainstorm questions internally and generates additional features.

  **Requirements:** R1, R2, R3, R5

  **Files:**
  - Modify: `skills/printing-press/SKILL.md`

  **Approach:**
  - In Step 1.5c.5 (Auto-Suggest Novel Features), after the existing 5-category gap analysis, add a 6th category: "Self-brainstorm"
  - The self-brainstorm section instructs the LLM to answer 3 questions using research context:
    1. Based on the research brief's top workflows and user profiles, what workflows does the typical power user of this API do that aren't covered in the absorbed features?
    2. Based on competitor repo issues, community pain points, and ecosystem gaps found in Phase 1/1.5, what are the most annoying limitations that a CLI with SQLite could fix?
    3. Based on the NOI and domain archetype, what single "killer feature" would make a power user install this CLI over any alternative?
  - Each answer that produces a concrete feature gets scored with the same 4-dimension system
  - Features scoring >= 5/10 are added to the transcendence table with evidence citing which research findings informed them
  - Update the gate message to say "auto-suggested and brainstormed" instead of just "auto-suggested"

  **Patterns to follow:**
  - The existing Step 1.5c.5 gap analysis categories and scoring system
  - The existing "Evidence" column requirement for transcendence features

  **Verification:**
  - Step 1.5c.5 now includes a self-brainstorm section with 3 internal questions
  - The absorb gate message references both auto-suggested and brainstormed features
  - "Brainstorm more features" option still exists at the gate for user additions
  - No new user-facing prompts are added before the gate

- [ ] **Unit 2: Update gate option 2 to be additive with structured questions**

  **Goal:** Reword option 2 so it's clear the user is adding their own ideas on top of the auto-brainstorm. Keep the 3 structured questions but reword them to ask for personal knowledge the research couldn't surface.

  **Requirements:** R4

  **Dependencies:** Unit 1

  **Files:**
  - Modify: `skills/printing-press/SKILL.md`

  **Approach:**
  - Change option 2 label from "Brainstorm more features" to "Add your own feature ideas"
  - Change option 2 description from "Interactive dialogue to explore your own feature ideas before building" to "Add features from your personal experience that research couldn't surface"
  - Keep the 3 structured questions but reword them to target user-specific knowledge:
    1. "What workflows do YOU personally use `<API>` for that we might have missed?"
    2. "What frustrates YOU about this API that the research didn't surface?"
    3. "What's YOUR killer feature - something only you'd think of?"
  - The structured questions still generate scored features added to the transcendence table
  - After the brainstorm round, return to the gate with the updated manifest

  **Verification:**
  - Option 2 label and description updated
  - The 3 structured questions still fire when the user picks option 2
  - Questions are reworded to target personal/experiential knowledge, not research-derivable knowledge
  - Features from user brainstorm are still scored and added to transcendence table

## Sources & References

- Current absorb gate: `skills/printing-press/SKILL.md` Phase 1.5, lines 869-1023
- Step 1.5c.5 (Auto-Suggest): lines 950-1002
- Phase Gate 1.5 (brainstorm option): lines 1004-1023
- Redfin run transcript showing brainstorm generated portfolio tracker (10/10), smart scoring (9/10), export engine (8/10) - all features the LLM could have generated from research alone
