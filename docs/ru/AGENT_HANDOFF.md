# Передача контекста агенту (читать первым в новом чате)

**Репо:** https://github.com/PavelNeyman/netductor  
**Релиз:** `v0.7.0-dev`  
Документация user-facing — **EN + RU**.

## Модель fleet

| Имя | Где | Зачем |
|-----|-----|--------|
| **primary** | За рубежом | Источник правды, TG-бот active, API, бэкапы. `nd-primary` |
| **secondary** | РФ | Вход VPN (БС), Lampac, standby TG через SOCKS→primary. `nd-secondary` |

VPN **не** балансируем. В коде VPN-агент может называться `relay`.

## Чистый деплой

1. Primary: binary → secrets TG → `netductor install` → doctor  
2. Secondary с primary: `netductor fleet provision-secondary --host IP --password …`  
3. Recover: `netductor recover --key … backup.ndenc` (+ `COMPONENTS.txt` рядом)

SNI по умолчанию для экспериментов с БС: **`api.vk.me`**.

Подробности и cheat sheet: [AGENT_HANDOFF.md](../AGENT_HANDOFF.md) (EN, полный).
