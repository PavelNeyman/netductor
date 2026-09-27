# Runbook: install / recover

**RU** · [EN](../RUNBOOK-INSTALL-RECOVER.md)

Идемпотентный install/update; recover с secondary :8790 + token + backup key.

С **0.9.71:** pre-restore install не обрывается на vpn-users; post-restore pass ставит api/telegram/backup; ufw пускает IP secondary на :8789; tg — реальный бинарь.

Критерий успеха: doctor fail=0 без ручного scp/install/ufw.  
Подробный drill: [RECOVER-DRILL](RECOVER-DRILL.md).
