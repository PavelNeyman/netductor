# Безопасность

- API по умолчанию 127.0.0.1; сессии в `/etc/netductor/sessions`.
- Секреты: `/etc/netductor/secrets`.
- Бэкапы: AES-256-CBC, ключ `backup_key`.
- Edge-агент: token; только allowlist команд.

## Модель аутентификации (production)

См. английскую версию docs/SECURITY.md — session hash-at-rest, edge device tokens, rate-limit enroll, TG admin pre-provisioned.


## SSH TOFU (MikroTik / relay)
- Первый коннект сохраняет host key
- Управление: `netductor ssh-hosts`, TUI, Admin, Telegram → Ноды → SSH hosts
- `NETDUCTOR_MT_STRICT=1` — неизвестные MT-хосты отклоняются
- Подробнее: [TOFU.md](TOFU.md)
