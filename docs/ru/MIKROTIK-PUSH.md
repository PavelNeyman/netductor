# Push RSC на MikroTik (ваши креды)

## Важно: откуда SSH

Команда должна выполняться **с машины, которая видит MikroTik по сети**:

| Откуда | Когда работает |
|--------|----------------|
| **Ваш Mac/ПК в домашней LAN** | Да — `192.168.x.x` доступен |
| **Через VPN в дом** | Да, если маршрут до MT есть |
| **Core VPS в NL** | **Нет**, если MT только в частной сети без проброса 22 |

Креды **не храним** на сервере: передаёте один раз в команде/API.

## CLI (предпочтительно с Mac в LAN)

```bash
# бинарь под macOS из releases, или go run на Mac
netductor sites add --id home --name Home --mt mt-home

netductor sites push \
  --id home \
  --host 192.168.88.1 \
  --user admin \
  --password 'ВАШ_ПАРОЛЬ' \
  --rpi-lan 192.168.88.2 \
  --port 22
```

Что уйдёт на MT: identity, note, таблица `via-rpi`, default route на IP RPi.

## Через API core (только если core достаёт до MT)

```bash
# session token
TOK=$(netductor vpn session 24 | head -1)

curl -sS -X POST http://127.0.0.1:8787/api/sites/push-rsc \
  -H "Authorization: Bearer $TOK" \
  -H "Content-Type: application/json" \
  -d '{
    "site_id": "home",
    "host": "192.168.88.1",
    "user": "admin",
    "password": "ВАШ_ПАРОЛЬ",
    "port": 22,
    "rpi_lan": "192.168.88.2"
  }'
```

## Перед push

1. RPi уже с OpenWrt и IP `--rpi-lan`.
2. На MT включён SSH (`/ip service print` → ssh).
3. User/password или позже ключ.

## После push

```bash
netductor sites add --id home --name Home --rpi <device_id_после_approve> --mt mt-home
```

Проверка: с LAN `traceroute 1.1.1.1` должен идти через RPi, если policy/route via-rpi активен для этого трафика.
