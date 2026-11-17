# Policy: Updates, migrations, and desired-state convergence

**Status:** mandatory for agents and contributors · **2026-11-17**  
**Related:** [UPDATE-CONTRACT.md](UPDATE-CONTRACT.md), `internal/migrate`, `internal/upgrade` (CF-15), [PLAN-FLEET-MESH-EVOLUTION.md](PLAN-FLEET-MESH-EVOLUTION.md), host-baseline / RepairStateOnUpgrade

This document is the **source of truth** for how fleet changes must land so operators can **skip intermediate releases** and still end in a correct, standard configuration.

---

## 1. Operator reality

- The operator **does not** install every intermediate tag.
- A host may jump `0.9.X → 0.9.X+N` (or stack apply a tip after many commits).
- Therefore **every merge to main** must keep:
  - **forward** compatibility for skip-version upgrades, or
  - an explicit **upgrade floor** (`min_from` / `DeclareBinaryFloor`) when a jump is unsafe.

Do **not** assume “everyone is on last week’s binary.”

---

## 2. Binary / release rules (CF-15)

| Rule | Detail |
|------|--------|
| Breaking on-disk or apply-order change | Same commit: `upgrade.DeclareBinaryFloor(id, minVersion, reason)` |
| Idempotent / additive migration | **No** floor |
| Release artifact | `upgrade.json` from floors; preflight fails if local `< min_from` |
| Skip-version | Allowed when no floor blocks the path; migrations must still converge |

Ordinary features never rely on “you must have run 0.9.Y first” **unless** a floor says so.

---

## 3. Startup / stack migrations (`internal/migrate`)

Contract of each step:

```text
Required(ctx) → whether work is still needed (drift-aware)
Apply(ctx)    → make system match desired state (idempotent)
Verify(ctx)   → confirm outcome
```

| Rule | Detail |
|------|--------|
| Journal | Durable under state dir; fail-closed on corrupt journal |
| Idempotent | Re-run safe; `Required` false when already compliant |
| Drift | If an older node drifted (manual edit, partial deploy), **Required must be true** until standard is met |
| Order | Runner before Ready / as part of stack apply post-path (existing hooks) |
| Role-aware | Migrations that only apply on control or egress must no-op cleanly on other roles |
| Tests | Prefer regression tests for Required/Apply when touching prod paths |

Existing patterns: `ensureRoleFileMigration`, `repairStateOnUpgradeMigration`, baseline dirs, aptguard policy, etc. **Extend this registry** for new standards (hostname, region labels, capability defaults, …).

---

## 4. Desired-state convergence (IaC mindset) — **mandatory**

Netductor treats the **code + role + documented conventions** as desired state. After update, the node must **converge**, not only “start the new binary.”

### 4.1 What “converge” means

On `stack apply`, `migrate run`, host-baseline, agent upgrade, and edge provision/update paths (as applicable):

1. **Detect** deviation from the current standard (hostname pattern, role file aliases, unit env, paths, firewall role, naming conventions, …).
2. **Apply** the standard automatically when safe.
3. **Verify** or surface a clear doctor/migrate failure when automatic change is unsafe (needs operator confirm).

### 4.2 Examples (not exhaustive)

| Area | Desired state | On upgrade |
|------|---------------|------------|
| Hostname | Convention e.g. `nd-<role>-<region>` when region/role known | If non-compliant and policy enabled → rename + persist; else doctor WARN until set |
| Role file | Accept control/egress; dual-write legacy as required | `ensureRoleFile` / Normalize path |
| Firewall / SSH port | Role baseline (52222 key-only post-bootstrap) | host-baseline / EnsureHostBaseline |
| LE hooks / state JSON modes | Current RepairState paths | repair migration (versioned stamp when logic grows) |
| Peer/region labels | Schema from fleet plan Phase 1 | migrate + default empty region |
| Edge / OpenWrt | Templates and agent identity rules | provision + agent_update convergence |

**Routers and VPS are in scope:** any managed device type must move toward the same documented standard after an update path that touches it—not only greenfield deploys.

### 4.3 Safety

- Prefer **idempotent, reversible** fixes.
- Destructive renames (hostname changing management DNS, LE cert CN): implement with explicit migration id, prechecks, and doctor notes; do not silent-break redirect/LE without hooks.
- Never skip Verify for “best effort” on security baselines (SSH, mTLS material completeness).

### 4.4 Implementation vehicle

| Mechanism | Use for |
|-----------|---------|
| `internal/migrate` Migration | Cluster/node desired state once binary is new |
| `EnsureHostBaseline` / host-baseline | Firewall, timers, role file, snapshot |
| `RepairStateOnUpgrade` | LE hooks, JSON modes, known path repairs (stamp when expanded) |
| Edge agent update / template apply | Router UCI/agent layout |
| Doctor | Report residual drift that was not auto-fixed |

New fleet-plan items (region, hostname convention, capabilities) **must** ship with a migration or baseline check, not “docs only.”

---

## 5. Seamlessness (ops + UI)

From [UPDATE-CONTRACT.md](UPDATE-CONTRACT.md) — still binding:

- Updates apply cleanly on hosts that last ran an **older** client/CLI.
- Config format changes ⇒ **migrate**, not “manual edit the JSON.”
- UI: do not break existing surfaces; **add** controls for new features (EN/RU).
- One download/verify engine; clients do not invent asset names.

---

## 6. Engineering cadence (releases vs main)

| Situation | Action |
|-----------|--------|
| Normal plan work on `main` | Commit + push; **do not** cut a GitHub Release / Formula bump for every commit |
| Operator needs a batch of fixes | Cut **one** release when Phase slice is ready or on request |
| Breaking migration / floor / cannot develop further without new binary on host | **May** cut a release (document why) |
| Unrelated hot feature requested | May release that slice; then return to plan |

Agents must **not** assume production already runs tip of `main`. Design migrations and floors so that the **next** operator `stack apply` from their current version still converges.

---

## 7. Checklist before merging fleet / naming / baseline changes

1. [ ] Idempotent migration or baseline hook if desired state changed.  
2. [ ] `Required` true on drifted legacy hosts (not only empty greenfield).  
3. [ ] Floor declared if skip-version is unsafe.  
4. [ ] Doctor or migrate status surfaces residual drift.  
5. [ ] EN + RU docs for operator-visible behavior.  
6. [ ] No reliance on “must install every intermediate tag.”  
7. [ ] No drive-by release unless §6 requires it.

---

## 8. Progress hooks

- Fleet naming / region / hostname convergence: track under [PLAN-FLEET-MESH-EVOLUTION.md](PLAN-FLEET-MESH-EVOLUTION.md) Phase 1 + this policy.  
- Expand `repairStateOnUpgradeStamp` when RepairState gains paths.  
- Prefer one migration id per coherent standard (e.g. `2026-11-hostname-convention`) rather than silent one-off scripts.
