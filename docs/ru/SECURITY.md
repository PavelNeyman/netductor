**RU** · [EN](../SECURITY.md)

# Безопасность

- Node API только loopback; non-local bind **запрещён**
- :8789 mTLS; WAN только arm TTL или CIDR
- Нет plain :8788, TRUST_PROXY, permanent ALLOW_PUBLIC
- Redirect только :8443 TLS; root 404
- Admin UI только Mac op
- Backup encryption key offline; recovery arm SSH
- Secrets в `/etc/netductor/secrets` mode 700

Freeze: [ARCHITECTURE-FREEZE.md](ARCHITECTURE-FREEZE.md).
