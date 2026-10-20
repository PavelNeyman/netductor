# OpenWrt lab (эмуляция / железо)

## Цель

Проверять edge deploy (probe → card → modules → provision) без сюрпризов production Cudy.

## Что уже учитывает агент / deploy

| Факт | Поведение |
|--|--|
| Один eth (SBC) vs lan+wan (travel) | preset / capabilities |
| WAN | `wan` когда uplink есть |
| Wireless | `wifi-device` radio* на реальном hw; в чистой VM часто **нет** |
| Flash/arch | real: mipsle/arm; lab VM: amd64/arm64 agent |

Agent: SSH с пустым/заданным паролем, put через stdin (без SFTP), radio через UCI (не только Cudy `radio0`).

## Топология lab — как ПК в LAN, не «роутер как ноутбук»

Неверно:

```text
Mac NAT → OpenWrt forced DHCP client on lan
```

Верно: stock OpenWrt VM/router с LAN, Mac на том же L2, SSH на `192.168.1.1` (или staged `10.120.x.1`).

## Прогон

1. Offline-prep (при offline deploy).
2. Probe → карточка.
3. Modules (usb/overlay если накопитель виден).
4. Provision; после reboot — маршрут на новый LAN.

## USB / flash

Мало места на Cudy → sing-box/USB packages; probe может ставить kmod из кэша Mac (нужен интернет на Mac при miss). См. PLAN storage/USB в репе.

## Ссылки

`docs/UI-BUTTON-AUDIT.md`, edge hybrid / storage plans, handoff.
