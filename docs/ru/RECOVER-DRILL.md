# Drill recover (primary со secondary)

**RU** · [EN](../RECOVER-DRILL.md)

**Статус:** live unattended drill **пройден** 2026-09-27 на **v0.9.71**.

**Цель:** бэкап → wipe primary → recover без потери данных и **без ручных доработок**.

## Условия
- Secondary online; актуальный `.ndenc` в `peers/core/`
- `recovery_token` на secondary; у оператора есть `backup_key`
- Ключ Mac `~/.ssh/netductor_primary`
- Бинарь ноды с GitHub Release (**не** из бэкапа)

## Шаги

1. **Бэкап** на primary: `netductor backup now` — в `COMPONENTS.txt` baseline + optional.
2. Убедиться, что архив на secondary.
3. **Снести** primary; запомнить root-пароль.
4. На чистом primary поставить `netductor` с Release.
5. На secondary: `netductor recovery arm --ttl 2h`.
6. `netductor recover --from-secondary https://SECONDARY:8790 --recovery-token … --key …` (+ `NETDUCTOR_OPERATOR_PUBKEY`).
7. **Проверка без ручного install:** doctor fail=0, api/sing-box/blocky/bot active, `netductor-tg` — файл ELF, ufw :8789 для IP secondary, secondary online, SSH `:52222`.
8. Снять arm recovery на secondary.

## Как устроен recover (0.9.71+)

Сначала install по COMPONENTS (vpn-users может soft-fail без секретов → continuing), затем tar, затем **второй проход** api/telegram/backup/vpn-users, затем ufw secondary.

Ожидаемый хвост лога: `(continuing)` → `post-restore component pass` → `recover: done`.
