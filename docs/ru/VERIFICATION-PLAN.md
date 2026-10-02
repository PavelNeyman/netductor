# Netductor — поэтапный план проверки

**Назначение:** системный разбор сценариев и модулей с учётом реального железа.  
**Прогресс:** отмечать `[x]` только после выполнения; не пропускать отметки.  
**Stop line на момент создания:** v0.9.191 (2026-10-02).  
**Правило:** EN ↔ RU — **полная смысловая** паритетность (AGENTS §7), не «краткий пересказ».

Связанные: [AGENT_HANDOFF.md](../AGENT_HANDOFF.md) · [OPEN_ITEMS.md](../OPEN_ITEMS.md) · [AGENTS.md](../../AGENTS.md) · EN: [VERIFICATION-PLAN.md](../VERIFICATION-PLAN.md)

---

## 0. Как пользоваться файлом

1. Идти **по одной фазе** (P0/A → B → …).  
2. Для пункта: **ревью кода** и/или **живой тест** — как указано.  
3. Ставить `[x]` только когда сделано; кратко писать в **журнал прогресса**.  
4. Заблокированное железом оставлять `[ ]` с пометкой `blocked: hardware`.  
5. Новые баги → фикс + релиз; тег указывать в журнале.

---

## 1. Матрица железа и окружения

| ID | Устройство / среда | Роль | Ограничения (нельзя игнорировать) |
|----|-------------------|------|-----------------------------------|
| **H-P** | Primary VPS (за рубежом) | Control plane, API, TG, LE, git/CI/registry, опционально Lampac | После harden SSH **52222** только ключ; mTLS **:8789** по умолчанию не с WAN; лимиты LE |
| **H-S** | Secondary VPS (РФ) | Вход VPN (VLESS Reality), agent, peer backup_pull | Тонкая нода — **не** полное зеркало; agent → primary :8789 по SP; лимит RAM/CPU |
| **H-Mac** | Mac оператора | `netductor-op` TUI/Web/CLI деплой | Formula brew; ключи `~/.ssh`; креды `~/.netductor/` |
| **H-OW** | OpenWrt (Cudy и т.п.) | Edge agent, guest Wi‑Fi, опционально LAN VPN | **mipsle** softfloat; **Dropbear** (нет SFTP); **~16 МБ flash / ~128 МБ RAM**; ash ≠ bash; на части сборок нет `base64`; factory SSH **:22** пароль, затем harden |
| **H-Cam** | Tapo C200 | RTSP / NVR | Только LAN; API Tapo (не «универсальный ONVIF»); Wi‑Fi |
| **H-MT** | MikroTik (+ опционально RPi OpenWrt) | Маршрутизация площадки | Не везде контейнеры ROS; RSC/скрипты + agent на RPi; не паритет с OpenWrt |
| **H-Phone** | iOS (SR / Happ / INCY) | Клиенты VPN | БС/DPI; вход предпочтительно **secondary**; кнопки url нужен redirect base |
| **H-Corp** | Mac/iPhone + корп OpenConnect | Сплит с домашним VPN | Private через OC; интернет через наш туннель — чувствителен клиентский профиль |

**Живой инвентарь (ведёт оператор):** заполнять после каждого wipe.

| Хост | IP / имя | Заметки |
|------|----------|---------|
| Primary | `2.27.118.70` / `p*.nd.neyman.top` | |
| Secondary | `92.255.77.253` / `s*.nd.neyman.top` | |
| Redirect | `i*.nd.neyman.top:8443` | LE; CF без proxy |
| OpenWrt | (ожидает) | Cudy mipsle — деплой edge прерван на 0.9.190; echo/ash исправлен в 0.9.191 |

---

## 2. Сценарии использования (сквозные)

У каждого сценария — **этапы**. Модули сверяются с этими этапами.

### S1 — Чистый флот с Mac

