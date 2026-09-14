# TOFU (Trust On First Use) для SSH

## Идея
Как у SSH при первом подключении к серверу:
1. **Первый** успешный коннект — публичный host key **запоминается**
2. **Следующие** — ключ **должен совпасть**
3. Если ключ **другой** — отказ (возможен MITM или переустановка ОС)

Это лучше, чем `InsecureIgnoreHostKey` (принимать любой ключ каждый раз), и проще, чем раздавать CA/сертификаты.

## Где у netductor

| Цель | Файл | Поведение |
|------|------|-----------|
| **MikroTik** (sites push, TUI manage) | `StateDir/mikrotik/known_hosts.json` | ключ `host:port` → base64(key) |
| **Relay provision** (SSH при enroll RU VPS) | `StateDir/relay/ssh_known_hosts.json` | ключ = hostname/IP |

Обычно StateDir = `/var/lib/netductor`.

## MikroTik

```text
1) netductor sites push … или TUI → MikroTik manage
2) SSH handshake → host key ещё нет в JSON → сохраняем (TOFU)
3) Повторный push → ключ сверяется
4) Переустановили RouterOS / сменился ключ → ошибка host key mismatch
   → удалить запись в known_hosts.json (или весь файл) и подключиться снова
```

Строгий режим (не писать новые ключи):

```bash
export NETDUCTOR_MT_STRICT=1
# неизвестный хост → отказ, пока ключ не добавлен вручную
```

Пароли/ключи пользователя **не** кладём в known_hosts — только fingerprint host key сервера.

## Relay provision

При `netductor relay enroll` / provision с core на новый VPS:
- первый SSH сохраняет ключ RU-ноды
- повторный provision на тот же IP с другим ключом — ошибка

## Ограничения TOFU
- Первый коннект всё ещё уязвим к MITM (как и OpenSSH без known_hosts)
- Делайте первый push **из доверенной сети** (домашний LAN к MT)
- После компромисса/переустановки — чистите JSON осознанно
