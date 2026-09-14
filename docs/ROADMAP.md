# Roadmap

## Done
- Go-only control plane (install, VPN, API, TUI, TG, edge, relay)
- Production auth: session hash-at-rest, device tokens, enroll rate-limit, TG admin allowlist
- Node registry with stable UUID + hostname roles (`nd-<role>-…`)
- Relay RU node (provision, agent, preferred links, RU exit toggle)
- Flow mismatch monitoring; SSH TOFU known_hosts + multi-UI manage
- Site model (MikroTik + RPi), RSC push, live SNI
- `netductor update`; audit log; APT hardened install

## Next (needs owner hardware/domain)
- OpenWrt + MikroTik e2e on real devices
- HTTPS / Mini App
- Optional: SNI auto policy under carrier WL

## Tests
- `go test ./...` on every change
- VPS clean install smoke when host available
