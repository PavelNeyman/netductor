# Paths (FHS)

| Path | Role |
|------|------|
| `/usr/local/bin/netductor` | node CLI + API |
| `/usr/local/bin/netductor-tg` | Telegram bot |
| `/usr/local/bin/netductor-agent` | edge/secondary agent |
| `/usr/local/share/netductor/admin` | optional static admin assets |
| `/usr/local/share/netductor/scripts` | helper scripts (tapo, …) |
| `/etc/netductor` | config, secrets, sessions |
| `/var/lib/netductor` | state, metrics, edge, profiles, lampac data, telegram runtime |

Legacy `/opt/netductor` is no longer the install prefix. On upgrade, `paths.MigrateFromOpt()` moves data into the table above.
