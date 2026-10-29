# Phase 2 follow-up — atomic install + version probe

**Base:** HEAD after `855a27a`  
**Date:** 2026-10-29

## Changes
- `ApplyHostComponent` downloads to `dest.new`, keeps `dest.prev`, renames into place.
- On health failure after restart, restores `dest.prev` when present.
- `probeBinaryVersion` runs `<bin> version` / `--version` and requires target version substring.
- Tests: atomic replace, version probe match/mismatch; existing apply tests use `SkipVersionProbe`.

## Seamlessness (operator feedback)
- CLI flags may change if needed; host must still update cleanly.
- Configs: migrate when formats change.
- UI: do not break; add surfaces for new update features when shipped.
- Engine under the hood for tg/agent confirmed.

## Still open (Phase 2)
- Full failure matrix (disk full, permission, interrupted updater) as dedicated tests.
- Wire `node`/`stack.Apply` through same job registry (optional same phase).
- Edge `agent_update` path unchanged (`confirm=yes`).
