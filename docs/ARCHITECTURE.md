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
| Secondary / Edge agent | mTLS `:8789` | Client cert required; no IP allowlist (edge behind ISP NAT) |
| Edge agent | **mTLS `:8789`** same agent plane | Client certs on router; independent of site VPN; admin `:8787` stays localhost |

Devices do **not** SSH to each other after provision. Operator SSH is Mac → device with `~/.ssh/netductor_primary` only.
