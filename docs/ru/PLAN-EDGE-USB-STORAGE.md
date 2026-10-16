# ПЛАН: USB-накопитель на edge (extroot + fallback на внутреннюю flash)

Статус: **запланировано** (ещё не в коде)  
Связано: `PLAN-EDGE-DEPLOY-HYBRID.md`, модули `overlay_ext` / `vpn_client`, класс Cudy TR1200 (16 МиБ flash, USB 2.0).  
Контекст: после v0.9.402 (проверка места для vpn_client + опциональный data-mount под sing-box).

## Цель

На OpenWrt магазина/travel с маленькой внутренней flash:

1. Детектировать USB и показывать его в карточке устройства.
2. Только с **явного согласия** оператора — подготовить USB (опционально format) и использовать как **полный extroot** (`/overlay` на USB).
3. Внутренняя flash — **тёплый fallback**: agent, сеть, Wi‑Fi, firewall — магазин работает **без VPN**, если USB нет или он мёртв.
4. **Периодическая синхронизация** критичных файлов с USB-overlay → internal overlay, чтобы fallback был актуальным.
5. **Алерты** при пропаже/болезни USB; health накопителя.

Не цель: требовать USB для базового LAN/AP/agent.

## Принципы продукта (не ломать)

| Принцип | Суть |
|---------|------|
| **Agent → primary напрямую** | `SERVER=https://<primary>:8789` (mTLS). **Не** зависит от LAN VLESS/sing-box. В клиентском JSON — `SoftFallback` и direct на `PrimaryHost`. |
| **Падение VPN ≠ отключение магазина** | Нет sing-box/VLESS → egress LAN через **ISP WAN**. Интернет у сотрудников остаётся; пропадает только путь «через netductor». |
| **Падение USB ≠ кирпич** | Boot без USB → internal overlay; AP + DHCP + agent работают. Нет: тяжёлых пакетов с extroot (**sing-box**). |
| **Деструктивные операции — только opt-in** | Format / разметка / pivot extroot — только галочки в UI + предупреждение. |

### Путь agent сейчас (проверка в коде)

- Provision пишет `SERVER=https://primary:8789`, не через пользовательский VLESS.
- Генератор VLESS-клиента: `SoftFallback` + **direct** на `PrimaryHost`, чтобы control plane не уходил в туннель.
- Регресс-тест: VPN down или USB вынут — heartbeat agent до primary при живом WAN.

## Железо (класс TR1200)

| | Обычно |
|--|--|
| Internal flash | 16 МиБ SPI NOR; overlay часто ~8–12 МиБ free |
| RAM | ~128 МиБ |
| USB | 1× USB 2.0 (питание ~2.5 W — модемам часто нужен powered hub) |
| Встроенный microSD | **Нет** на TR1200 |
| Что **влезает без sing-box** | Agent, mTLS, UCI (network/wireless/dhcp/firewall), guest — **да**. **Полный sing-box — нет**. |

## Архитектура

```text
                    ┌──────────────────────────────┐
                    │ USB (ext4, UUID в fstab)      │
                    │  = /overlay (extroot)         │
                    │  sing-box, опциональные pkg   │
                    └──────────────▲─────────────────┘
                                   │ pivot при boot
┌──────────────────────────────────┴──────────────────┐
│ Boot: ROM + попытка extroot                         │
│ USB OK  → большой overlay (возможен VPN)            │
│ USB fail → internal overlay (AP + agent + ISP)      │
└─────────────────────────────────────────────────────┘
         периодический sync (whitelist) → pin на internal
```

### Решение: полный extroot

В этом плане целевой режим при наличии USB и согласии — **весь /overlay на USB**.

- Data-mount только под sing-box — запасной путь (частично уже есть в vpn_client), если оператор **не** включает format/extroot.
- Рекомендуемый путь: **extroot**.

## UI / поток оператора

### Карточка устройства (Web, op; не deploy из TG)

- Internal: overlay total/free.
- USB: есть/нет; устройство; размер; разделы; FS; mount; free/used; UUID.
- Health: last seen, mount ok, ошибки I/O при наличии.

### Галочки

| Контрол | Смысл |
|---------|--------|
| **Use USB for storage** | Мастер-включение |
| **Format USB (destroy data)** | mkfs.ext4 — **обязательное подтверждение** |
| **Extroot (перенос /overlay на USB)** | Pivot + reboot |
| **Pin fallback to internal** | Sync whitelist internal←USB (по умолчанию **вкл** при extroot) |

### Порядок модулей

1. Probe (в т.ч. USB).
2. Format/extroot при выборе.
3. Agent + UCI (работают с internal).
4. Pivot + reboot.
5. На большом overlay — **sing-box** / vpn_client.
6. Таймер sync.

Нет USB при выбранном extroot → **жёсткая ошибка**, без «успешного» reboot вслепую.

## Фазы реализации

| Фаза | Содержание |
|------|------------|
| **A** | Детект + поля карточки + RU/EN, без деструктива |
| **B** | Mount уже готового ext4 по UUID |
| **C** | Opt-in format + extroot (копия overlay, fstab, reboot) |
| **D** | vpn_client на большом overlay; SoftFallback + PrimaryHost |
| **E** | Pin + sync whitelist (agent, mTLS, UCI сетей; **не** sing-box) |
| **F** | Алерты missing/unhealthy/extroot failed + recovery ✅ |

## Алерты

| Событие | |
|---------|--|
| USB ожидался, нет после boot | ⚠️ storage missing |
| Extroot настроен, не смонтирован | 🔴 extroot failed |
| RO / I/O errors | ⚠️ storage unhealthy |
| Восстановление | ✅ по тому же паттерну, что channel/path |

## Безопасность

- Format только с явного согласия в той же сессии.
- В fstab — UUID, не `/dev/sda1`.
- Приватные ключи оператора на edge не хранить.
- TG — только алерты, не deploy.

## Тесты (лаб)

1. Без USB — базовый deploy; vpn без места — явный fail/skip.  
2. USB + format + extroot — после reboot overlay на USB, sing-box, soft-fallback.  
3. Вынуть USB — reboot — AP + agent; алерт; без sing-box.  
4. Вернуть USB — extroot снова; recovery-алерт.  
5. Убить VLESS — ISP у LAN; agent бьёт primary.

## Вне скоупа (пока)

- RAID, multi-disk.
- Модем+storage на одном stick как основной дизайн.
- sing-box на internal flash.
- Тихий format в фоне.

## Открытые продуктовые вопросы

1. Интервал sync (предложение: 12ч + после успешного deploy).  
2. Format: только раздел vs весь диск (предложение: один раздел; whole-disk только если диск «пустой»).  
3. Мин. free на internal после pin — иначе отказ sync.
'''
