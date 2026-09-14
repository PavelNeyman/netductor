# Белые списки (БС) и netductor

## Модель (по исследованию openlibrecommunity / Habr)

1. **L3**: если IP **не** в белом списке — пакет **drop** (не доходит до Reality).
2. **L7**: если IP в списке, ТСПУ смотрит **SNI** в ClientHello; «опасные» SNI → RST.
3. **UDP** (QUIC/WG/DNS) на мобильных часто мёртв; рабочий путь — **TCP 443**.

Источники данных: [twl](https://github.com/openlibrecommunity/twl), [rewl](https://github.com/openlibrecommunity/rewl), API `https://wly.zarazaex.xyz/check?ip=…`.

## Что делает netductor

| Компонент | Поведение |
|-----------|-----------|
| Default SNI | `api.vk.me` (переопределение: `set-sni` / пресеты) |
| Fingerprint | `firefox` |
| Flow | всегда `xtls-rprx-vision` |
| Ссылка / QR | **сначала relay**, если online |
| Client JSON | dual-hop `core-via-relay` (detour), IPv4, block UDP/443 и IPv6, DNS split |
| `vpn wl-check [ip]` | проверка IP через публичный WL API |

## Что нельзя обойти конфигом

Если **IP вашей RU-ноды не в L3 whitelist** оператора — с LTE до неё **нет пути**. Нужен VPS/IP из проходимого пула (Timeweb/Yandex/VK Cloud и т.п., reroll IP, сверка с twl/rewl).

## Рекомендуемые SNI

`api.vk.me`, `api.vk.com`, `userapi.com`, `ya.ru`, `yastatic.net`, `storage.yandex.net`, `okcdn.ru`, `id.x5.ru` — см. `netductor vpn set-sni` и пресеты.

Избегать как SNI: `twitter.com` / `x.com` / очевидный blacklist.
