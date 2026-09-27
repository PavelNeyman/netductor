# Безопасность

**RU** · [EN](../SECURITY.md)

- SSH key-only :52222, fail2ban
- API не на весь мир; agent mTLS + allowlist/CIDR
- Нет постоянных env-footgun’ов
- Backup encrypted; private keys оператора на Mac
- Redirect только TLS :8443

См. также doctor checks и ARCHITECTURE-FREEZE.
