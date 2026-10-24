> **Stop: v0.9.455.** Thin-UI jobs core + Phase 3 audit + Operate Result dock. Remaining: review follow-ups F1–F10 (below), lab e2e (OpenWrt/NVR/MikroTik), future (OIDC/guest session).

- [x] Optional **build-host** VPS (SSH executor, 0.9.361); agent-pull later
- [x] Internal releases P0–P4 + multi-repo projects (0.9.282–0.9.283); channel face vs uplink alerts (0.9.284); see [CI-WORKFLOWS.md](CI-WORKFLOWS.md)
- [x] Edge deploy hybrid (shared backend + thin UI; op-SSH factory; no TG deploy) — [PLAN-EDGE-DEPLOY-HYBRID.md](PLAN-EDGE-DEPLOY-HYBRID.md) **done** (lab e2e remains)
**EN** · [RU](ru/OPEN_ITEMS.md)

# Open items

Authoritative plan: [ARCHITECTURE-PLAN.md](ARCHITECTURE-PLAN.md) · freeze: [ARCHITECTURE-FREEZE.md](ARCHITECTURE-FREEZE.md)


## Blue-Green Secondary (2026-10-24)

- [x] R1 blockers (0.9.497): validate, WSS isolation, promote/remove, material fail-closed
- [x] R2 recover (0.9.498)
- [x] R3 inventory + API (0.9.498)
- [x] R4 Web/TG/CLI inventory UI (0.9.499)
- [ ] **R5 live cutover** on lab/prod VPS — operator; runbook [BLUE-GREEN-SECONDARY.md](BLUE-GREEN-SECONDARY.md)

## Stop line

**v0.9.455** (code tip 2026-10-19; docs synced 2026-10-20) — Operate Result dock (0.9.422–0.9.426), offline-prep token fallback (0.9.427). Latest published release: v0.9.426. Prior: 0.9.414 thin-UI jobs core; 0.9.404 cleanup.

### Review 2026-10-20 follow-ups (static full-complex; tests not run)

Report: [REVIEW-REPORT-2026-10-19.md](REVIEW-REPORT-2026-10-19.md) · plan: [REVIEW-PLAN-2026-10-19.md](REVIEW-PLAN-2026-10-19.md). Items marked **owner** wait for approval because they change code.

- [x] **Run `go test ./internal/... ./cmd/... -count=1`** — passed 2026-10-20 on tip 0.9.429 (`GOMAXPROCS=1 NETDUCTOR_DISABLE_VPS_WORKER=1`, timeout 180s). `internal/incident` 0.027s, no fork-bomb. All packages ok.
- [x] **F1 (S-1, Medium, owner)** revoke VPN sub-token on user delete (`RevokeNative`) + reject disabled users in `SubscriptionBody` + test
- [x] **F2 (Q-1, Medium, owner)** add Web i18n keys `edge_st_g_bak`, `rd_mode`, `rd_pos`, `result_hint` (EN+RU)
- [x] **F3 (Q-2, owner)** self-assignment in `internal/git/projects.go:108`
- [x] **F4 (Q-3, owner)** gofmt commit (129 files) + `gofmt -l` in suite
- [x] **F5 (owner)** remove verified dead code (no-caller funcs removed 0.9.431; test-only kept) DC-A (~45 funcs, ~25 consts/vars); decide DC-B per item
- [x] **F6 (S-2, Medium, owner)** recovery TLS: fail closed on invalid pin, explicit first-contact flag
- [x] **F7 (S-5/R-4)** `httpx.NewServer` on node serve only (60s). Op serve stays 2h. Other listeners not migrated.
- [x] **F8** `ufw delete` via argv; `wg pubkey` via stdin. Docker `curl | sh` left on purpose.
- [x] **F9 (S-8, owner)** propagate `saveDevices` errors in `internal/edge`
- [x] **F10 (R-1…R-8, owner)** refactors: split god functions (`registerEdgeAPI` 937 lines, TG `handleCallback` 779, …), shared helpers
- [x] **D-1 (Medium, owner)** — RU pairs for surviving EN-only ops docs (2026-10-20). See PLAN-DOCS-D1.md.
- [x] **D-1 (Medium, owner)** RU twins (missing EN deleted earlier; survivors translated): CI-WORKFLOWS, CLEANUP, DAY2-GROUPS, DISASTER-RECOVERY, GH-PRIVATE-SETUP, OPENWRT-LAB, PLAN-CI-NEXT, PLAN-EDGE-DEPLOY-HYBRID, PLAN-GIT-CI-OPS, PLAN-INTERNAL-RELEASES, PLAN-NVR-IMPLEMENTATION, PLAN-NVR-UI, PLAN-REMOVE-TG-TOPICS, PLAN-TG-MENU, RECOVER-BOT, RELEASE, REVIEW-PROGRESS, TG-ALERTS-CHANNEL, YOUTUBE-ADS
- [ ] **§8 live fleet smoke** (`netductor doctor`, `host-audit`, `channel status`, secondary agent version) — needs operator access
- [x] Static review run, report and plan published (EN+RU); handoff / OPEN_ITEMS / AGENTS stop lines aligned to 0.9.427

