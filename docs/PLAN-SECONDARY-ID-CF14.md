# Plan: Secondary / egress identity & CLI (CF-14)

**Status:** planned  
**Date:** 2026-11-22  
**Related:** [ADR-ROLE-REGION.md](ADR-ROLE-REGION.md), [PLAN-FLEET-MESH-EVOLUTION.md](PLAN-FLEET-MESH-EVOLUTION.md), CF-14 hostname rules

## Problem

Operators now use **CF-14 hostnames** (`nd-egress-ru-msk`, `nd-control-nl-ams`) and region labels, but:

| Layer | Today | Friction |
|-------|--------|----------|
| Agent / registry **id** | `secondary-<hex>` (e.g. `secondary-bb9952c66617ca91`) | Opaque; stack apply / `secondary cmd` show hex |
| CLI command group | `netductor secondary …` | Word “secondary” vs role **egress** |
| `secondary status` first column | always agent id | Hostname is second column only |
| Cmd target | **id only** | Cannot `cmd nd-egress-ru-msk upgrade:…` |

Renaming **on-disk agent id** is **not** a free rename: mTLS client CN/paths, enroll tokens, heartbeat, offsite backup keys, and journals may key off id.

## Goals

1. **Human target resolution:** any CLI that takes a secondary/egress target accepts **id OR hostname OR region-unique short name**.
2. **Display:** status/cluster/stack notes prefer **hostname** (CF-14), id secondary in verbose/`--json`.
3. **Vocabulary:** dual-read **egress** aliases for CLI without breaking `secondary` (compat window).
4. **Do not** force-rewrite historical agent ids in one shot unless a safe migration ships.

## Non-goals

- Multi-master / Phase 5.
- Renaming Go package `internal/secondary` in the same change set.
- Changing OpenWrt edge device ids in this plan (separate: `cudy-…` already operator-chosen).

---

## Target model

```text
Stable identity (never required to equal hostname):
  agent_id = secondary-<hex> | future: egress-<hex> for NEW agents only

Display / operator handle:
  hostname = nd-egress-<region>[ -N ]   (nodes DesiredHN / Name)
  region   = ru-msk | …

Resolve(target string) → agent_id:
  1. exact agent_id match
  2. exact hostname / Name match (unique)
  3. unique region match if only one online egress in region
  4. else error: ambiguous / not found
```

---

## Phases

### Phase A — Resolve + display (low risk, do first)

**Code**

- `internal/secondary`: `ResolveTarget(s string) (id string, err error)` implementing the order above.
- Wire into: `secondary cmd`, `region`, `remove`, `prefer` (if id-like), stack apply hint printer, TG/op status lines.
- `secondary status` (human):  
  `nd-egress-ru-msk  online  id=secondary-bb99…  region=ru-msk  agent=0.9.761`  
  or keep tabular but **hostname first**.
- Stack apply NOTE:  
  `netductor secondary cmd nd-egress-ru-msk upgrade:vX` (resolved server-side).

**Tests**

- Unique hostname → id; ambiguous two `ru-msk` without hostname → error; id still works.

**Docs**

- Update CLI help / `docs/FLEET.md` one paragraph.

**Acceptance**

- Operator never needs to copy hex for day-2 cmds when hostname is set.

### Phase B — CLI alias `egress` (compat)

**Code**

- `netductor egress …` as alias of `secondary …` (same flags).
- Help text: “egress (alias: secondary)”.
- i18n keys dual.

**Acceptance**

- Scripts using `secondary` keep working ≥ 2 minor releases.

### Phase C — New agents id prefix (optional)

**Code**

- New `IssueToken` / join: id prefix `egress-` instead of `secondary-` (or `nd-e-` short).
- Dual-read loaders accept both prefixes forever.

**Migration**

- **No** forced rename of existing `secondary-*` rows.

### Phase D — Optional id rewrite migration (high cost — gate)

Only if product requires id == hostname.

**Requires**

- Quiesce agent; re-issue mTLS client under new id; update registry, tokens, backup anchors; dual-read old id → new for one release; document recover if mid-fail.

**Default recommendation:** **skip D**; A+B give operator UX without identity surgery.

---

## Implementation notes

1. Hostname source of truth: secondary registry `Name` / nodes `Hostname` / DesiredHN — prefer the same field `secondary status` already shows as host.
2. Do not resolve against **offline** duplicates without `--force` (ghost `secondary-efd9…` style).
3. JSON/`--json` keeps stable `id` field for automation.
4. EN+RU help strings.

## Out of scope follow-ups

- Primary CLI `netductor control` alias for `…` long term.
- Sub profile labels already CF-14-ish; independent.

## Progress

| Phase | Status |
|-------|--------|
| A Resolve + display | **todo** |
| B `egress` CLI alias | todo |
| C new id prefix | todo (optional) |
| D rewrite existing ids | deferred / not default |

---

## Operator today (until A lands)

```bash
# id is still the cmd key
netductor secondary cmd secondary-bb9952c66617ca91 upgrade:v0.9.761
netductor secondary status   # host column = nd-egress-ru-msk
```
