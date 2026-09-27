**RU** · [EN](../MTLS.md)

# mTLS (:8789)

- Agent plane **только TLS 1.3 + client cert**
- Plain **:8788** в продукте **нет** (`PLAIN_AGENT` → refuse)
- WAN: deny, кроме service CIDR SP/PS + `api-allow.cidr`
- Временный WAN: только `netductor api-public arm` / кнопка TG (TTL)
- Оператор к API: SSH tunnel или VPN, не постоянный public bind
