# Architecture

Fleet: **primary** (abroad control plane) + optional **secondary** (RU VPN entry only). See [FLEET.md](FLEET.md). Plan: [PLAN-SECONDARY-VPN-ONLY.md](PLAN-SECONDARY-VPN-ONLY.md).

Single Go binary **netductor** on each VPS:

| Plane | Role |
|-------|------|
| install | packages + systemd units |
| serve | Admin API + static admin UI :8787 |
| vpn | multi-user VLESS Reality + HY2 (sing-box) |
| collect / probes | metrics + alerts via Telegram |
| doctor / status | health |
| tui | Bubble Tea + Huh |

**netductor-agent** on OpenWrt: outbound heartbeat + command poll.  
**netductor-tg**: operator bot.

No Python control plane. No shell install modules.
