# PLAN: Build-host (locked 0.9.364+)

## Model (non-negotiable)

| Rule | |
|--|--|
| Orchestrator | **operator (Mac) only** — same as primary/secondary |
| Primary | registry, ship control plane, receive backups, tunnel ends — **never** SSHs to B for deploy/build |
| Build execution | **agent only** — no primary→B SSH for CI |
| Operator SSH | Mac → B with **same** ed25519 as P/S; pubkey installed at deploy |
| Data plane B | **PB + BP only** + SSH :52222. No public API/agent ports |
| Agent → Primary | **tunnel only** (`10.87.12.1` / failover `10.87.13.1` :8789). No WAN :8787 |
| Failover | Full dual-path like SP/PS: health, alerts, auto switch PB↔BP |
| Multi-host | First-class from the start (N build hosts, label, slot pick) |
| Fleet wizard | **Last** phase after everything else works |
| Backups | B→P (and P can push→S); P/S snapshots **also** land on B so each of {P,S,B} holds copies of all three; recover API temporary like secondary |

## Secondary reference (tunnel)

On secondary today:

- Preferred core URL: `https://10.87.10.1:8789` (SP)
- Public URL kept as **fallback** file for emergency
- Failover tick switches agent URL tunnel↔public and can move users SP on VLESS down

**Build-host differs:** **no public agent fallback**. Only PB/BP. If both paths down → offline + alert (operator SSH still possible on :52222).

## Dual paths PB / BP

| Path | Subnet | WSS | Role |
|--|--|--|--|
| **PB** | 10.87.12.0/30 | 8446 | Primary listens; **build initiates** (outbound from B) |
| **BP** | 10.87.13.0/30 | 8447 | Build listens; **primary initiates** (outbound from P) |

Failover: prefer PB for agent; on PB down → BP; alerts; auto restore when stable (same thresholds pattern as SP/PS).

UFW on B: allow WSS BP from primary IP, SSH 52222 (key only). Deny other inbound.

## Decisions log (user 2026-10-13)

1. Remove node `/api/build-host/deploy` (and Web node deploy).
2. Web deploy only via **operator** backend.
3. Real agent token at deploy; state on primary.
4. Reliable install agent + node tools on B from op.
5. E2E: tunnels + agent + list online.
6. Idempotent redeploy / agent upgrade from op.
7–11. Full path health/doctor/alerts/UFW/tunnel-only agent URL.
12–14. Agent job queue only (no SSH CI); logs stream; multi-host **now**.
15–20. Ship polish (hard cut old cmds, Mac waiting, version, triggers/secrets/cache parity).
21–23. Ship + host + path alerts linked.
24–28. Full UI parity + i18n + opcatalog.
29. Tunnel-only B; mirror S control plane; dual path failover.
30. Triple-site backup copies + recover API like S.
31–32. Baseline role=build + docs.
33. **Fleet** — implement at **end**.
34–35. See handoff notes (non-goals explained, not scheduled).

---

## Implementation phases

### P1 — Correct skeleton (current)
- [ ] Delete/disable primary `DeployBuildHost` API + Web node deploy buttons
- [ ] Op deploy: Mac pubkey, real token, agent+bootstrap binary, register multi-host state on P
- [ ] PB/BP bootstrap + health/failover (copy SP/PS patterns)
- [ ] Agent SERVER= tunnel VIP only
- [ ] Doctor + alerts paths
- [ ] Docs: only op

### P2 — Agent is the only builder
- [ ] API: claim/result job for ROLE=build (mTLS or token over tunnel)
- [ ] Ship worker: agent-only executor (remove RunRemoteTestBuild SSH path)
- [ ] Log pull / stream
- [ ] Multi-host scheduling (pick online host, label affinity)

### P3 — Ship product complete
- [ ] Hard cut old command names in all UI/docs
- [ ] Mac waiting_external + complete UX
- [ ] Auto version + secrets/cache/triggers parity

### P4 — UI parity
- [ ] Web op Build-host wizard
- [ ] TG status PB/BP + ship (no legacy deploy text)
- [ ] opcatalog + i18n

### P5 — Backup / recover / baseline
- [ ] B backup → P → optional S; P/S → B mirrors
- [ ] Temporary recover API on B (pattern S)
- [ ] Doctor baseline role=build

### P6 — Fleet (last)
- [ ] Optional Build-host step in Fleet wizard (after P1–P5 green)

---

## Non-goals (explained)

### Matrix CI / full GitHub Actions (item 34)
GitHub Actions **matrix** = one workflow fans out into many parallel jobs (os × go × arch). Full Actions = `uses:` marketplace actions, hosted runners, services:, complex expressions.

We intentionally stay on **netductor workflow subset** (`run` + `container`): enough to ship binaries and stay portable. Matrix fan-out on one VPS would RAM-kill primary/build; if needed later, sequential arch loop on B is enough — not GH matrix semantics.

### CRIU / container freeze mid-step (item 35)
CRIU checkpoints a running process tree to disk and restores later. Attractive for “pause mid-test”, but: needs privileged kernel support, fragile with docker, not portable across hosts. Our pause model is **step boundary snapshot** (state JSON + drop containers) — enough and operable.

