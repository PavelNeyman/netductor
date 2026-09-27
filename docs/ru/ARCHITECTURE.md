# Архитектура (обзор)

**RU** · [EN](../ARCHITECTURE.md)

Кратко: self-hosted control plane.

- **Primary** — API, bot, DNS (blocky), бэкапы, mTLS-сервер, домен/redirect
- **Secondary** — вход VPN для пользователей, agent heartbeat на primary :8789
- **Edge (OpenWrt)** — агент на объекте, enroll/recovery
- **Mac op** — деплой флота, TUI/Web installer, credentials

Канон и запреты: [ARCHITECTURE-FREEZE](../ARCHITECTURE-FREEZE.md).
