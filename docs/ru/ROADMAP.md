# Roadmap

## Сделано
- Go-only control plane (install, VPN, API, TUI, TG, edge, relay)
- Auth: session, device tokens, enroll rate-limit, TG admin allowlist
- Реестр нод (UUID + hostname `nd-<role>-…`)
- Relay RU (provision, agent, preferred links, RU exit)
- Flow mismatch; SSH TOFU + управление в CLI/TUI/Admin/TG
- Sites (MikroTik + RPi), RSC push, live SNI
- `netductor update`; audit log; hardened install

## Дальше (нужны железо / домен)
- e2e OpenWrt + MikroTik на реальных устройствах
- HTTPS / Mini App
- Опционально: авто-SNI при БС оператора

## Тесты
- `go test ./...` на каждое изменение
- Smoke на чистой VPS при доступе
