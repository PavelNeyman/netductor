# Безопасность

**RU** · [EN](../SECURITY.md)

## Сеть
- SSH **52222**, только publickey, fail2ban  
- Node API **:8787** localhost (доступ с Mac через tunnel)  
- Agent **:8789** mTLS; CIDR/api-allow; без постоянного «открыть мир»  
- Redirect только **HTTPS :8443**  
- Нет публичного :80 в steady-state  

## Секреты
Private SSH на Mac. Backup encrypted. TG token/admin в secrets.  
Убраны постоянные footgun env: API_ALLOW_PUBLIC / PLAIN_AGENT / …

## Doctor
Проверяет passwordauth, порты, mTLS certs, bot, redirect base.  
WARN redirect без LE — ожидаемо до `tls le`.