| Этап | Действие | Критерий успеха |
|------|----------|-----------------|
| S1.1 | Установка/обновление `netductor-op` (brew) | `version` = релиз |
| S1.2 | Деплой **primary** (пароль → ключ, harden 52222) | doctor ok; API; unit’ы active |
| S1.3 | Домен + LE (`p`/`i`) | сертификат; redirect `/healthz` |
| S1.4 | TG-бот + admin id | `/menu`; язык RU/EN |
| S1.5 | Деплой **secondary** + svc-paths SP/PS | agent online; VLESS; peer ping |
| S1.6 | Опциональные addons (git, CI, registry, Lampac) | только выбранные; без тихой установки |
| S1.7 | Креды на Mac | `~/.netductor/credentials/*` |

**UI:** Web Fleet, TUI-мастер, CLI `deploy primary|secondary`.

### S2 — Day-2 без переустановки

| Этап | Действие | Успех |
|------|----------|-------|
| S2.1 | Status / fleet health | версии primary/secondary правдивы |
| S2.2 | VPN users: add / rename / Access QR | QR и ссылки; peer’ы `edge-*` **скрыты** из Users |
| S2.3 | Stack/update apply (только ручное подтверждение) | нет цикла авто-отката; local/prev согласованы |
| S2.4 | Backup + pull на secondary | возраст OK; артефакт на peer |
| S2.5 | API public arm (TG) | временный :8789; auto disarm |
| S2.6 | DNS block lists / tools | применение без ложных doctor alarm |

### S3 — Аварийное восстановление

| Этап | Действие | Успех |
|------|----------|-------|
| S3.1 | Wipe primary | на secondary есть backup_pull |
| S3.2 | `recover --from-secondary` (или путь с Mac) | компоненты по списку; данные восстановлены |
| S3.3 | Перевыпуск LE (серты не в `.ndenc`) | redirect поднят |
| S3.4 | Heartbeat secondary agent | online; VPN работает |
| S3.5 | Ключ оператора | SSH без тупика по паролю |

### S4 — OpenWrt edge first-boot (в LAN)

| Этап | Действие | Успех |
|------|----------|-------|
| S4.1 | Factory OpenWrt; Mac в той же сети | SSH :22 пароль (пустой или известный) |
| S4.2 | Probe arch → agent (mipsle/…) | верный asset; кэш ok |
| S4.3 | Provision agent + mTLS **без** harden | бинарь и material на роутере |
| S4.4 | Network UCI staged (без reload) | скрипт безопасен для ash; только commit |
| S4.5 | Guest `--stage` | UCI+nft; без reload посреди деплоя |
| S4.6 | Harden (pubkey, SSH password off) | пароль LuCI = New root (harden не трогает) |
| S4.7 | Reboot; новый LAN IP при смене | enroll → ожидание approve |
| S4.8 | Approve; apply template (VPN/DNS) | private Wi‑Fi через secondary VLESS; soft fallback WAN |
| S4.9 | Guest flow | captive/desk; TTL; QR WIFI |

**Риски железа:** Dropbear без SFTP; ash; размер flash; guest только radio0; half-state при обрыве.

### S5 — OpenWrt day-2 / offline

| Этап | Действие | Успех |
|------|----------|-------|
| S5.1 | LuCI on/off/extend (agent + SSH в LAN) | TTL auto-off; опционально TG |
| S5.2 | agent_update с релиза primary | версия совпала; без полного wipe |
| S5.3 | Recovery token на LAN | enroll без перепрошивки |
| S5.4 | Offline pack / bootstrap-token | установка без WAN на роутере |

### S6 — Гостевой Wi‑Fi (коммерческий паттерн)

| Этап | Действие | Успех |
|------|----------|-------|
| S6.1 | Guest SSID (по умолчанию видимый / опция hidden) | изоляция; без forward в LAN |
| S6.2 | Клиент в сети → captive | интернет только после grant |
| S6.3 | Staff desk PIN / QR (дефолт ~10 мин, макс 24 ч) | nft MAC allow |
| S6.4 | Истечение TTL | блок; повторный grant возможен |

