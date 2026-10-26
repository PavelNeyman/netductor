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

## Runbook миграции (v0.9.539)

Раздельные Reality-профили. DNS не переключает профиль. `prepared` не значит traffic-ready. Promote не автоматический.

1. Provision B (`Promote=false`). SSH-порт primary — `PrimarySSHPort` (по умолчанию 52222).
2. Проверки инфраструктуры: units, ping, свежий heartbeat, непустые PBK/SID, SingBoxOK, UplinkOK.
3. Внешний клиентский трафик на публичный IP B профилем B. Это не часть успешного deploy.
4. `peer ready` и `peer promote` только после этого теста трафика.
5. Пользователи выбирают профиль B из `/sub/{token}`. Пока оба свежие, в списке оба.
6. Необязательная ручная смена DNS общего hostname. Это не замена профиля B.
7. Окно наблюдения. A остаётся для rollback.
8. Удалять A только после приёмки: остановить сервисы A, `secondary remove <agent-id>` (убирает A из `/sub`), затем `peer remove <peer-id>`.
9. Rollback до конца окна: DNS обратно на A, promote A. Не удалять A первым.

`peer add` на removed id падает. Нужен новый id или recover. Пустой SID/PBK не попадает в `/sub`.

## 0.9.545

- На B `peer validate` читает локальный `material.json`. Id необязателен. На primary `peer validate <id>` читает `peers/<id>`.
- `peer remove` также снимает UFW-правила peer и каталог `peers/<id>`. Сервисы на самом старом Secondary он не останавливает.
- `ready` — только инфраструктура. Клиентский Reality-трафик он не доказывает. Promote только после этого теста.
- Shadowrocket helper сохраняет оба профиля, если hostname общий, а Reality-ключи разные.
