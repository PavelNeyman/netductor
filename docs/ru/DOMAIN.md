# Домен и TLS (только явные хосты)

Все публичные имена задаёт **оператор**. Автоподстановки `p.` / `s.` / `i.` из base **нет**.

## Для production

| Роль | Флаг / поле | Пример |
|------|-------------|--------|
| CORE / primary | `--primary` / `domain_primary` | `p2.nd.example.com` |
| VPN entry | `--vpn` / `domain_vpn` | `s.nd.example.com` |
| Import / redirect | `--redirect` / `domain_redirect` | `https://i2.nd.example.com:8443` |
| Метка (опц.) | `--base` / `domain_base` | `nd.example.com` → только `DOMAIN=` в conf |
| Email LE | `--le --email` / `le_email` | вместе с primary (+ redirect для второго имени) |

```bash
netductor domain set \
  --primary p2.nd.example.com \
  --vpn s.nd.example.com \
  --redirect https://i2.nd.example.com:8443 \
  --base nd.example.com \
  --le --email admin@example.com
```

Deploy (Mac):

```bash
netductor-op deploy primary … \
  --domain-primary p2.nd.example.com \
  --domain-vpn s.nd.example.com \
  --domain-redirect https://i2.nd.example.com:8443 \
  --domain-base nd.example.com \
  --le-email admin@example.com
```

## Порты

| Порт | Сервис | Сертификат |
|------|--------|------------|
| **443** | sing-box Reality | Маскировка SNI — **не** LE |
| **8443** | `netductor-redirect` | **Let's Encrypt** |
| **80** | **выкл.** после LE | Для `certbot renew` |
| **52222** | SSH | только ключ |
| **8789** | agent mTLS | внутренний CA |

## Let's Encrypt

- Только **явные** имена (`--domains` или `domain set --primary` + host из `--redirect`).
- Один **`--base` серты не выпускает** и имена не придумывает.
- Другие имена (`p2`/`i2` вместо `p`/`i`) обходят лимит duplicate certificate на том же apex.

## Cloudflare (опционально)

Orange только на hostname **redirect**. CORE и VPN — **DNS only**, иначе ломается Reality на 443.
