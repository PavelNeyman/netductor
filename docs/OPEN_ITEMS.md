# Open items

## Done (recent)
- [x] TUI: Site wizard + MikroTik manage + Live SNI
- [x] sites push CLI/API, line-by-line RSC
- [x] Live SNI set-sni / Admin
- [x] Client configs (Shadowrocket/sing-box)
- [x] Admin Sites/Status parity
- [x] Flow mismatch counters (CLI / TG / API)
- [x] SSH TOFU known_hosts (MT + relay) + UI: CLI, TUI, Admin, Telegram
- [x] Edge API: device token bound to device_id; session-only devices/cmd
- [x] TG: SSH hosts under Nodes

## Remaining (hardware / domain / policy)
- [ ] Site wizard on real MT + RPi
- [ ] HTTPS / Mini App (needs domain)
- [ ] Live SNI auto under carrier WL (policy)

## Commands (current CLI)
```bash
netductor install | tui | doctor | serve
netductor vpn list|add|link|mismatch|session|set-sni
netductor ssh-hosts list|forget|clear
netductor relay | nodes | sites | edge
```
