---
title: "fix: Redact private workspace/org data from public artifacts"
type: fix
status: active
date: 2026-04-08
---

# fix: Redact private workspace/org data from public artifacts

## Overview

During the Linear CLI publish, the PR description included the user's private workspace name ("Esper Labs") in the live test results. The README and generated code were clean, but the skill's Phase 5 (dogfood testing) and Phase 6 (publish) create public-facing text that can leak private information. The printing-press and publish skills need explicit redaction rules.

## Problem Frame

The printing-press skill instructs Claude to run live dogfood tests and report results. Those results naturally include workspace-specific data (org names, team names, user emails, member names). When this data flows into PR descriptions or manuscripts that get published to a public repo, it becomes a privacy leak.

Three surfaces can leak:
1. PR descriptions created by the publish skill (confirmed leak)
2. Manuscripts archived and published alongside the CLI (.manuscripts/ in the library repo)
3. Acceptance reports and shipcheck proofs written during the session

## Requirements Trace

- R1. PR descriptions must never contain workspace names, org names, user emails, or team member names
- R2. Manuscripts published to the library repo must be redacted
- R3. The main printing-press skill must instruct Claude to redact before archiving
- R4. The publish skill must instruct Claude to redact before creating/updating PRs

## Scope Boundaries

- Does not add automated scanning (that's a separate tooling improvement)
- Does not change generated CLI code (the README template is already clean)
- Skill-level instruction changes only

## Key Technical Decisions

- **Instruction-level fix, not code-level**: The leak happens in Claude-authored text (PR descriptions, test reports), not in generated code. The fix is explicit redaction instructions in the skills.
- **Redact at two points**: (1) when writing acceptance/shipcheck proofs, (2) when creating PR descriptions. Belt and suspenders.

## Implementation Units

- [ ] **Unit 1: Add redaction rules to printing-press skill Phase 5**

**Goal:** Prevent private workspace data from appearing in acceptance reports and manuscripts

**Requirements:** R1, R2, R3

**Dependencies:** None

**Files:**
- Modify: `skills/printing-press/SKILL.md` (Phase 5 dogfood section)

**Approach:**
- Add a cardinal rule to the Phase 5 section: "When reporting live test results, NEVER include workspace names, organization names, team member names, or user email addresses. Use generic descriptions instead: 'the workspace' not 'Esper Labs', 'team members' not 'patrick@esperlabs.ai', '5 overloaded members' not their names."
- Apply to acceptance report format, shipcheck proof, and any stderr output captured in manuscripts
- Add to the existing Secret & PII Protection section as a new category alongside API keys

**Test expectation:** none - skill instruction change, verified by human review of next generation

**Verification:**
- Next CLI generation's acceptance report contains no workspace-specific names
- Manuscripts archived to library repo contain no private data

- [ ] **Unit 2: Add redaction rules to publish skill PR descriptions**

**Goal:** Prevent private workspace data from appearing in public PR descriptions

**Requirements:** R1, R4

**Dependencies:** None (parallel with Unit 1)

**Files:**
- Modify: `skills/printing-press-publish/SKILL.md` (Step 8 PR description section)

**Approach:**
- Add instruction before the PR description template: "Before constructing the PR body, scan any live test results or acceptance data for workspace names, org names, team member names, and email addresses. Replace with generic descriptions. The PR is public."
- Add to the PR description template a reminder comment: `<!-- REDACT: Remove all workspace-specific names, emails, and team member identities before publishing -->`

**Test expectation:** none - skill instruction change, verified by human review of next publish

**Verification:**
- Next publish PR description contains no workspace-specific data

## Risks & Dependencies

| Risk | Mitigation |
|------|------------|
| Claude may still leak data despite instructions | The secret-protection.md reference already handles API keys; extending it to cover PII makes the pattern consistent |
| Manuscripts may contain data from before the fix | Only affects future generations; existing manuscripts in the library repo should be audited manually |

## Sources & References

- Evidence: PR #23 in mvanhorn/printing-press-library contained "Esper Labs workspace" in live test results
- Related: `skills/printing-press/references/secret-protection.md` (existing secret scanning rules)
