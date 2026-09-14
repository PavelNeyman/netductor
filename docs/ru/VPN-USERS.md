# Пользователи VPN

```bash
netductor vpn add alice "laptop"
netductor vpn list
netductor vpn link alice
```

Файлы: `/etc/netductor/clients/<name>/`.


## Reality SNI

Default is **ya.ru** (not Cloudflare). Change:

```bash
netductor vpn set-sni ya.ru
# or: SINGBOX_REALITY_SNI=... at install
# stored in /etc/netductor/secrets/singbox_reality_sni
```

## Reality SNI и белые списки

При белых списках SNI/dest Reality должен быть **доменом из белого списка РФ**. Cloudflare и Microsoft — неверный default.

| Сценарий | Как |
|----------|-----|
| **Домашний интернет** | Зарубежный core + Reality, SNI из БС (по умолчанию **`ya.ru`**; также `vk.com`, `mail.ru`, …) |
| **Мобильный (жёсткий БС)** | Телефон → **промежуточный VPS в РФ** (белый IP) → зарубежный netductor. SNI на входе тоже из БС. |

```bash
netductor vpn set-sni ya.ru
# /etc/netductor/secrets/singbox_reality_sni
```

Автоподнятие RU-relay в netductor пока **нет** — только ручной hop для мобильных, пока не сделаем отдельный plane.



## Переименование

```bash
netductor vpn rename old new
```
UUID не меняется. Fragment в ссылке: `#nd-relay` / `#nd-core` (не имя пользователя).

Subscription: `netductor vpn link NAME sub` или TG/Admin **Subscription**.
