# Обновление

## Primary (VPS)

1. **TG → Инструменты → Обновления → Обновить primary** — качает assets с GitHub Releases (`netductor`, `netductor-tg`), пишет `/etc/netductor/VERSION`, restart `netductor-api` + `netductor-telegram-bot`.
2. Или вручную:

```bash
TAG=v0.8.0
curl -fsSL -o /usr/local/bin/netductor \
  "https://github.com/PavelNeyman/netductor/releases/download/${TAG}/netductor-linux-amd64"
chmod 755 /usr/local/bin/netductor
# аналогично netductor-tg → /opt/netductor/bin/
systemctl restart netductor-api netductor-telegram-bot
echo 0.8.0 > /etc/netductor/VERSION
```

## Secondary

`netductor secondary …` / UI Nodes → upgrade (агент secondary тянет команды с primary).

## OpenWrt agents

**Без авто-раскатки.** Оператор: UI/CLI `edge cmd <id> agent_update` с URL бинаря из release.  
Новая версия primary **не** обязана сразу обновлять всех агентов.

## Политика

| Компонент | Источник | Кто решает |
|-----------|----------|------------|
| primary bin + tg | GitHub Release | оператор (TG/CLI) |
| secondary | cmd с primary | оператор |
| edge agent | `agent_update` | оператор, точечно |
| main (dev) | может опережать release | не для prod auto |

Self-update verifies **SHA256SUMS** from the same release (override: `NETDUCTOR_UPDATE_SKIP_VERIFY=1`).
