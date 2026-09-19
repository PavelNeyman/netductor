# Shadowrocket и netductor

## Профили

| Файл | Назначение |
|------|------------|
| VLESS URI из бота | Сервер (предпочтительно **secondary**), `flow=xtls-rprx-vision` |
| `nd-oc.conf` / SR Config | Mac + OpenConnect: LAN/corp + **RU/gov DIRECT** |
| `shadowrocket-routing.conf` | После `vpn client-config`: только RU DIRECT + FINAL PROXY |

## Госуслуги / банки / «Моя школа»

С апреля 2026 сервисы режут **зарубежный exit** и часто **детектят VPN**.

**Что делает профиль:**

1. `DOMAIN-SUFFIX,ru` + gov/bank keywords → **DIRECT** (домашний IP).
2. `GEOIP,RU` → DIRECT.
3. Остальное → **PROXY** (VLESS secondary/core).

**Важно:**

- Global Routing = **Config**.
- Один VLESS с Vision; не мешать внешние balancer/JSON на тот же :443.
- Если приложение всё равно пишет «выключите VPN» — это детект tun (per-app / временно выключить VPN), не лечится другим IP.
- За границей для Госуслуг нужен **RU exit (secondary)**, не home DIRECT.

## Генерация

```bash
netductor vpn client-config operator
# …/sing-box-client.json   — domain_suffix + keyword + geoip-ru → direct
# …/shadowrocket-routing.conf
# …/shadowrocket-uris.txt
```

## Не делать

- Full tunnel без RU rules.
- Второй outbound на тот же host без `flow=xtls-rprx-vision`.
- Резолв RU-доменов только через зарубежный DNS (в sing-box RU → `77.88.8.8`).
