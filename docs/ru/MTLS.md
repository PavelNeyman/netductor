**RU** · [EN](../MTLS.md)

# mTLS (:8789)

- Agent plane **только TLS 1.3 + client cert**
- Plain **:8788** в продукте **нет** (`PLAIN_AGENT` → refuse)
- WAN: deny, кроме service CIDR SP/PS + `api-allow.cidr`
- **Travel edge (ISP NAT):** durable `netductor api-public permanent on` → flag `/etc/netductor/edge-plane-public`, UFW allow :8789.
  Security is **not** source-IP allowlist (addresses change). App-layer DMZ:
  bootstrap token → enroll → **pending** only; commands / heartbeat-auth need **approve** + device token + client cert.
- Временный WAN (legacy): `netductor api-public arm` / TG (TTL) — не нужен при permanent on
- Оператор к API: SSH tunnel или VPN, не постоянный public bind
