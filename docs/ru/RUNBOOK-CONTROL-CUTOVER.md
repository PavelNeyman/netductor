# Runbook — cutover control (active → standby)

**Область:** Phase 3 active/standby. Не multi-master. Авто-promote нет.

## Условия

| Проверка | Действие |
|----------|----------|
| Роль standby | `control_role=standby` на целевом узле |
| Свежий inbox | недавний `control sync-pull` (или timer) |
| Ключ backup offline | есть `backup_key` |
| Агенты | в `SERVERS=` есть URL standby |
| DNS | понятно, какие имена указывают на API/redirect |

## Шаги

1. Убедиться, что старый active мёртв или изолирован (два writer недопустимы).
2. На **standby**:
   ```bash
   netductor control restore-inbox --key "$BACKUP_KEY" --confirm
   netductor control sot-check
   netductor control promote --confirm
   ```
3. **Fence** старого active: `set-role standby --confirm` или выключение.
4. **DNS / клиенты** — имена control/API на новый IP; LE при необходимости перевыпустить. Egress VPN обычно не трогаем.
5. **Агенты** — `SERVER`/`SERVERS`; при залипании — рестарт agent.
6. **Проверка:** `doctor`, `secondary status`, `GET /api/control/status`.

## Откат

Новый узел в standby; старый `promote` только если его SoT ещё авторитетен; вернуть DNS.

## Не делаем

Авто-promote, два живых writer, смена VLESS egress только из‑за переноса control.
