# Fleet: primary и secondary

См. также: [AGENT_HANDOFF.md](AGENT_HANDOFF.md) · [BACKUP.md](BACKUP.md) · [DEPLOY.md](DEPLOY.md)

## Имена

| Термин | Хост | Смысл |
|--------|------|--------|
| **primary** | За рубежом | Control plane, TG active, бэкапы. `nd-primary` |
| **secondary** | РФ | Вход VPN (БС), Lampac, standby TG. `nd-secondary` (в коде VPN — `relay`) |

VPN не балансируем.

## Деплой secondary

```bash
netductor fleet provision-secondary --host IP --password '…' [--sni api.vk.me]
```

## TG failover

Active на primary; standby на secondary через SOCKS→primary, только если primary **жив**, а unit бота упал.
