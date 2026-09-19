# Roadmap

## Done
- Go-only control plane (install, VPN, API, TUI, TG, edge)
- Fleet **primary / secondary** (`internal/fleet`, `provision-secondary`)
- SSH key-only after first login (primary install + secondary provision)
- Cross-VPS backup + `recover` + COMPONENTS manifest
- Secondary agent; preferred client links on RU IP; RU/direct routing on secondary
- TG bot standby (SOCKS→primary) + `/api/bot-status` on `:8788`
- TG UI: nav under message, actions in body (no duplicate Add/Menu)
- TUI Setup wizard + Tools
- Node registry UUID + `nd-primary` / `nd-secondary` hostnames

## Next (owner)
- [ ] OpenWrt + MikroTik e2e on real hardware
- [ ] Optional HTTPS / domain
- [ ] Optional Path B limited user-facing TG bot

## Tests
- `go test ./...` on change
- Dual-VPS smoke: install primary → provision-secondary → link @RU → doctor

## Handoff
[AGENT_HANDOFF.md](AGENT_HANDOFF.md)

## 2026-09-19
- Closed: client RU/gov direct for Gosuslugi stack.
- See OPEN_ITEMS.md + REVIEW-2026-09-19.md.
