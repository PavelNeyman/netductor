# Edge-агент (OpenWrt)

**RU** · [EN](../EDGE-AGENT.md)

## Роль
Агент на роутере: enroll, heartbeat к primary (mTLS), применение desired state, guest Wi-Fi при включении, recovery-код на объекте.

## Деплой
Только с Mac (TUI/Web → DeployEdge), не отдельный «флот через TG».

Мастер: SSH/IP, pubkey Mac, параметры WAN (dhcp/static/pppoe), LAN, SSID 2.4/5 (пустые поля = как у заполненной сети), DHCP pool.

## Сеть
Роутер за NAT провайдера — allowlist по белому IP не подходит; связь agent→primary по mTLS исходящая.

## После деплоя
Password SSH выключается, остаётся ключ Mac. Взаимодействие day-2 — agent plane, не mesh SSH между устройствами.
