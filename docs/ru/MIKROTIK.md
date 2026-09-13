# MikroTik (без железа — каркас)

## Модель
Одинаковый enroll с OpenWrt: устройство само ходит на core, pending → approve.

1. **RSC bootstrap** — identity + scheduler heartbeat (`/tool fetch` → `/api/edge/heartbeat`).
2. **Команды** — очередь на core, ROS script забирает JSON по расписанию (в разработке).
3. **VPN** — нативный VLESS+Reality на ROS пока слабый; временный вариант: WG/L2TP до relay или внешний пакет. WAN fallback: interface-list `nd-vpn` off при недоступности relay:443.
4. **Контейнеры** — не требуем (не все платы поддерживают).

## Генерация RSC
```bash
netductor edge mikrotik-rsc --name mt-office --relay 92.255.77.253 --core https://CORE
```
Или Admin → Edge → MikroTik RSC.

## Ограничения (честно)
- Нет прогона на железе в этом релизе.
- Reality-клиент ROS не паритетен sing-box; production VPN на MT — через relay WG или ждать зрелости ROS.
