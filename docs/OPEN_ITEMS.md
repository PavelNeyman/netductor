# Open items

## Product decisions locked
- [x] Subscription removed (single VLESS/HY2 links only)
- [x] Fleet primary (abroad) / secondary (RU) — not equal active-active cores
- [x] No VPN load-balancer VPS
- [x] Lampac localhost only; prefer secondary placement
- [x] TG active on primary; standby on secondary via SOCKS→primary (not MTProxy)

## Still open
- [ ] Dual-VPS clean smoke after wipe
- [ ] Real OpenWrt / MikroTik hardware e2e
- [ ] Domain + HTTPS for admin (optional)
- [ ] Path B: separate limited user-facing TG bot (documented idea only)

## Handoff
New agents: start at [AGENT_HANDOFF.md](AGENT_HANDOFF.md).
