Status **0.9.410:** Phase 2 CLI+TUI jobs list; cancel cooperative.

# Plan: True thin UI (jobs + state on backend)

**Status:** Phase 0–1 implemented; Phase 2 TUI+Web reattach in **0.9.409** (v0.9.408); Phase 2–4 incremental · **Date:** 2026-10-17 · **Tip context:** v0.9.407+

## Goal

All long-running and multi-step work lives in a **backend process**. Every UI (Web, TUI, Telegram, CLI) only:

1. **Starts** an operation → receives `job_id`
2. **Polls** (or subscribes to) status / log / result
3. **Cancels** if needed
4. After UI restart / page refresh / bot reconnect — **reattaches** to the same `job_id` and shows current state

UI must never be the process boundary for work that can take >2–3 seconds or touch SSH/deploy/build.

---

## Current state (honest map)

| Surface | Backend today | Long ops | Survives UI refresh? |
|--|--|--|--|
| **Primary Web / session API** | `netductor` on VPS | Many short JSON; some enqueue | Partial (agent queues yes) |
| **Telegram** | bot on primary | Mostly enqueue + reply; some sync handlers | Mostly yes for agent cmds; not for in-handler SSH |
| **CLI on VPS** | same process | Sync commands | N/A (terminal is the client) |
| **Operator Web / TUI / CLI** (`netductor-op`) | local process on Mac | **Edge probe/provision, Fleet, primary/secondary/build deploy** run **inside the HTTP/TUI request** (stream or block) | **No** — abort client = stop or orphan without trackable job |
| **Edge day-2** (agent online) | primary `EnqueueCmd` | Queue + agent poll | **Yes** (primary registry) |
| **Build-host ship** | primary job claim | Queue | **Yes** |
| **Git project build** | primary CI | Partially async | Improve |

**Already good patterns to reuse**

- `edge.EnqueueCmd` / secondary cmd queue / build-host `jobs/claim`
- Operator `Reporter` + `Step{ID,Message,Done,Err}` (`internal/operator/events.go`) — structured progress, still tied to one request
- TG “queued” replies for reboot / NVR / edge storage

**Gap**

- No unified **Job** store on **op** (LAN deploys)
- No unified **Job** store on **primary** for operator-triggered bulk ops that today block the HTTP handler (stack apply stream, large doctor, mirror, etc. where still sync)
- Web op has **no** durable `job_id` in sessionStorage → refresh restarts probe

---

## Principles

1. **One job model, two hosts**
   - **PrimaryJobStore** — everything that primary can execute or queue to agents
   - **OperatorJobStore** — everything that requires Mac LAN SSH (factory OpenWrt, first-time VPS deploy from op)
2. **Same API shape** on both (thin clients share code paths):
   ```http
   POST /v1/jobs              { "type": "edge.probe", "params": {...} } → { "job_id": "…" }
   GET  /v1/jobs/{id}         → { status, steps[], log_tail, result, error, updated_at }
   GET  /v1/jobs/{id}/log     → text/event-stream or line poll
   POST /v1/jobs/{id}/cancel
   GET  /v1/jobs?status=running&type=edge.probe
   ```
3. **Types are stable strings** (not free-form): `edge.probe`, `edge.provision`, `fleet.deploy`, `primary.deploy`, `secondary.deploy`, `buildhost.deploy`, `stack.apply`, `git.build`, …
4. **Idempotent start (optional):** `client_token` / `dedupe_key` so double-click does not spawn two identical probes
5. **Secrets stay in params on the server only**; UIs never re-send passwords on poll
6. **CLI** is also a thin client: `netductor-op job wait <id>` / `netductor job wait <id>`

---


## Operator process model (TUI / Web / CLI on Mac)

**Decision (2026-10-17):** do **not** split TUI into a separate product or binary. Split **job lifetime** from **UI lifetime**.

| Process | Role |
|--|--|
| `netductor-op serve` (127.0.0.1:7373) | **Owns** `OperatorJobStore`, runs SSH/deploy/probe workers, persists job log/state |
| `netductor-op tui` | Thin client: `POST/GET/cancel /v1/jobs` only — no in-process SSH for slow paths |
| Browser → same serve | Same API as TUI |
| `netductor-op job wait\|list\|cancel` | CLI thin client |

Rules:

1. Closing TUI or refreshing Web **must not** cancel running jobs (cancel is explicit).
2. TUI on start: if serve is down → start embedded serve **or** print “run `netductor-op serve`” (product choice in Phase 1: prefer **auto-start background serve** on first job if nothing listens on the port).
3. TUI “Running jobs” panel on home: reattach by `job_id` after restart.
4. Optional later: launchd/user agent keeps `serve` always up — **not** required for Phase 0–1.
5. Primary jobs stay on VPS (`/api/jobs`); when TUI operates through a primary session it polls primary, not local serve.

**Rejected:** TUI as sole process holding goroutine jobs (dies with the terminal).

## Job record (minimal schema)

```json
{
  "id": "j_01H…",
  "type": "edge.probe",
  "status": "queued|running|succeeded|failed|cancelled",
  "created_at": "…",
  "updated_at": "…",
  "steps": [
    { "id": "usb_pkgs", "message": "installing kmods", "done": false }
  ],
  "log_seq": 42,
  "result": { },
  "error": "",
  "cancel_requested": false
}
```

- Logs: ring buffer or file under `~/.cache/netductor/jobs/` (op) / `/var/lib/netductor/jobs/` (primary)
- TTL: finished jobs kept 24–72h; running killed on process restart unless we add resume (phase 2)

---

## Phase 0 — Contract + store (both hosts)

