---
title: "Should We Decompose the Printing Press Skill Into Agents?"
type: refactor
status: active
date: 2026-03-28
---

# Should We Decompose the Printing Press Skill Into Agents?

## The Question

The printing press SKILL.md is 435 lines with 33 sections covering research, ecosystem analysis, code generation, building, and verification. Each phase is a distinct competency. Should we factor some phases into dedicated agents that the main skill dispatches?

## Current Architecture

```
skills/printing-press/SKILL.md (435 lines - orchestrator + everything)
skills/printing-press-score/SKILL.md (already factored out)
skills/printing-press-catalog/SKILL.md (already factored out)
No agents/ directory
```

The skill is the orchestrator AND the worker for every phase. It tells Claude what to research, how to build, what to verify - all in one file.

## The Case FOR Agents

### 1. Fresh context per phase

When the skill is one massive document, the LLM carries all 435 lines plus all prior phase outputs in context. By Phase 3 (Build), the research instructions from Phase 0-1.5 are noise. An agent gets a fresh context window focused on just the build task with the relevant inputs passed in.

### 2. Specialized prompting per competency

Research (Phase 1) requires different prompting than building (Phase 3) than verification (Phase 4). Research needs web search patterns. Building needs code generation patterns and the absorb manifest. Verification needs binary execution and scoring. Each agent can be optimized for its competency.

### 3. Parallelization

Some phases are naturally parallelizable:
- Phase 1 research + Phase 1.5 ecosystem research could run simultaneously
- Phase 4 shipcheck tools (dogfood, verify, scorecard) already run in parallel
- Emboss's re-research + current audit could overlap

With agents, the orchestrator dispatches them in parallel naturally.

### 4. Reusability

A "research-api-landscape" agent could be reused by emboss mode, by the top-50 candidate ranking, and by any future tool that needs to understand an API's ecosystem.

### 5. The precedent works

`printing-press-score` was already factored out as a separate skill. Phase 4.9 uses `compound-engineering:cli-agent-readiness-reviewer` - an EXTERNAL agent. The pattern of the orchestrator calling specialized agents is already in use.

## The Case AGAINST Agents

### 1. SKILL.md is only 435 lines

This is not actually that long. v1 was 1,664 lines. v2 already did the heavy compression. 435 lines is readable in one pass.

### 2. Coordination overhead

Every agent dispatch adds: prompt construction, context passing, result parsing, error handling for agent failures. For a 30-minute process, adding 5 agent dispatches could add 5 minutes of coordination overhead.

### 3. State passing is messy

The skill builds up state across phases: the brief informs the absorb manifest, which informs the build priorities, which inform verification. Passing this between agents means serializing/deserializing state at each boundary. The single-skill approach keeps all state in the conversation context naturally.

### 4. Debugging is harder

When something goes wrong in a single skill, you can trace the conversation. With agents, you're debugging across multiple context windows with only the outputs visible.

### 5. The LLM ignoring instructions is the REAL problem

The reason phases get skipped isn't because the skill is too long - it's because the LLM rationalizes past "mandatory" instructions. Breaking into agents doesn't fix this. The LLM can skip dispatching an agent just as easily as it can skip a phase.

## My Recommendation: Hybrid - Keep the Skill, Add 3 Agents

Don't rewrite the skill. Keep it as the orchestrator. But extract the 3 heaviest phases into agents that the skill dispatches via the Agent tool.

### Agent 1: `printing-press-research` (Phase 1 + 1.5)

**What it does:** Takes an API name, runs ALL the research (brief + ecosystem absorb), returns the brief + absorb manifest as markdown.

**Why agent:** Research is web-search-heavy and benefits from a fresh context window without prior build artifacts. It's the same work whether called from the main skill, from emboss, or from the top-50 ranking.

**Inputs:** API name, spec path (optional), prior research to reuse (optional)
**Outputs:** Brief markdown + absorb manifest markdown
**Size:** ~150 lines of agent instructions

### Agent 2: `printing-press-builder` (Phase 3)

**What it does:** Takes a generated CLI directory + absorb manifest + brief, builds all absorbed features and transcendence commands with the 7-principle agent checklist.

**Why agent:** Building is code-generation-heavy. It benefits from a fresh context focused on the codebase, not on research. This is also the natural delegation point for Codex mode.

**Inputs:** CLI directory, spec path, absorb manifest, brief
**Outputs:** Build log markdown, list of commands built
**Size:** ~100 lines of agent instructions

### Agent 3: `printing-press-verifier` (Phase 4 + 4.9)

**What it does:** Runs the full shipcheck (dogfood + verify + scorecard) and the agent readiness review, with fix loops.

**Why agent:** Verification is binary-execution-heavy. It runs Go commands, parses output, decides what to fix. Benefits from a fresh context without all the research/build noise.

**Inputs:** CLI directory, spec path
**Outputs:** Shipcheck report markdown, verify pass rate, scorecard score
**Size:** ~80 lines of agent instructions

### What stays in the main skill

The orchestrator (~150 lines):
- Mode detection (default, codex, emboss)
- Phase 0 (resolve + reuse - lightweight, stays inline)
- Dispatch Agent 1 (research)
- Present absorb manifest, wait for approval (Phase Gate 1.5)
- Phase 2 (generate - one bash command, stays inline)
- Dispatch Agent 2 (builder)
- Dispatch Agent 3 (verifier)
- Phase 5 (live smoke - lightweight, stays inline)
- Final report

### The math

| Component | Current | Proposed |
|-----------|---------|----------|
| Main skill | 435 lines | ~150 lines |
| Research agent | 0 | ~150 lines |
| Builder agent | 0 | ~100 lines |
| Verifier agent | 0 | ~80 lines |
| **Total** | **435 lines** | **~480 lines** |

Total lines go UP slightly. But each component is focused and has its own context window. The orchestrator is clean and readable.

## When to Do This

**Not now.** The skill works. v2 is only 435 lines. The immediate priority is:
1. Ship CLIs from the library (Cal.com, Notion, Linear, HubSpot)
2. Run emboss on the Cal.com CLI that was already generated
3. Get real feedback from running the press on real APIs

The decomposition makes sense AFTER we've run the press 5-10 times and feel the pain of the monolithic skill. Right now it's theoretical. After 5 runs we'll know which phases actually need isolation.

**Trigger to decompose:** If the skill exceeds 600 lines, or if context window pollution visibly degrades Phase 3 build quality (e.g., the LLM starts confusing research findings with build instructions).

## Acceptance Criteria

- [ ] Decision: decompose now or defer?
- [ ] If now: create agents/ directory, write 3 agent SKILL.md files, refactor main skill
- [ ] If defer: save this plan, revisit after 5 CLI runs

## Sources

- Current skill: `skills/printing-press/SKILL.md` (435 lines, 33 sections)
- Factored example: `skills/printing-press-score/SKILL.md` (separate scoring skill)
- External agent example: Phase 4.9 uses `compound-engineering:cli-agent-readiness-reviewer`
- Context window research: compound-engineering's subagent-driven-development skill uses fresh context per task
