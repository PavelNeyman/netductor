# Открытые темы

## Сделано (недавнее)
- [x] TUI: Site wizard + MikroTik manage + Live SNI
- [x] sites push CLI/API, line-by-line RSC
- [x] Live SNI set-sni / Admin
- [x] Client configs (Shadowrocket/sing-box)
- [x] Admin Sites/Status parity
- [x] Flow mismatch counters (CLI / TG / API)
- [x] SSH TOFU known_hosts (MT + relay) + UI: CLI, TUI, Admin, Telegram
- [x] Edge API: device token bound to device_id; session-only devices/cmd
- [x] TG: SSH hosts under Nodes

## Осталось (нужно железо / домен / политика)
- [ ] Прогон Site wizard на реальном MT + RPi
- [ ] HTTPS / Mini App (нужен домен)
- [ ] Live SNI auto under carrier WL (policy)

## Команды (актуальный CLI)
```bash
netductor install | tui | doctor | serve
netductor vpn list|add|link|mismatch|session|set-sni
netductor ssh-hosts list|forget|clear
netductor relay | nodes | sites | edge
```