- [x] Product improvements plan (no hardware e2e) — [PLAN-PRODUCT-IMPROVEMENTS.md](PLAN-PRODUCT-IMPROVEMENTS.md) **closed** (2026-10-12); TG menu structure pass included


Full review complex: [REVIEW-SUITE.md](REVIEW-SUITE.md) · [ru/REVIEW-SUITE.md](ru/REVIEW-SUITE.md).

Hygiene / dead-code / allowlist campaign (post-0.9.403 tip): [CLEANUP-PASSES-2026-10.md](CLEANUP-PASSES-2026-10.md) · [ru/CLEANUP-PASSES-2026-10.md](ru/CLEANUP-PASSES-2026-10.md). Tip may be **ahead of** GitHub Release until 0.9.404+ is cut.



**v0.9.455** (2026-10-13) — build-host recover + fleet optional; prior 0.9.349 SR General.

### Closed in 0.9.320–0.9.331 (security / DNS / SSH)

- [x] Metrics display + channel HTML tables + recovery alerts EN/RU
- [x] unattended-upgrades (security-only), integrity manifest, backup age, SERVFAIL alerts
- [x] TG reboot pick-node + one-shot codes; Blocky DoT (primary)
- [x] SSH allowlist: TG Tools, API, agent, Web Advanced, opcatalog (`m:sshkeys:`)
- [x] Reboot challenge parity: API + Web Advanced + opcatalog (same codes as TG)

### Closed in 0.9.310–0.9.318 (ops / security)

- [x] Path e2e, custom policy presets, devices, canary mismatch, incident pack
- [x] Backup verify CLI/timer + TG/Web/API
- [x] Post-review: verify caps, incident no shell, update session, decrypt RAM, stream tar
- [x] Clip share token works without op session (`GET /api/nvr/clip` exempt)
- [x] Release GET handlers requireSession; nvr secrets/go2rtc perms in doctor
- [x] TG Day2 Tools (no Status/Users/Fleet clones); hub labels via `T()`

### Still open (lab / product, not code blockers)

- [ ] **Read-only guest operator session** (status/users view only; no stack/firewall/reboot) — future
- [ ] **OIDC operator login** — design in [OIDC-DESIGN.md](OIDC-DESIGN.md); implement when external IdP exists — future


## Design plans (docs)

- [x] [PLAN-SERVICE-ACCESS-POLICY](PLAN-SERVICE-ACCESS-POLICY.md) **P0–P3 done**; **P0–P4 done** for policy product surface (catalog, routes, UI, presets, doctor, service-net VIP); dual-node smoke still operator
- [x] [HOST-AUDIT](HOST-AUDIT.md) — one-shot bundle + residual packages/paths/apt sources (0.9.197–0.9.198); live primary+secondary CLEAN 2026-10-03
- [x] Expand host-audit denylist (RMM/otel/avahi + timeweb-zabbix apt) — script ↔ Go in sync (0.9.198)
- [x] Firewall status in all UIs + alerts (0.9.259: metrics/digest/TG AlertOnce)
- [x] Baseline snapshot at end of install (0.9.259) + doctor auto-diff listen/units (0.9.260)


## Closed

- [x] Architecture freeze (code + docs)
- [x] Recover automation 0.9.70–0.9.75 (post-restore, LE re-issue, vpn apply/mux, no footgun knobs)
- [x] Live unattended recover drill 2026-09-27
- [x] Docs EN/RU pairs (42)

## Phase Updates (priority)

- [x] Node: `update check` / `list` / apply with optional pre-backup + API `/api/update/*` + opcatalog (0.9.97)
- [x] TG: Updates screen lists releases + apply tag with pre-backup (0.9.98)
- [x] Web: Updates tab + badge + release picker/apply (0.9.101)
- [x] TUI: header chip when op update available (cached 30m) (0.9.102)
- [x] TG notify on new GitHub release (collect `release_update`, once/day per tag) (0.9.102)
- [x] Pre-upgrade: wait for secondary `backup_pull` ACK (45s soft) before apply (0.9.100)
- [x] Secondary→primary restore: `netductor recover --from-secondary URL --recovery-token TOKEN --key KEY` (already; noted 0.9.100)
- [x] Edge version shown on node card/list (agent heartbeat)
- [x] Edge agent_update buttons on TG Updates (0.9.101)
- [x] Primary apply queues secondary `upgrade` for online agents (0.9.103)

## Closed recently

- [x] Stack orchestrator MVP (0.9.116)
- [x] Peer backup prune on secondary

- [x] TG OOM: backup exclude + out-of-process update + MemoryMax (0.9.115)

