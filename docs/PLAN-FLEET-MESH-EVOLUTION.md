# Plan: Fleet resilience & mesh evolution

**Status:** accepted direction 2026-11-17  
**Related:** [ADR-ROLE-REGION.md](ADR-ROLE-REGION.md), [ARCHITECTURE-FREEZE.md](ARCHITECTURE-FREEZE.md), CF-14 in [PLAN-CODE-FIXES-AFTER-B.md](PLAN-CODE-FIXES-AFTER-B.md)

## Goal

Evolve from a single control + single egress deployment toward a **resilient fleet** where:

- node **role** is functional, not geographic;
- nodes can carry **extra workloads** (not only VPN entry/exit);
- loss of one machine does not take down user data-plane or (later) management;
- a path to stronger mesh remains open **without** starting from multi-master SoT.

## Non-goals (explicit)

| Item | Why |
|------|-----|
| Multi-master control (any node writes users/policy) as Phase 0–3 | Split-brain, CA/policy conflicts; high cost for a small fleet |
| Renaming all Go packages (`internal/secondary`, …) in one shot | Compatibility cost; aliases are enough |
| Forcing every node to run build/CI | Build stays a **capability**, not universal |
| Replacing classic TG UI with widgets while Bot API blocks `sendWidget` | Platform constraint |

**Optional endgame only:** multi-master SoT after Phases 1–4 prove insufficient.

---

## Model (target vocabulary)

### Three axes (orthogonal)

| Axis | Meaning | Examples |
|------|---------|----------|
| **Control-ness** | Who holds cluster source of truth | `control` (SoT / lease holder), `member` (not SoT) |
| **Capabilities** | What the node may run | `vpn_ingress`, `vpn_egress`, `dns`, `media`, `build`, `nvr`, … |
| **Region** | Placement label for humans & sub | `ru-msk`, `fi-hel`, `us-nyc`, `de-fra` |

**Role field (compat):** keep on-disk / CLI aliases:

| Canonical (CF-14) | Legacy | Typical capabilities |
|-------------------|--------|----------------------|
| `control` | `primary` | SoT + optional workloads |
| `egress` | `secondary` | usually `vpn_ingress` (+ path to internet / control) |
| `build` | — | `build` (+ optional warm standby later) |
| `edge` | — | site agent / OpenWrt |

Long term, **egress** may be expressed as `member` + capabilities; keep `egress` id for dual-read forever unless a dedicated migration ships.

### Geography

- **Control is not “abroad”.** Control = management plane (API, policy, users, orchestration). It may sit in any jurisdiction you accept for secrets and API availability.
- **Egress is not “Russia-only”.** Egress/ingress = user VPN attachment and/or traffic path; many regions allowed.
- Common prod template `control` in EU + `egress` in RU remains a **recommendation**, not the definition of roles.

### Region string

```text
region ::= <iso3166-1-alpha2> "-" <city-slug>
```

Examples: `ru-msk`, `fi-hel`, `nl-ams`, `us-nyc`.  
Country = ISO alpha-2 (`fi` not `fin`). City = short Latin slug.  
Multiple nodes in one city: same `region`, distinct peer/instance id (or suffix `-2` only in **hostname**, not in region id).

### Hostname convention

```text
nd-<role>-<region>           # e.g. nd-control-fi-hel, nd-egress-ru-msk
nd-build-<region>
nd-edge-<site>
```

### State hygiene (required for later HA / mesh)

Any cluster-wide document (peers, users, policy, …) should grow toward:

- `revision` / monotonic version  
- `updated_at`  
- `writer_id` (node or lease identity)  

Even with a single writer today — so standby and future replication do not require a format break.

---

## Phases

### Phase 0 — Docs & mental model (this plan)

**Deliverable**

- [x] This plan + ADR clarification (role ≠ geography; region `cc-city`).
- [x] ARCHITECTURE / FREEZE wording: remove “primary = abroad” as definition; keep as example topology.
- [x] UI/docs glossary: control / egress display labels (API keys still primary/secondary).

**Acceptance:** new docs never define control by country.

---

#
## Sub / VLESS profile labels (locked 2026-11-17)

- URI fragment uses **region + flag** (`#🇷🇺-ru-msk`); **do not** put role in the fragment.
- Source of truth: peer `region` (operator at deploy / `secondary region`); empty → legacy IP-based label.
- Hostname `nd-*-ru-msk` may **hint** region for labels and migrate backfill; IP geolocation is UI suggest only.
- No interactive prompts during `stack apply`; residual empty region → doctor / day-2 set.
- Label-only change: subscription refresh updates names; UUID/endpoints unchanged (manual single-link imports keep old display name until re-added).

## Phase 1 — Naming, region, capabilities (schema)

**Intent:** node is a unit with labels; not only “VPN secondary”.

**Work**

1. Peer / device / node records: optional `region` (`cc-city` validation).
2. Optional `capabilities[]` (or start with flags: `vpn_ingress`, `build`, …).
3. Sub / profile **display names** include region when set (`RU Moscow`, `nd-egress-ru-msk`).
4. Doctor / Web / TG / CLI: show canonical role + legacy in parentheses where helpful.
5. Hostname hints in deploy wizards (`nd-egress-ru-msk`).

**Compatibility:** default empty region; existing fleets unchanged.

