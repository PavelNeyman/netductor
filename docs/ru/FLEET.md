# Fleet: primary и secondary

## Имена

| Термин оператора | Типичный хост | Внутри |
|------------------|---------------|--------|
| **primary** | Зарубежный VPS | Control plane: users, policies, TG-бот, админка, бэкапы. Hostname `nd-primary` |
| **secondary** | RU VPS | Вход VPN (БС) + тёплые сервисы (Lampac). Hostname `nd-secondary`. VPN-агент в коде ещё может писать `role=relay` |

VPN **не** балансируем. Secondary — вход по умолчанию; primary — источник правды.

## Развёртывание secondary с primary

```bash
netductor fleet provision-secondary \
  --host 92.x.x.x --password '…' [--sni api.vk.me]
```

Делает: VPN join → роли fleet → sync → Lampac → bot standby (SOCKS→primary) → hourly sync.

Низкоуровневый только VPN: `netductor relay provision …`
