# Бэкапы

**RU** · [EN](../BACKUP.md)

## Что входит
- Конфиг `/etc/netductor/` (включая secrets, conf, hostname files)
- State `/var/lib/netductor/` (users, secondary registry, git data по возможности)
- Шифрование AES → файл `.ndenc`
- Рядом с архивом: `COMPONENTS.txt` (что ставить при recover)
- Бинарники **не** кладутся в бэкап — всегда GitHub Release

## Команды
```bash
netductor backup now          # локальный .ndenc
netductor backup list
# peer: secondary agent тянет архив в peers/core/
```

Ключ: `backup_key` в secrets или `NETDUCTOR_BACKUP_KEY`. Хранить offline на Mac.

## COMPONENTS
Автоманифест: optional (lampac/git/registry) + **baseline** primary (dirs…telegram…backup).  
Sparse list не должен убирать core при recover (merge DefaultComponents с 0.9.70+).

## Offsite / secondary
Secondary держит копии peers; recovery API `:8790` (arm TTL) отдаёт latest по Bearer `recovery_token`.  
На проводе ключ шифрования не обязателен — он у оператора.

## Recover
См. [RECOVER-DRILL](RECOVER-DRILL.md) и [RUNBOOK-INSTALL-RECOVER](RUNBOOK-INSTALL-RECOVER.md).

**Важно:** каталог Let's Encrypt (`/etc/letsencrypt`) обычно **не** в бэкапе. После wipe primary перевыпустить:
`netductor tls le --email … --domains i.nd.neyman.top,p.nd.neyman.top`.

## Credentials на Mac
`netductor credentials collect` — локальная копия метаданных/секретов для restore.
