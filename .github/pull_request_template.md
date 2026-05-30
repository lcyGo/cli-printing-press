<!--
Maintainer-owned PRs may use a shorter body. Community PRs must keep these sections.
Write for reviewers: explain why this change is right, not every line that changed.
-->

## Intent

<!-- What problem does this solve? Link the issue if one exists. -->

Issue:

<!-- Use Closes/Fixes only when this PR fully resolves the issue. Use Refs or N/A otherwise. -->

## Approach

<!-- Explain the fix shape or design choice. Avoid a file-by-file change list. -->

## Scope

Primary area:

Why this belongs in this repo:

<!-- Printed-CLI-only fixes belong in the generated CLI or public library repo. If the symptom came from a printed CLI, explain the general Printing Press behavior this changes. -->

## Catalog Justification

<!-- Required when this PR adds or edits catalog/*.yaml or catalog/specs/**. Otherwise write "N/A". -->

Embedded catalog fit:

Distinct blueprint pattern:

Closest existing entries checked:

Source provenance:

Auth and tenant assumptions:

Safe default surface:

Generation path:

Stale-body check:

## CLI Shape

<!--
Required when this PR adds or edits a generatable catalog entry (one with spec_url/spec_format, or a spec file under catalog/specs/). Wrapper-only entries with no generatable spec may write "N/A".

Show the command surface `cli-printing-press generate <name>` produces, so reviewers can see what commands/flags/hosts the entry yields without checking out and generating it:
1. The generated CLI's top-level `--help` output (or its command list).
2. A short table mapping each user-facing command to its endpoint/host, flagging any novel or read-only commands.
-->

## Risk

<!-- What could this break? Include generated output, MCP surface, auth, catalog, publish flow, verifier, scorer, or release behavior if relevant. -->

N/A

## Output Contract

<!-- Required only if this PR changes templates, generated files, manifests, command output, MCP schemas, scorecard output, catalog rendering, or pipeline artifacts. Otherwise write "N/A". -->
<!-- For generator/template changes, name the generated-output evidence: emitted-code assertion, compiled generated CLI case, golden fixture, or why the existing cases are sufficient. -->

N/A

## Verification

<!-- List commands actually run. Say "Not run" with a reason if not run. -->

- [ ] Generator/template change: verified generated output, including emitted-code assertions or compiled generated CLI output
- [ ] Generator/template change: covered the affected fallback or variant shape, not only happy-path fixtures
- [ ] Generator/template change: checked emitted definitions and call sites for matching gates

## AI / Automation Disclosure

- [ ] No AI or automation was used
- [ ] Human-reviewed: AI or automation was used, and a human reviewed the work for intent, fit, and obvious issues before submission
- [ ] AI-reviewed only: an AI agent reviewed the work, but no human reviewed it before submission
- [ ] Fully automated: generated and submitted without human review for this specific change
