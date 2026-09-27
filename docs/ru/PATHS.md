**RU** · [EN](../PATHS.md)

# Пути (FHS)

| Путь | Назначение |
|------|------------|
| `/usr/local/bin/netductor` | CLI + API ноды |
| `/usr/local/bin/netductor-tg` | Telegram-бот |
| `/usr/local/bin/netductor-agent` | агент edge/secondary |
| `/usr/local/share/netductor/admin` | опциональная статика admin |
| `/usr/local/share/netductor/scripts` | вспомогательные скрипты |
| `/etc/netductor` | конфиг, secrets, sessions |
| `/var/lib/netductor` | state, metrics, edge, profiles, lampac, telegram |

Префикс `/opt/netductor` **не** используется. Канон: [ARCHITECTURE-FREEZE.md](ARCHITECTURE-FREEZE.md).
