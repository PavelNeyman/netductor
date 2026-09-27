# Порты

**RU** · [EN](../PORTS.md)

| Порт | Где | Назначение |
|------|-----|------------|
| 52222/tcp | primary, secondary | SSH key-only |
| 443/tcp | secondary (и primary sing-box) | VLESS+Reality пользователи |
| 4443/tcp | secondary | exit-in (внутренний) |
| 8787/tcp | primary | Node API, localhost |
| 8789/tcp | primary | Agent mTLS |
| 8443/tcp | primary | Redirect HTTPS |
| 53 | primary | Blocky, localhost |
| 8790/tcp | secondary | Recovery API только при `recovery arm` |

**Не использовать:** публичный plain :8788, постоянный публичный :80.
