# MikroTik + RPi OpenWrt (целевая схема)

**На ROS VLESS/Reality не поднимаем.**

## Архитектура
```
[Клиенты LAN] → MikroTik (только маршрутизация / firewall / DHCP)
                      │
                      │ L2/L3 same LAN
                      ▼
              Raspberry Pi (OpenWrt)
                      │
                      │ netductor-agent + VLESS→relay
                      ▼
                   relay (RU) → core
```

## Роли
| Устройство | Делает |
|------------|--------|
| **MikroTik** | LAN, NAT/firewall, policy routing (или static route) на RPi как gateway для нужных dest / mark-routing |
| **RPi OpenWrt** | netductor-agent, VPN client на relay, DNS/blocky по желанию, enroll/approve как обычный edge |
| **relay** | RU direct / non-RU uplink |
| **core** | control plane |

## MikroTik (минимум)
1. Отдельный interface-list или routing table для «через RPi».
2. Маршрут default или policy: mark connection → route to RPi IP.
3. Heartbeat/API с ROS **не обязателен**, если управление только RPi.
4. RSC-генератор в репо остаётся опциональным (identity/note), не VPN.

## OpenWrt на RPi
Стандартный edge: `netductor edge …` / agent, primary VLESS на relay, WAN fallback при недоступности relay.

## Честно
- Без тестов на железе MT+RPi в этом релизе.
- VLESS на RouterOS не целевой путь.
