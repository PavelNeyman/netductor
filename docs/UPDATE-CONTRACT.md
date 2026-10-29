**EN** · [RU](ru/UPDATE-CONTRACT.md)

# Unified update contract (Phase 1)

> Status: **accepted for Phase 1** (types + transition tests in tree; full engine wrap is Phase 2)  
> Date: 2026-10-29  
> Base: `main` after Phase 0 audit (`d00c78c` / follow-up commits)  
> Code: `internal/update/contract.go`, `contract_test.go`  
> Phase 0 map: [audits/2026-10-29-phase0-update-baseline-e79e87b.md](audits/2026-10-29-phase0-update-baseline-e79e87b.md)

## 1. Goal

One **logical** model for “what is updated where”, shared by CLI, API, Telegram, and Web.  
Clients **must not** invent release asset names or implement a second download/verify engine. They call the backend with **component + instance + target version** (or “latest compatible”); the backend resolves the asset via `AssetName` + release metadata.

Existing implementations (`netductor update`, Edge `agent_update`, op self-update, deploy ship) remain until Phase 2 adapts them behind this contract.

## 2. Component IDs

| ID | Binary family | Typical role |
|----|---------------|--------------|
| `node` | `netductor-linux-*` | Primary or Secondary **host** control-plane binary |
| `tg` | `netductor-tg-linux-*` | Telegram bot (usually Primary) |
| `agent` | `netductor-agent-linux-*` | Edge device agent; optional agent unit on Secondary |
| `op` | `netductor-op-darwin-*` | Mac operator only |

**Out of unified catalog v1 (explicit):** addons (lampac/registry/git), NVR helpers — keep existing addon APIs; do not surface as `ComponentID` until a supported path is documented.

**Instance identity**

| Field | Meaning |
|-------|---------|
| `instance_id` | Stable id: primary node id / secondary agent id / edge `device_id` / `local` for op |
| `role` | `primary` \| `secondary` \| `edge` \| `operator` |
| `platform` | `goos`, `goarch`, optional `extra` (GOARM/GOMIPS) |

A job always targets **(component, instance_id)**. Updating Secondary `node` must not restart Edge `agent` on another instance.

## 3. Canonical fields (`ComponentInstance`)

| Field | Description |
|-------|-------------|
| `component` | ComponentID |
| `instance_id` | InstanceID |
| `role` | Role |
| `platform` | Platform |
| `installed_version` | From VERSION file / package metadata when present |
| `running_version` | Observed from process when reliably known |
| `build_commit` | When embedded / reported |
| `desired_version` | Operator-selected target |
| `available_version` | From check against releases |
| `state` | JobState for in-flight or last terminal op |
| `health_ok` | Last health probe |
| `last_error` | Machine-oriented message / code |
| `updated_at_unix` | Last state change |

**Version display rule:** UI must not show `desired_version` as “running” until `state=succeeded` after health check (Phase 3 will harden product-wide).

## 4. State machine (`JobState`)

```
idle → checking → available → queued → downloading → verifying → installing
        → restarting → health_checking → succeeded → idle

Any of downloading…health_checking may → failed
installing|restarting|health_checking|failed → rollback_in_progress → rolled_back → idle|checking
queued|downloading|verifying → cancelled → idle|checking
unknown → checking|unsupported|idle
```

Implemented as `CanTransition` / `ValidateTransition` in `internal/update/contract.go`.  
Self-transitions allowed (progress heartbeats).

**Terminal for a single job:** `succeeded`, `failed`, `rolled_back`, `cancelled`, `unsupported`.

## 5. Shared backend operations (contract)

| Operation | Purpose | Notes |
|-----------|---------|-------|
| `catalog` | List `ComponentInstance` for fleet | Filter by role/component |
| `detail` | One instance | Includes last job summary |
| `check` | Refresh available_version | Read-only; state checking→available\|idle |
| `compatible` | List tags/assets for instance platform | Uses `AssetName`; no client-side guessing |
| `start` | Begin job | Body: component, instance_id, target_version\|latest, idempotency_key |
| `job` | Get job by id | Progress + state |
| `history` | Recent jobs for instance | Bounded |
| `rollback` | Where supported | Only if previous artifact retained |

**Asset resolution (server-side only)**

```text
asset = AssetName(component, platform)  // empty → unsupported
verify asset ∈ release SHA256SUMS for target tag
```

**Idempotency:** same `idempotency_key` + same target within TTL returns the existing job id (no second download).  
**Concurrency:** at most **one active non-terminal job** per `(component, instance_id)`. Further `start` → conflict or queue policy (v1: reject with conflict).  
**Timeout:** default **900s** (`DefaultJobTimeoutSec`); then `failed` or operator-visible stale.  
**Cancellation:** cooperative between `queued`…`verifying`; after `installing` prefer rollback path if defined.  
**Stale UI:** TG/Web callbacks older than **300s** must re-fetch job/detail before apply-like actions.

## 6. Authorization

| Surface | Auth |
|---------|------|
| CLI on node | Local operator/root (`AuthCLI`) |
| HTTP `/api/update/*` (and future catalog) | Existing session (`AuthSession`) |
| Telegram | Operator allowlist + server-side re-check (`AuthTG`); **no secrets in callback_data** |

Edge device cannot start updates for other instances; Primary may enqueue `agent` job for a `device_id` subject to session/TG auth.

## 7. Mapping to current code (Phase 0)

| Contract | Current path |
|----------|----------------|
| `node`/`tg`/`agent` on host | `netductor update --component …` + `internal/update` |
| `agent` on edge device | `edge cmd … agent_update` + `agentUpdate` |
| `op` | `netductor-op` update / brew |
| Release publish | `scripts/release.sh`, GHA |

Phase 2 will route `start` through these implementations without changing asset naming.

## 8. Tests

```bash
go test ./internal/update/ -count=1
```

Covers: ValidComponent, AssetName matrix, happy-path transitions, illegal transitions, rollback path, self-transition.

## 9. Non-goals (Phase 1)

- Implementing the HTTP catalog handlers (can follow in Phase 2/5/6).
- Signed provenance (R-05).
- Renaming edge binary on disk (Phase 4).
- Automatic fleet-wide bulk update.
