# Roadmap

## Done
- Go-only control plane (install, VPN, API, TUI, TG, edge)
- Fleet **primary / secondary** model (`internal/fleet`, `provision-secondary`)
- Secondary = RU VPN entry + warm services; primary = control plane
- Cross-VPS backup + `recover` with COMPONENTS manifest
- Relay/secondary agent, preferred client links, RU exit toggle
- TG bot standby design (SOCKS via primary, `/api/bot-status`)
- Node registry UUID + hostnames (`nd-primary`, `nd-secondary`)
- Site model (MikroTik + RPi), RSC push, live SNI
- Production auth: session hash, device tokens, TG allowlist

## Next (owner action)
- [ ] Clean dual-VPS smoke: wipe primary + secondary, full provision-secondary path
- [ ] OpenWrt + MikroTik e2e on real hardware
- [ ] Optional HTTPS / domain
- [ ] TG UI copy: consistent primary/secondary wording everywhere

## Tests
- `go test ./...` on every change
- VPS clean install + secondary provision when hosts available

## Handoff
- [AGENT_HANDOFF.md](AGENT_HANDOFF.md) for new chat/agent context
