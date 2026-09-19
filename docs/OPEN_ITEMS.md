# Open items

## Done recently (through 0.7.34-dev)

- [x] TG Access rich QR + URI + body/nav split
- [x] Import redirect on :80 + TG url buttons (SR/Happ/INCY)
- [x] Doctor: redirect + healthz; alerts secondary/sing-box
- [x] mTLS client on secondary provision
- [x] RU/gov **client + SR** direct (Gosuslugi/banks/geoip-ru) — `ru_direct.go`, `nd-oc.conf`
- [x] Secondary = VPN entry only (not full mirror)
- [x] NVR/Tapo Go control MVP (code); review in NVR-CODE-REVIEW.md
- [x] SecondaryDir path helper + less hardcode IP in TUI; clientcfg tests; release workflow_dispatch
- [x] Review 2026-09-19: [REVIEW-2026-09-19.md](REVIEW-2026-09-19.md)

## Still open

- [ ] Redirect **HTTPS** / durable domain (`NETDUCTOR_REDIRECT_BASE`)
- [ ] Admin UI: **not** public (locked)
- [ ] OpenWrt + MikroTik + Tapo **e2e on real hardware**
- [ ] Release assets always tracking `main` (process)
- [x] Prefer `secondary/` state (legacy `relay/` still readable)
- [ ] Path B: limited end-user bot (design only)

## Notes for agents

**RU-direct:** home ISP for `category-ru`/gov; secondary for WL entry and abroad RU-IP needs. App-level “disable VPN” = tun detection — not fixed by more routes.

**Admin TLS:** only if binding changes off localhost.

**Hardcoded base:** set env after reinstall; do not rely on compiled test IP.

## Family messenger

See [MESSENGER-EVAL.md](MESSENGER-EVAL.md). Snikket primary candidate; Matrix fallback.

## NVR

See [PLAN-NVR-TAPO.md](PLAN-NVR-TAPO.md) — hardware validation remaining.
