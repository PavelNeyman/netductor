# Ревью безопасности (продолжение)

## Исправлено
- Edge: device token привязан к `device_id` (heartbeat, commands, metrics, backup)
- Upload backup ограничен 50 MiB
- MaxBytesReader 64 MiB на API
- Security headers: nosniff, DENY frame, no-referrer

## Осознанные ограничения
- MikroTik SSH: `InsecureIgnoreHostKey` (LAN one-shot)
- `/health` без auth (liveness only)
- API по умолчанию localhost; public bind — осознанный `api-bind`

## Рекомендации оператору
- Не открывать :8787 в интернет без VPN/TLS
- Bootstrap token ≥ 32 символов, только enroll
- После push MT сменить пароль / ключ


## TG bot
- Admin только из `telegram_admin_id` или `NETDUCTOR_TG_ADMIN`
- `NETDUCTOR_TG_CLAIM_FIRST=1` — единственный способ claim через /start
- Callback: `From.ID == admin`

## Relay agent plane (:8788)
- Token ≥ 16, constant-time compare
- Heartbeat body ≤ 1 MiB
- Config export uses ActiveSNI; no new agent token on pull