| Task | Where |
|--|--|
| Package `internal/jobs` (store interface, memory+disk, types, JSON API helpers) | shared |
| Wire **OperatorJobStore** into `netductor-op` serve mux `/v1/jobs*` | op |
| Wire **PrimaryJobStore** into main API `/api/jobs*` (session-auth) | primary |
| Document types list in this file + `docs/ru/` | docs |
| Tests: create → run stub → poll → cancel | unit |

**Acceptance:** CLI can `POST` a no-op job and `GET` it after process still up; after kill, disk-backed job still readable as `failed` or `interrupted`.

---

## Phase 1 — Operator: edge + fleet (highest pain)

Convert **streaming handlers** to job runners:

| Type | Today | After |
|--|--|--|
| `edge.probe` | sync in `/v1/edge/preview` | job; result = facts + plan card JSON |
| `edge.provision` | stream `/v1/edge` | job; steps = existing deploy stages |
| `edge.offline_prep` | stream | job |
| `fleet.deploy` | stream + Reporter | job + steps |
| `primary.deploy` / `secondary.deploy` / `buildhost.deploy` | stream | job |

Web/TUI (both clients of **serve**, not job owners):

- Start → `POST /v1/jobs` on local serve → save `job_id` in `sessionStorage` / TUI model
- Poll every 1s; render steps + log tail
- On load: if `job_id` present and status running → **reattach**, do not start new
- Banner: “Probe in progress…” with Cancel
- TUI home: **Running jobs** list from `GET /v1/jobs?status=running`
- TUI must not call `deploy.*` / `ProbeDeviceFacts` directly for slow paths once Phase 1 lands

**Acceptance:** start probe → refresh browser → same progress continues; second probe while first running either queues or returns existing `job_id` (product choice: **reject duplicate** for same host).

---

## Phase 2 — Primary: wrap remaining sync bulk ops

| Type | Notes |
|--|--|
| `stack.apply` | already has alerts; expose job + percent/log |
| `git.project.build` / `git.project.sync` | align with CI queue (one at a time already desired) |
| `doctor.run` | optional full report as result blob |
| `backup.create` / `recover.*` | long |
| `vpn.apply` / `policy.apply` / `secondary.sync` | short but still jobs for parity |
| Agent-backed | **do not duplicate** — status API aggregates `EnqueueCmd` as jobs or links `cmd_id` |

TG/Web/TUI on primary:

- Any button that today blocks or “fires and forgets” without id → returns `job_id` and a “status” screen/card

**Acceptance:** Telegram “Stack apply” → message with job id → edit message on completion without depending on webhook duration.

---

## Phase 3 — Full UI coverage checklist

Every user-visible action classified:

| Class | Behavior |
|--|--|
| **Read** (status, list, card) | sync GET OK |
| **Mutate fast** (<2s) | sync POST OK **or** job (prefer job if side effects) |
| **Mutate slow** | **must** be job |
| **Agent queue** | already async; UI shows cmd id + poll result |

Surfaces to audit (tick when job-backed or intentionally sync):

- [ ] Edge: probe, provision, offline-prep, luci, storage day-2
- [ ] Fleet / primary / secondary / build-host deploy
- [ ] Stack apply / rollback / doctor
- [ ] VPN apply, policy, uplink, SR config export
- [ ] Secondary status / sync / prune / agent update
- [ ] Git: project add/sync/build/queue
- [ ] NVR: camera add, PTZ, record, leases (agent)
- [ ] Sites / rooms
- [ ] DNS custom lists
- [ ] Users / profiles / credentials issue
- [ ] Alerts / channel test
- [ ] Cleanup / host-baseline / snapshot doctor
- [ ] TG menu tree (no orphan long callbacks)

**Parity rule:** if Web has a job for X, TG and TUI and CLI must be able to start/wait the same `type` (thin clients).

---

## Phase 4 — UX polish

- Web: global “Jobs” drawer (running + last failed)
- TG: silent edit of status message every N s while job running
- TUI: jobs panel; block navigation away with “detach OK, job continues”
- Op restart: mark running jobs `interrupted`; optional resume only where safe (not half-SSH)
- Metrics: `jobs_running`, `jobs_failed_total{type=}`

---

## Out of scope (for now)

- Moving factory OpenWrt SSH to primary (impossible without path to LAN)
- Multi-op cluster (one Mac op is enough)
- Full event bus (NATS); HTTP poll is enough

---

## Implementation order (practical)

1. `internal/jobs` + op `/v1/jobs` + primary `/api/jobs` (Phase 0)  
2. `edge.probe` + Web/TUI reattach via serve (Phase 1 slice) — **unblocks current Cudy pain**  
3. `edge.provision` + offline-prep  
4. Fleet / primary / secondary / build-host  
5. Primary stack/git bulk  
6. Audit checklist (Phase 3) + TG parity  
7. Jobs drawer / polish  

---

## Risks

| Risk | Mitigation |
|--|--|
| Two backends confuse users | Job list labels `host=op|primary`; Web op only shows op jobs unless session to primary |
| Password in job params on disk | 0600 files; redact in log; wipe params after terminal state |
| Zombie SSH after cancel | context cancel passed into deploy SSH helpers (must thread `context.Context`) |
| Scope creep | Ship Phase 0+1 before touching VPN/git |

---

## Success criteria

1. Refresh during edge probe does **not** start a second install of USB packages.  
2. No deploy/probe handler holds the HTTP request for the full duration (returns `job_id` in <500ms).  
3. Same job model documented once; all four UIs use it for slow paths.  
4. Agent queues remain the path for day-2 edge; not reimplemented as op jobs.

---

## Related

- `internal/operator/events.go` — Step reporter (fold into job steps)
- `docs/PLAN-EDGE-DEPLOY-HYBRID.md` — edge phases
- Edge agent `EnqueueCmd` — day-2 already thin
