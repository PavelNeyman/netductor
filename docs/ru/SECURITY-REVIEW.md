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
