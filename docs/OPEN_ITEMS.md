- [ ] Edge deploy hybrid (shared backend + thin UI; op-SSH factory; no TG deploy) — [PLAN-EDGE-DEPLOY-HYBRID.md](PLAN-EDGE-DEPLOY-HYBRID.md) P0 types done
**EN** · [RU](ru/OPEN_ITEMS.md)

# Open items

Authoritative plan: [ARCHITECTURE-PLAN.md](ARCHITECTURE-PLAN.md) · freeze: [ARCHITECTURE-FREEZE.md](ARCHITECTURE-FREEZE.md)

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

## Owner next

- [x] Force-update live VPS to **0.9.197** (primary stack apply + secondary `cmd upgrade:v0.9.197`) 2026-10-03
- [x] Dual-node smoke after update (VPN, bot, stack, secondary, policy) — live 0.9.261 **2026-10-06**
- [x] Edge template VPN UI + policy hardening (0.9.164–170)
- [x] Remove legacy VPS admin UI (0.9.171)
- [x] Unit test expansion + opcatalog unique IDs (0.9.172)

- [x] **Show password** Web (Show/Hide) + TUI Ctrl+P (0.9.103)
- [x] Day-2 groups in opcatalog (`Groups()`) → Web tabs / TG Tools / TUI catalog (0.9.104)
- [x] Full dual-node smoke (primary + secondary) — live deploy 2026-09-29 (LE, bot, SP, secondary)
- [ ] Edge stock-OpenWrt contract: factory reset → one deploy → fully configured (lab = dual-NIC, no manual UCI) — [OPENWRT-LAB.md](OPENWRT-LAB.md)
- [ ] Hardware e2e (OpenWrt / Tapo / MikroTik) — lab notes [OPENWRT-LAB.md](OPENWRT-LAB.md); radio detect shipped 0.9.271
- [x] Remove TG private topics (keep alerts channel) — [PLAN-REMOVE-TG-TOPICS.md](PLAN-REMOVE-TG-TOPICS.md) (0.9.269)
- [ ] Optional: CI Formula SHA automation

Latest review: [REVIEW-PROGRESS.md](REVIEW-PROGRESS.md).

## Web UI / Fleet

See [WEB-UI-FIXES.md](WEB-UI-FIXES.md) — **implemented** 0.9.93–94 / 0.9.259.
- ~~OpenWrt opkg feed / ipk for agent~~ — **rejected** (0.9.155): first-boot binary+SCP; day-2 stack/agent_update. Revisit only if fleet needs offline opkg without primary.

## Channel health + incident logs (2026-10-07)

- [x] Alert batch: solo ~2s flush; multi coalesce ≤8s; same key one (0.9.274)
- [ ] Channel probes: client→secondary:443, secondary→primary Reality/uplink, mTLS 8789, optional ICMP
- [ ] Ring buffer / journal export last 1h on alert + UI download
- [ ] Log retention timer (default 04:00 Europe/Moscow), UI like backup schedule
