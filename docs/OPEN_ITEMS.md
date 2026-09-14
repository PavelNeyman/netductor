# Open items

## Done (ops pack)
- [x] Alerts, refresh-links, audit/sessions/doctor/dry-run/runbook
- [x] sni-import, multi-relay helper URIs
- [x] **Subscription removed** from product (single links only: VLESS primary / core / HY2)

## Future (shelved, not in progress)

### User Telegram bot — path B (separate bot)
Separate bot token + binary (`netductor-tg-user`), **no** admin handlers.

- Operator issues one-time bind / deep-link → `telegram_id` ↔ VPN user in registry.
- User bot: own VLESS + QR only; optional own traffic stats later.
- Never: user list, nodes, enroll, session, audit, other users’ data.
- Why separate: user-token compromise does not expose the fleet.

**Status:** design only. End-user self-service links may not be needed — decide later.

## Needs hardware / external
- [ ] Site wizard e2e (MikroTik + RPi OpenWrt)
- [ ] Real L3-WL entry (RU IP on whitelist / suitable path)
- [ ] HTTPS / Mini App — only with a domain and explicit need

## Useful to finish without hardware (candidates)
- [ ] TG ↔ Admin parity (no duplicate buttons, consistent cards)
- [ ] Per-user traffic from sing-box stats → registry (useful for operator even without user bot)
- [ ] Access UX: primary default; core/HY2 under “more”
- [ ] Cross-backup peer e2e restore on clean VPS
- [ ] Edge WAN-fallback / policy — validate on real OpenWrt
- [ ] Docs: CLIENT / SHADOWROCKET without subscription; refresh day-1 runbook
