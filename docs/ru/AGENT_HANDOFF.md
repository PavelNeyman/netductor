# Передача контекста агенту (читать первым в новом чате)

**Репо:** https://github.com/PavelNeyman/netductor  
**Релиз:** `v0.7.0-dev`  
Документация user-facing — **EN + RU**.

Полная версия (EN): [AGENT_HANDOFF.md](../AGENT_HANDOFF.md)

## Модель

| Имя | Где | Зачем |
|-----|-----|--------|
| **primary** | За рубежом | Источник правды, TG active, API, бэкапы. `nd-primary` |
| **secondary** | РФ | Вход VPN (БС), Lampac, standby TG через SOCKS. `nd-secondary` |

VPN **не** балансируем. В коде VPN-агент может называться `relay`.

## SSH

Пароль только для первого входа → `install` / `provision-secondary` ставят ключ primary и **отключают пароль**.

## Деплой

1. Primary: binary → secrets TG → `netductor install` → `vpn set-sni api.vk.me`  
2. Secondary с primary: `netductor fleet provision-secondary --host IP --password …`  
3. Recover: `netductor recover --key … backup.ndenc`

**Не** удалять токены агента в post-provision до heartbeat.

## Telegram UI

- **Под сообщением** — только навигация (меню, назад).  
- **В тексте** — действия экрана (доступ, rename, VLESS/HY2, метрики ноды).  
Без дублирования «Добавить» / «Меню».

## TUI

`netductor tui` → мастер настройки (primary/secondary/OpenWrt/MikroTik) + Tools.