- [x] Fleet health digest API/CLI/TG (0.9.106)
- [x] DR checklist in TG + docs/DISASTER-RECOVERY.md
- [x] Optional alerts private topic (`telegram_alerts_thread_id`)
- [x] scripts/release.sh local release helper
- [x] btnDisabled / m:noop for spent actions


- [x] TG alert batch + dedupe + hub re-pin throttle (0.9.105)

## Verification plan

Phased scenarios + module review: [VERIFICATION-PLAN.md](VERIFICATION-PLAN.md) (RU: [ru/VERIFICATION-PLAN.md](ru/VERIFICATION-PLAN.md)). Mark progress there.

- [x] Unified **project ship** S0–S6 core (0.9.357); UI parity TG/Web open (pause/wait no VPS slot; Mac multi-wait; auto tag) — [PLAN-PROJECT-SHIP.md](PLAN-PROJECT-SHIP.md)

## Owner next

- [x] Force-update live VPS to **0.9.197** (primary stack apply + secondary `cmd upgrade:v0.9.197`) 2026-10-03
- [x] Dual-node smoke after update (VPN, bot, stack, secondary, policy) — live 0.9.261 **2026-10-06**
- [x] Edge template VPN UI + policy hardening (0.9.164–170)
- [x] Remove legacy VPS admin UI (0.9.171)
- [x] Unit test expansion + opcatalog unique IDs (0.9.172)

- [x] **Show password** Web (Show/Hide) + TUI Ctrl+P (0.9.103)
- [x] Day-2 groups in opcatalog (`Groups()`) → Web tabs / TG Tools / TUI catalog (0.9.104)
- [x] Full dual-node smoke (primary + secondary) — live deploy 2026-09-29 (LE, bot, SP, secondary)
- [ ] Edge stock-OpenWrt contract — **lab e2e only** (code/hybrid done) — [OPENWRT-LAB.md](OPENWRT-LAB.md)
- [x] Site rooms UI — [PLAN-SITE-ROOMS.md](PLAN-SITE-ROOMS.md) (store/API/Web/TG; max 5 photos, 2026-10-09)
- [ ] NVR hardware e2e (C200 record/live/clip) — code: TPAP Go, UI, doctor, retention alerts done; [PLAN-NVR-IMPLEMENTATION.md](PLAN-NVR-IMPLEMENTATION.md)
- [ ] Hardware e2e (OpenWrt / Tapo / MikroTik) — **lab** — [OPENWRT-LAB.md](OPENWRT-LAB.md); radio detect 0.9.271
- [x] Remove TG private topics (keep alerts channel) — [PLAN-REMOVE-TG-TOPICS.md](PLAN-REMOVE-TG-TOPICS.md) (0.9.269)
- [x] Private GH + dual tokens runbook — [GH-PRIVATE-SETUP.md](GH-PRIVATE-SETUP.md)
- [x] release.sh idempotent asset upload (422 skip)
- [x] Edge plan golden: Cudy-like dual-NIC + RPi fixtures
- [x] netductor-op project build (clone/detect/build/dist/--upload) (0.9.309)
- [x] Formula SHA automation (`release.sh` + `scripts/update-formula-sha.sh`) (0.9.310)

Latest review: [REVIEW-PROGRESS.md](REVIEW-PROGRESS.md).

## Web UI / Fleet

See [WEB-UI-FIXES.md](WEB-UI-FIXES.md) — **implemented** 0.9.93–94 / 0.9.259.
- ~~OpenWrt opkg feed / ipk for agent~~ — **rejected** (0.9.155): first-boot binary+SCP; day-2 stack/agent_update. Revisit only if fleet needs offline opkg without primary.

## Channel health + incident logs (2026-10-07)

- [x] Alert batch: solo ~2s flush; multi coalesce ≤8s; same key one (0.9.274)
- [x] Channel probes + CLI/doctor/collect alerts (0.9.275)
- [x] Journal export last 1h on alert + API/UI download (0.9.276)
- [x] Log retention timer (default 04:00 MSK / 01:00 UTC), API+TG+catalog (0.9.276)
- [x] Internal releases P0–P4 + later: auto-build on mirror-fetch, metrics local_release_* + build status (0.9.310)

## Edge USB storage

- Planned: [PLAN-EDGE-USB-STORAGE.md](PLAN-EDGE-USB-STORAGE.md) / [ru](ru/PLAN-EDGE-USB-STORAGE.md) — extroot + internal pin/sync + alerts (post-0.9.402).

| Thin UI jobs (all surfaces) | draft | [PLAN-THIN-UI-JOBS](PLAN-THIN-UI-JOBS.md) |

### Deferred (product)

- **Web terminal** (xterm.js + agent stream/PTY or exec allowlist) — not in tree; agent path preferred over raw SSH.
- **Web file manager** (agent `fs_list`/`fs_read`/`fs_write` with path allowlist) — deferred.
- **UI skins A/B/C** — implemented in operator Web (Settings → skin).

- [x] **S-1 note** disable revokes the subscription URL. Re-enable issues a new token. Rename moves the token (0.9.439).
