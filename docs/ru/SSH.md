# SSH-доступ к нодам netductor

## Ключ флота

После provision core генерирует (или использует) ed25519-ключ:

- private: `/root/.ssh/id_ed25519` на **core**
- public: `/root/.ssh/id_ed25519.pub`
- тот же public ставится в `authorized_keys` на **relay** и edge при enroll

Парольный вход на relay после enroll обычно **отключается**. На core может оставаться включённым (зависит от hardening).

## Как подключиться

```bash
chmod 600 netductor_vps_id_ed25519
ssh -i netductor_vps_id_ed25519 root@2.27.118.70    # core
ssh -i netductor_vps_id_ed25519 root@92.255.77.253  # relay
```

Ключ оператора храните локально; в репозиторий **не** коммитить.

## Восстановление доступа

1. С консоли провайдера: вписать свой pubkey в `/root/.ssh/authorized_keys`
2. Или временно включить `PasswordAuthentication yes` и `systemctl reload ssh`
