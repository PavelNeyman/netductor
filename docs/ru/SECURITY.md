**RU** · [EN](../SECURITY.md)

# Безопасность

## Поза продукта (freeze)

- API ноды только **127.0.0.1** (без флага «открыть»)
- Agent plane только **mTLS :8789** (plain :8788 в коде нет)
- Rate-limit без доверия к XFF
- UI оператора — **netductor-op** на Mac
- Redirect только **:8443** после LE
- Recovery: arm по SSH, ключ бэкапа только offline
- Edge: только per-device token после approve
- SSH **52222**, key-only

## Conf

`ndconfig.Load` **не загружает** удалённые ключи в env (даже если остались в старом conf). Включить их через conf/env **нельзя**.

## Backup / LE

Серты LE не в архиве — перевыпуск при recover при `DOMAIN` + `LE_EMAIL`.

См. [ARCHITECTURE-FREEZE.md](ARCHITECTURE-FREEZE.md).

## SSH host keys (TOFU)

- CLI: `netductor ssh-hosts list|forget|clear`
- Password deploy Secondary **не** сбрасывает доверие к host key по умолчанию (v0.9.490+). При смене ключа — отказ; явный forget и один контролируемый retry. Политика: [SECONDARY-DEPLOY.md](SECONDARY-DEPLOY.md) §1.

