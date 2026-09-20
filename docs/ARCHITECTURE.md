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

No Python control plane. No shell install modules.

## Control-plane transport (agent → primary)

| Path | Default | Notes |
|------|---------|--------|
| Admin UI/API | `127.0.0.1:8787` | Not public. Operator uses SSH tunnel: `ssh -L 8787:127.0.0.1:8787 primary` |
| Secondary agent | mTLS `:8789` when certs ready; else plain `:8788` | Token auth always. Prefer mTLS; plain is legacy (`NETDUCTOR_PLAIN_AGENT=1`) |
| Edge agent | `SERVER=` URL in agent config | Prefer **https** or reach primary **via VPN tunnel** after enroll path exists. Plain `http://PUBLIC:8787` is token-auth only and **not** confidential in transit |

Devices do **not** SSH to each other after provision. Operator SSH is Mac → device with `~/.ssh/netductor_primary` only.
