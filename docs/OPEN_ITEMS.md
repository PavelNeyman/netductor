# Open items

Baseline: **v0.8.13** (PKI revoke/rotate, cert expiry doctor, plane ban).

## Done (software)

- [x] Primary/secondary, edge recovery, RU-direct, NVR code MVP
- [x] mTLS :8789, rate-limit + temporary IP ban
- [x] Client cert **list / revoke / rotate**; doctor expiry WARN ≤30d
- [x] Self-update SHA256; recovery LAN bind
- [x] Mac TUI deploy wizards; version pins CI

## Still open (hardware / optional product)

| Item | Notes |
|------|--------|
| OpenWrt / MikroTik / Tapo **e2e** | Real devices |
| Domain + redirect HTTPS | Optional |
| Admin public | **Not** doing |
| Path B user bot | Design only |
| CA rollover dual-trust | Long-term |
| Auto-push cert after rotate | Use re-provision for now |

## Policy

Admin/API not WAN; agents no auto-rollout; secondary = VPN entry only.
