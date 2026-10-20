# NVR implementation (план)

## Контур

- go2rtc / segments path `/var/lib/netductor/nvr/`
- API session: cameras, live, PTZ, recorder, clip tokens
- Doctor / alerts при проблемах storage

## Безопасность

- Clip token: redeem только путей **внутри** NVR root (path traversal closed)
- Live stream: write deadline снят для длинных стримов (см. api_nvr_live)

## UI

Day-2 group `media` — Web / TG / TUI через opcatalog. См. `PLAN-NVR-UI.md`, `DAY2-GROUPS.md`.

## Storage

`nvr prepare-storage`, retention, mount hints в doctor. Запись по умолчанию может быть выключена.

## Открытое

Two-way audio / Tapo deep integration — research; e2e на железе отдельно.
