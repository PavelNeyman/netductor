# ПЛАН: USB на edge (extroot + fallback + бэкап/restore)

Статус: **в коде** (lab e2e ещё нет; релиз отложен) (решения ниже **согласованы**; в коде ещё нет)  
Связано: `PLAN-EDGE-DEPLOY-HYBRID.md`, модули `overlay_ext` / `vpn_client`, класс TR1200 (16 МиБ flash, USB 2.0).  
Контекст: после v0.9.402.


## Прогресс реализации

| Фаза | Статус |
|-------|--------|
| A Detect + card + vpn gate | **сделано** (facts USB_DISK/BLOCK, Card, plan skip without USB, Web card/modules) |
| B Mount existing ext4 | **сделано** (data mount path in applyEdgeStorage) |
| C Format + extroot | **сделано** (opt-in `usb_format` + overlay_ext → fstab UUID, reboot) |
| D vpn after large overlay | **сделано** (skip vpn same pass when extroot needs reboot) |
| E Pin sync | **сделано** (nd-pin-sync + cron 12h) |
| F Alerts + mute | **частично** (expect_usb flag on device; full TG alerts TBD) |
| G Backup A/B + remote restore | **частично** (`internal/edgestorage` manifest store) |
| H Offline restore | **done** (storage_restore local + pin merge) |


## Цель

1. Детект USB в карточке (объём, FS, занято/свободно, mount, UUID).
2. Только с **явного согласия** — format и **полный extroot**.
3. Internal — тёплый fallback: agent, сеть, Wi‑Fi, firewall; магазин **без VPN**, если USB нет.
4. Периодический **sync** whitelist → pin на internal.
5. **Алерты** + в UI «не ждать USB» (mute).
6. **Бэкапы на primary** + remote и **offline** restore на новую флешку.

Не цель: USB обязателен для AP/agent. **Модем 4G на USB пока не учитываем.**

## Согласованные решения

| # | Решение |
|---|--------|
| D1 | Целевой режим = **полный extroot**; data-mount только аварийный путь. |
| D2 | Порядок: **тонкий стек на internal** → USB/extroot/reboot → **sing-box** → pin sync. |
| D3 | Шаг модулей: **нет USB на probe** → **vpn_client выключен/неактивен** (и format/extroot). |
| D4 | Agent → primary **напрямую** (`:8789` mTLS). SoftFallback + PrimaryHost. |
| D5 | USB out → boot с pin → AP + agent + ISP. |
| D6 | Format/extroot только галочки + warning в той же сессии. |
| D7 | fstab по **UUID**. |
| D8 | Pin: whitelist; без sing-box; безопасная замена (checksum / stage+rename). |
| D9 | LuCI ASU off на internal **до** extroot и в whitelist pin. |
| D10 | Истина при живом USB = **USB overlay**; sync **active → pin**, не наоборот на каждом boot. |
| D11 | Алерты storage + UI «ожидать USB / mute». |
| D12 | Guest/nft/firewall essentials в pin. |
| D13 | Бэкап **A** (sysupgrade-style) + **B** (tar файлов overlay, не образ диска). |
| D14 | Remote restore: format → extroot → B → **merge pin с internal сверху** → алерт → **ручной reboot** из UI. |
| D15 | Offline restore — см. ниже. |

## Что влезает на internal без sing-box

**Да:** agent, mTLS, UCI, guest, ASU off. **Нет:** полный sing-box.

## Порядок деплоя

```text
1) Probe (+ USB) → карточка
2) Модули: нет USB → vpn_client OFF
3) Тонкий стек на internal (без sing-box)
4) Format/extroot по согласию → reboot
5) sing-box / vpn_client на большом overlay
6) Pin sync + бэкап A (+ B)
```

## UI

Карточка storage; галочки Use USB / Format / Extroot / Pin / Expect USB alerts; day-2: backup now, restore from primary, restore from offline pack, mute, manual reboot after restore.

## Бэкапы на primary

| Тип | Содержимое |
|-----|------------|
| **A** | Конфиг OpenWrt (как sysupgrade -b) — и **без** USB |
| **B** | tar.gz **файлов** с extroot (не dd) |

Путь вида `/var/lib/netductor/edge-backups/<device_id>/`, манифест, last+previous, по расписанию и после deploy, mTLS, mode 0600.

### Remote restore (есть путь до primary)

1. UI confirm → agent format → layout → download B → extract.  
2. **Поверх — актуальный internal pin** (старый B не откатывает SERVER/mTLS).  
3. Алерт «можно reboot».  
4. Reboot **вручную** из UI.  
5. Проверка → recovery OK.

### Offline restore (нет сети до primary)

1. **Pack заранее** с доступом к primary (как offline-prep): B (+ опционально A), checksums, привязка к **device_id**.  
2. Доставка: вторая флешка, каталог `nd-restore/`, или с ноутбука по LAN.  
3. «Restore from local pack» → тот же pipeline, merge **internal pin сверху**.  
4. Когда primary снова доступен — залить свежие A/B.

Ограничения: pack с секретами; отказ при несовпадении device_id; лучше готовить pack, пока устройство здорово.

## Pin whitelist (ориентир)

agent, `/etc/netductor-agent/**`, network/wireless/dhcp/firewall/system + guest, ASU off.  
**Не:** sing-box, кэши opkg, большие логи.

Sync: ~12ч + после deploy; не синкать, если на internal мало места.

## Фазы

| Фаза | |
|------|--|
| A | Детект + card + gate vpn без USB |
| B | Mount готового ext4 по UUID |
| C | Format + extroot + reboot |
| D | vpn_client на большом overlay |
| E | Pin + sync |
| F | Алерты + mute |
| G | Backup A/B + remote restore + merge |
| H | Offline pack + local restore |

## Тесты

Без USB → vpn выключен; extroot + sing-box; вынуть USB → AP+agent+алерт; remote restore; offline restore; VPN down → ISP + heartbeat.

## Вне скоупа сейчас

Модем на USB · образы диска · auto-reboot после restore · sing-box на internal

## Мелочи открыты

Порог free для pin · format один раздел vs whole-disk (предпочтительно один раздел)


## Residual (not code)

- Hardware lab e2e on Cudy + USB
- Release cut when operator requests
