# Blue-Green Secondary (приёмка R5)

Базовая версия кода: **v0.9.499+**. Полный текст: [../BLUE-GREEN-SECONDARY.md](../BLUE-GREEN-SECONDARY.md).

Кратко: `inventory/register` → `peer add` (слот) → bootstrap B → `validate` → `ready` → тест по IP B → `promote` → DNS → окно наблюдения → `rollback` / `remove`.

**Promote ≠ DNS. Register ≠ deploy.**

## Профили подписки (v0.9.535)

`GET /sub/{token}` остаётся живым URL. `sub_profile=secondary` отдаёт URI всех свежих Secondary, не только preferred.

- Общий hostname из `NETDUCTOR_VPN_HOST` или `/etc/netductor/vpn_hostname`.
- У каждого URI свои `pbk`/`sid`/`sni` и уникальный `#nd-secondary-<ip>`.
- Устройства без heartbeat или старше 3 минут не попадают в список.
- `primary` — только core. `both` — все Secondary плюс core.
- DNS переключается вручную. `peer promote` DNS не меняет.

Два разных удаления:

| Команда | Что убирает |
|--|--|
| `netductor secondary remove <agent-id>` | VPN-устройство из `secondary/devices.json` — это убирает A из `/sub` |
| `netductor svc-paths peer remove <peer-id>` | Service-plane peer/слот на primary. Само по себе URI из подписки не удаляет |

Не удалять A, пока трафик B и DNS cutover не подтверждены. `peer remove` отказывает active peer и больше не ставит `removed`, если очистка unit/файлов не прошла.