### S7 — NVR / камеры

| Этап | Действие | Успех |
|------|----------|-------|
| S7.1 | Leases на edge | найти Tapo MAC/IP |
| S7.2 | Static lease + путь на primary | RTSP по задуманной схеме |
| S7.3 | Запись сегментов + ротация | диск не забивается бесконечно |
| S7.4 | UI список/скачивание (Web/TG) | доступ через VPN |
| S7.5 | Motion / PTZ / night (порт Tapo) | hardware e2e |

### S8 — Площадка MikroTik + RPi

| Этап | Действие | Успех |
|------|----------|-------|
| S8.1 | RPi OpenWrt agent как edge | та же модель enroll |
| S8.2 | Скрипты маршрутизации MT | без неподдерживаемых контейнеров |
| S8.3 | Инвентарь site/location | устройства сгруппированы |

### S9 — Клиентский VPN UX

| Этап | Действие | Успех |
|------|----------|-------|
| S9.1 | Вход по умолчанию **secondary** | VLESS Reality дома и в LTE при доступности |
| S9.2 | Primary / sub для оператора | выход с РФ из-за границы (выбранные пользователи) |
| S9.3 | Профиль SR + OpenConnect | корп LAN + интернет |
| S9.4 | Redirect url-кнопки | https base; i.* без orange CF |

### S10 — Тонкий git / CI / registry (опционально)

| Этап | Действие | Успех |
|------|----------|-------|
| S10.1 | Установка только по выбору | нет тихой установки во Fleet |
| S10.2 | Список репо / pipeline | UI оператора |
| S10.3 | Registry push/pull | ограниченный auth |

---

## 3. Инвентарь модулей → сценарии

| Модуль / плоскость | Путь (ориентир) | Сценарии | Фокус ревью |
|--------------------|-----------------|----------|-------------|
| **op** deploy | `internal/deploy`, `cmd/netductor-op` | S1, S4–S5 | порядок SSH, ash, Dropbear, abort vs warn |
| **op** Web/TUI | `internal/operator` | S1–S2, S4 | паритет, prefill, без тихих addons |
| **node** install | `internal/install` | S1, S3 | COMPONENTS, harden, LE |
| **node** API | `cmd/netductor`, edge plane | S2–S5 | mTLS, rate limit, arm |
| **tg** | `cmd/netductor-tg` | S2, S9 | шаблоны A/B/C, правдивые версии, топики |
| **agent** | `cmd/netductor-agent` | S4–S7 | guest stage, soft VPN fallback, luci TTL |
| **edgeagent** UCI | `internal/edgeagent` | S4 | DesiredUCI, ShellApplyStaged для ash |
| **edge** templates | `internal/edge` | S4.8 | peer `edge-*`, dns vpn, fallback |
| **vpn** users | `internal/vpn` | S2, S9 | скрытие edge peer; Reality |
| **secondary** / svcpaths | `internal/secondary`, `svcpaths` | S1.5, S3 | SP/PS только служебка |
| **stack** / update | `internal/stack`, `update` | S2.3 | **только ручной** apply/rollback |
| **backup** | пути backup | S2.4, S3 | по COMPONENTS; без бинарей в архиве |
| **guest** | agent + nft | S6 | TTL grant |
| **nvr** / tapo | `internal/nvr`, `tapo` | S7 | специфика C200 |
| **mikrotik** | `internal/mikrotik` | S8 | генерация RSC |
| **dnsblock** | blocky | S2.6 | UI списков |
| **tlsle** / domain | `internal/tlsle`, `domain` | S1.3, S3.3 | перевыпуск после recover |
| **git/ci/registry** | optional | S10 | нагрузка на primary |

---

## 4. Фазы работ (строго по порядку)