**Acceptance**

- Deploy/register can set `region` without breaking agents on old binaries (ignore unknown fields).
- Sub shows human label when region present.
- No requirement to migrate role files off `primary`/`secondary` yet.

---

### Phase 2 — Data-plane resilience (multi-egress)

**Progress:**
- [x] Multiple online secondaries in `/sub` (active + prepared/ready)
- [x] Region labels on profiles
- [x] Health-driven drop from sub (require singbox_ok from heartbeat)
- [x] Preferred IP order for /sub (`secondary prefer`); uplink_ok demotes sort only
- [x] Auto-promote intentionally **manual** (runbook); prefer order only via `secondary prefer`

**Intent:** first real “mesh” users care about — VPN survives loss of one entry node.

**Work**

1. Multiple online egress peers in registry (already partially true with A/B peers).
2. Health signals per peer (443 / agent / optional probe).
3. Subscription / profile set: include healthy `prepared|ready|active` peers; drop `failed` / dead.
4. Preferred / active peer policy documented; promote remains explicit until R5-class gates exist.
5. Operator runbook: [RUNBOOK-MULTI-EGRESS.md](RUNBOOK-MULTI-EGRESS.md) (RU: [ru/RUNBOOK-MULTI-EGRESS.md](ru/RUNBOOK-MULTI-EGRESS.md)).

**Non-goal:** automatic global policy rewrite by every node.

**Acceptance**

- Two egress in different regions (or same); kill one → clients still have a working profile path after sub refresh.
- Control outage does not immediately destroy existing data-plane sessions on egress (best-effort; document limits).

---

### Phase 3 — Control HA (active / standby)

**Notes:** [PHASE3-CONTROL-STANDBY-NOTES.md](PHASE3-CONTROL-STANDBY-NOTES.md).

**Progress:**
- [x] Agent multi-endpoint read path (`SERVER` + `SERVERS` failover)
- [x] SoT pull to standby inbox (`control sync-pull` / `/api/control/backup/latest`) — apply still explicit
- [x] Control role + promote CLI + API write fence on standby
- [ ] DNS/API cutover runbook live test

**Intent:** management plane survives loss of the active control node.

**Work**

1. Replicate SoT to standby (build on backup/peer pull; define required datasets).
2. Lease or explicit **promote** to active control.
3. Agents accept **list of control endpoints** (try active, fall back).
4. DNS/VIP or operator doc for `api` / redirect endpoints after promote.
5. Fencing notes: old active must not write after promote.

**Acceptance**

- Controlled promote: standby becomes SoT; agents reattach; users/policy apply works.
- No dual live writers in normal operation.

---

### Phase 4 — Node API everywhere, writes to leader

**Intent:** every node exposes local day-2 API; cluster writes still single-writer.

**Work**

1. Thin node API on members (doctor, local services, agent status) — extend existing agent/node surfaces.
2. Op/Web/TG cluster view; mutations that touch SoT go to **lease holder only**.
3. Clear errors when leader unreachable (read-only local vs cannot change users).

**Acceptance**

- Operator can inspect any node without SSH for supported read paths.
- User/policy mutations fail closed if leader is down (unless Phase 3 promote done).

---

### Phase 5 — Optional control mesh (multi-master) — decision gate

**Only if** Phases 2–4 are insufficient.

Requires: conflict rules for users/policy, membership, authenticated writers, test matrix for partition.  
**Default recommendation:** stay on active/standby; do not schedule Phase 5 until explicitly re-opened.

---

## Dependency order

```text
Phase 0 (docs)
    → Phase 1 (region / capabilities / labels)
        → Phase 2 (multi-egress data resilience)
            → Phase 3 (control standby)
                → Phase 4 (node API + leader writes)
                    → Phase 5 (optional)
```

Phase 2 must not wait for Phase 3. Phase 3 must not require Phase 5.

---



## Updates & convergence (mandatory)

See [POLICY-UPDATE-CONVERGENCE.md](POLICY-UPDATE-CONVERGENCE.md).

- Operator **skips** intermediate releases: migrations and CF-15 floors must support skip-version jumps.
- Changing standards (hostname `nd-<role>-<region>`, region, capabilities, host baseline) requires a **migrate / host-baseline / RepairState** path so **existing** VPS and routers converge after update—not docs-only and not greenfield-only.
- Day-to-day work on this plan: land commits on `main` **without** cutting a GitHub Release each time. Release in batches at phase boundaries, on operator request, or when a new binary is required for safe migration/compat.

## Implementation notes (for agents)

1. Prefer small PRs per phase checkbox; regression tests for registry/sub/role normalize.
2. Dual-read `primary`/`secondary` until a dated deprecation ships.
3. Do not default `NETDUCTOR_TG_WIDGETS=1` behavior changes on this plan; orthogonal.
4. Evidence for “done”: commands + sanitized output in `docs/audits/` when touching prod paths.
5. EN and RU docs stay meaning-equivalent.

## Progress

| Phase | Status |
|-------|--------|
| 0 Docs / ADR | in progress (this file + ADR update) |
| 1 region + hostname desire + capabilities labels | in progress |
| 2 Multi-egress data-plane | partial (peers/slots exist; productize health→sub) |
| 3 Control HA | not started |
| 4 Node API / leader | not started |
| 5 Multi-master | deferred |
