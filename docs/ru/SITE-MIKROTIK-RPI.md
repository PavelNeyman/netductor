# Сайт: MikroTik + RPi OpenWrt

## Одна нода или две?
**Site** (логическая площадка) = 2 устройства:
- RPi OpenWrt — edge agent (как Cudy), VPN
- MikroTik — только routing, RSC

## RPi vs Cudy
Тот же OpenWrt + netductor-agent. На RPi чаще без Wi‑Fi AP, один LAN к MT.

## MikroTik
Policy route на IP RPi. RSC из Admin Sites / API `/api/sites/rsc`.
