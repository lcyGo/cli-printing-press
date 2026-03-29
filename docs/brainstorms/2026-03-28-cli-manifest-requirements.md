---
date: 2026-03-28
topic: cli-manifest
---

# CLI Library Manifest

## Problem Frame

Generated CLIs in `~/printing-press/library/<cli-name>/` carry no provenance metadata. The only way to know when a CLI was created is to inspect filesystem timestamps, which are unreliable across copies, moves, and time. Metadata exists in `manuscripts/` and `.runstate/`, but those are separate directories tied to the generating machine — if the CLI folder is shared or examined in isolation, that context is lost.

A manifest file in each generated CLI directory would make it self-describing: when it was built, from what spec, by what version of printing-press, and how to trace back to the full generation record.

## Requirements

**Manifest Content**

- R1. Each generated CLI directory contains a `.printing-press.json` manifest file at its root
- R2. Manifest includes generation timestamp (`generated_at`, RFC 3339)
- R3. Manifest includes the printing-press version that produced it (`printing_press_version`)
- R4. Manifest includes the API name used during generation (`api_name`)
- R5. Manifest includes the spec source (`spec_url` and/or `spec_path` — whichever was used)
- R6. Manifest includes the spec format (`spec_format`: openapi3, internal, etc.)
- R7. Manifest includes the derived CLI name (`cli_name`)
- R8. Manifest includes the run ID linking back to the manuscript archive (`run_id`)
- R9. Manifest includes the catalog entry slug if the spec came from the catalog (`catalog_entry`, optional)
- R10. Manifest includes a SHA-256 checksum of the spec file used (`spec_checksum`)

**Schema Versioning**

- R11. Manifest includes a schema version field (`schema_version: 1`) so the format can evolve without breaking consumers

**Write Timing**

- R12. The manifest is written during the publish phase, when the CLI is copied to its library directory — not during intermediate generation steps

## Success Criteria

- Looking at any CLI folder in `~/printing-press/library/` immediately answers: when was this built, from what spec, and by what version of printing-press
- The manifest is present and correct for all newly generated CLIs
- The `run_id` in the manifest can be used to locate the corresponding manuscript archive

## Scope Boundaries

- No CLI to read/query manifests (could be added later, not part of this)
- No regeneration workflow triggered from the manifest (the data supports it, but the feature is out of scope)
- No migration to backfill manifests for previously generated CLIs
- Manifest is write-once at publish time; no update-on-regeneration semantics yet

## Key Decisions

- **Tier 2 scope**: Provenance + regeneration link, but not full portability (no vision features, generation config, or source repo in the manifest). Keeps it simple with room to extend via `schema_version`.
- **Dot-file naming (`.printing-press.json`)**: Signals "tooling metadata" without cluttering the project, but remains easily discoverable.
- **Spec checksum included**: Enables future "is this CLI up to date?" checks without requiring the full spec to be stored in the manifest.

## Outstanding Questions

### Deferred to Planning

- [Affects R5][Technical] Should `spec_path` be stored as an absolute path or relative? Absolute is more informative but machine-specific. Consider storing both or just the URL when available.
- [Affects R12][Technical] Where exactly in the publish pipeline (`internal/pipeline/publish.go`) should the manifest write hook into? Needs codebase inspection during planning.
- [Affects R10][Technical] Should the checksum be computed from the raw spec file bytes or the parsed/normalized spec? Raw bytes is simpler and more verifiable.

## Next Steps

-> `/ce:plan` for structured implementation planning