### Фаза A — Документ и инвентарь

- [x] A.1 EN-план со сценариями и матрицей  
- [ ] A.2 RU-двойник (этот файл) — отметить после вычитки паритета  
- [x] A.3 Ссылки из handoff + OPEN_ITEMS  
- [ ] A.4 Заполнить живой инвентарь после следующего wipe  

### Фаза B — Ревью кода по модулям (без железа)

Один блок модулей за сессию; заметки — в журнал прогресса.

- [ ] B.1 `internal/deploy` + `edgeagent` (путь OpenWrt 0.9.186–191)  
- [ ] B.2 `cmd/netductor-agent` guest + luci + vpn client  
- [ ] B.3 `internal/edge` templates / peers / enroll  
- [ ] B.4 Primary install + harden + LE  
- [ ] B.5 Secondary + svc-paths + backup_pull  
- [ ] B.6 Stack/update (без самовольной смены версии)  
- [ ] B.7 TG навигация + отображение версий + шаблоны карточек  
- [ ] B.8 Паритет op Web/TUI vs opcatalog  
- [ ] B.9 NVR/tapo/mikrotik vs заявленный UX  
- [ ] B.10 Security: порты, mTLS, recovery, токены  

### Фаза C — Живой smoke dual-VPS (без OpenWrt)

- [ ] C.1 Primary doctor + unit’ы  
- [ ] C.2 Secondary agent + VLESS  
- [ ] C.3 SP/PS health  
- [ ] C.4 TG menu + один VPN user QR  
- [ ] C.5 Backup + list  
- [ ] C.6 Ручной stack status vs файл `VERSION`  

### Фаза D — OpenWrt e2e (Cudy / mipsle)

- [ ] D.1 Factory + деплой ≥0.9.191 полный S4  
- [ ] D.2 Enroll approve + template VPN  
- [ ] D.3 Guest grant 10 мин  
- [ ] D.4 LuCI 1 ч / off / SSH LAN  
- [ ] D.5 agent_update  
- [ ] D.6 Multi-radio guest при необходимости (дыра в коде)  

### Фаза E — Клиенты и сплит

- [ ] E.1 Телефон secondary VLESS  
- [ ] E.2 Primary / sub политика оператора  
- [ ] E.3 SR + OpenConnect  

### Фаза F — NVR / site (когда есть железо)

- [ ] F.1 Tapo discover + запись  
- [ ] F.2 MT+RPi site  

### Фаза G — DR-учение

- [ ] G.1 Backup на secondary  
- [ ] G.2 Recover primary  
- [ ] G.3 LE + agent  

---

## 5. Известные пробелы

| Пробел | Влияние | Фаза |
|--------|---------|------|
| Guest AP только **radio0** | SSID может не подняться | D / B.2 |
| Нет авто-SSH на новый LAN IP | Оператор переподключается | D |
| LE не в backup | certbot после recover | G |
| Half-state OpenWrt при обрыве | Лучше factory и повтор | D |
| UI версий раньше врал (stack prev) | Доверять `netductor version` + файл | C.6 / B.6 |
| Hardware e2e открыт | OPEN_ITEMS | D–F |

---

## 6. Журнал прогресса

| Дата | Пункт | Результат |
|------|-------|-----------|
| 2026-10-02 | A.1 | Создан EN-план; stop **0.9.191**; ash-paren network stage исправлен |
| 2026-10-02 | A.2 | Создан RU-план (смысловой паритет) |
| | | Живой тест OpenWrt отложен (роутер возвращён) |

---

## 7. Чеклист сессии для агентов

1. Прочитать stop line в handoff.  
2. Взять **один** незакрытый пункт Phase B/C/….  
3. Ревью или тест; строка в журнале.  
4. Отметить `[x]`; при необходимости синхронизировать EN.  
5. Пункты с железом **не** закрывать без факта на устройстве.
