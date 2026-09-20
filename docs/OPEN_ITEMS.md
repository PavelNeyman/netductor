# Open items

Baseline: **v0.8.10** (2026-09-20). See [CHANGELOG](../CHANGELOG.md), [REVIEW-2026-09-20-POST](REVIEW-2026-09-20-POST.md).

## Done (through 0.8.1)

- [x] Operator docs aligned to v0.8.1 (bootstrap default, INSTALL/DEPLOY/RUNBOOK/RELEASES, CLI `secondary`/`fleet`, EDGE recovery)
- [x] Go control plane: install, VPN, API, TUI, TG, edge, secondary
- [x] Primary / secondary roles; secondary = **RU VPN entry only** (not full mirror)
- [x] Legacy `relay` naming removed from **operator CLI** (paths may still say `relay-in` / ExportRelayBundle)
- [x] Edge: enroll → pending → approve; LAN recovery `:7879`; recovery codes; register/set-site/export
- [x] Recovery hardening: private bind, LAN clients, SERVER_PIN, SHA256 self-update
- [x] RU/gov client direct (`ru_direct.go`, SR profile)
- [x] NVR/Tapo **code** MVP (Go port, storage, TG hooks) — hardware e2e open
- [x] TG Access / users / nodes / tools / updates; Admin UI VPN-only
- [x] mTLS material EnsureAll on serve; plain `:8788` only if no certs / PLAIN_AGENT=1
- [x] mTLS-only agent plane `:8789` for secondary **and** edge (no IP allowlist; NAT-friendly)
- [x] Workstation deploy centre (Mac TUI); operator SSH key post-bootstrap
- [x] Admin / TUI / TG EN+RU pass (v0.8.8–0.8.9)
- [x] Cross-VPS backup + COMPONENTS; doctor probes

## Still open (needs operator / hardware / domain)

| Item | Why blocked | Notes |
|------|-------------|--------|
| OpenWrt e2e (Cudy etc.) | No router in agent lab | Agent install, recovery page, enroll, CONTROL_ONLY |
| MikroTik + RPi site | No ROS hardware | Site wizard / RSC already in code |
| Tapo C200 live | No cameras | PTZ/KLAP/RTSP path in code |
| Redirect **HTTPS** / durable domain | No production domain | `NETDUCTOR_REDIRECT_BASE`, LE |
| Admin public TLS | Policy: **not** public | Keep VPN-only |
| Family messenger prod | Optional product | [MESSENGER-EVAL](MESSENGER-EVAL.md) |
| Path B end-user TG bot | Design only | Limited user bot — not started |
| Agent fleet auto-notify “update available” | Partial | Manual `agent_update`; UI list optional polish |
| Integration tests (enroll→approve→HB) | CI time | Unit coverage exists |

## Policy locks

- Admin / API **not** on the open internet without explicit decision
- Agents **not** auto-updated from primary release
- Secondary **not** a full service mirror

## For next agent / chat

1. Hardware: OpenWrt recovery + enroll drill
2. Optional: domain + redirect HTTPS
3. Optional: NVR first camera on site
4. Keep docs EN + `docs/ru/` in sync for any new feature
