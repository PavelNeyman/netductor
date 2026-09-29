**EN** · [RU](ru/OPEN_ITEMS.md)

# Open items

Authoritative plan: [ARCHITECTURE-PLAN.md](ARCHITECTURE-PLAN.md) · freeze: [ARCHITECTURE-FREEZE.md](ARCHITECTURE-FREEZE.md)

## Closed

- [x] Architecture freeze (code + docs)
- [x] Recover automation 0.9.70–0.9.75 (post-restore, LE re-issue, vpn apply/mux, no footgun knobs)
- [x] Live unattended recover drill 2026-09-27
- [x] Docs EN/RU pairs (42)

## Phase Updates (priority)

- [x] Node: `update check` / `list` / apply with optional pre-backup + API `/api/update/*` + opcatalog (0.9.97)
- [x] TG: Updates screen lists releases + apply tag with pre-backup (0.9.98)
- [ ] Web/TUI: update badge + release picker (same API)
- [x] Pre-upgrade: wait for secondary `backup_pull` ACK (45s soft) before apply (0.9.100)
- [x] Secondary→primary restore: `netductor recover --from-secondary URL --recovery-token TOKEN --key KEY` (already; noted 0.9.100)
- [x] Edge version shown on node card/list (agent heartbeat)
- [ ] Edge agent_update button on Updates screen (enqueue per device)

## Owner next

- [ ] **Show password** toggles in Web/TUI password fields (one-shot deploy secrets) — [WEB-UI-FIXES.md](WEB-UI-FIXES.md) § Show password. Plan only.
- [ ] **Web Control (Day-2) UX simplify** — too many tabs/buttons; [WEB-UI-FIXES.md](WEB-UI-FIXES.md) § Day-2. Plan only.
- [x] Full dual-node smoke (primary + secondary) — live deploy 2026-09-29 (LE, bot, SP, secondary)
- [ ] Hardware e2e (OpenWrt / Tapo / MikroTik)
- [ ] Optional: CI Formula SHA automation

Latest review: [REVIEW-0.9.75.md](REVIEW-0.9.75.md).

## Web UI / Fleet (pending code)

See [WEB-UI-FIXES.md](WEB-UI-FIXES.md) — labels, Lampac checkbox, CF proxy, secondary TOFU path on Mac. **Do not implement until prioritized.**
