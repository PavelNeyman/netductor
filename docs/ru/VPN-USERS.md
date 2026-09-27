**RU** · [EN](../VPN-USERS.md)

# Пользователи VPN

- Реестр: `/etc/netductor/clients` + state registry
- `netductor vpn list|add|rename|…` и UI (TG / op)
- Служебный пользователь **`relay-uplink`**: UUID secondary→primary, **без** vision (совместим с multiplex)
- После recover обязателен `vpn ensure-relay-uplink` + `vpn apply` (см. RECOVER-DRILL)
