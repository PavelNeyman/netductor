# Бэкапы

**RU** · [EN](../BACKUP.md)

- Шифрованные архивы `.ndenc` + `backup_key`
- `netductor backup now` — локально; pull на secondary (peer/agent)
- **COMPONENTS.txt** — baseline primary + optional (lampac/git/registry)
- Бинарники **не** из бэкапа — только с GitHub Release
- Recover: [RECOVER-DRILL](RECOVER-DRILL.md), [RUNBOOK-INSTALL-RECOVER](../RUNBOOK-INSTALL-RECOVER.md)

Ключ хранить offline; secondary recovery API отдаёт архив по Bearer-токену, ключ — у оператора.
