# Runbook: второй egress (multi-egress)

**Цель:** два и более входа VPN; при падении одного клиенты сохраняют рабочий профиль.  
**Не автоматически:** **promote** peer, DNS A для `vpn_hostname` и вывод старого узла — шаги оператора.

## Добавить B

1. Deploy egress B с **Promote=false**. Region, напр. `ru-msk`.
2. B в `prepared`, heartbeat online, `singbox_ok`.
3. В `/sub` есть A и B. Для теста: `netductor secondary prefer <IP_B>`.
4. Обновить sub; проверить только профиль B.
5. `svc-paths peer ready` / validate по слоту.
6. После успешного трафика: `netductor svc-paths peer promote <peer-id-B>`.
7. DNS A `VPN_HOST` → B **после** promote, если клиенты ходят по имени.
8. A не гасить, пока sub и пользователи стабильны.

| Сигнал | Эффект |
|--------|--------|
| Старый heartbeat | Нет в `/sub` |
| `singbox_ok=false` | Нет в `/sub` |
| `prefer` | Только порядок |
| `peer promote` | Active в service plane |

Авто-promote **не** делаем до R5-гейтов (план флота, Phase 2).
