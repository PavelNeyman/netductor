# Проходы очистки и security-ревью (2026-10)

**Для кого:** любой ИИ или человек, продолжающий netductor.  
**Стоп-линия на момент записи:** коммиты tip **после v0.9.403** (`git log v0.9.403..HEAD`).  
**Не** заменяет [REVIEW-SUITE.md](../REVIEW-SUITE.md) — это **журнал гигиены** и **бэклог**.

Связано: [AGENT_HANDOFF.md](../AGENT_HANDOFF.md) · [OPEN_ITEMS.md](../OPEN_ITEMS.md) · [REVIEW-SUITE.md](../REVIEW-SUITE.md) · [EN](../CLEANUP-PASSES-2026-10.md)

---

## 1. Зачем были проходы

После фич (USB на edge, build-host, мониторинг каналов, SSH allowlist, policy, SR) оператор попросил:

1. Ревью **всего** проекта, не только последнего диффа.  
2. Убрать **неиспользуемые меню TG** и **мёртвый код** (только при нуле ссылок).  
3. Ужесточить **allowlist команд** agent/secondary/build-host и session gate API.  
4. Зафиксировать итог, чтобы следующий агент не ходил по кругу.

Автотесты на живых VPS по запросу оператора иногда **пропускались**; unit-тесты пакетов в sandbox по возможности гонялись.

---

## 2. Что удалили / почистили (мёртвый код)

| Область | Действие | Заметки |
|---------|----------|---------|
| **TG** | Старое меню release; `m:release` → Versions | Кнопка обновления — в **Versions**, не в старое primary-only |
| **TG** | Неиспользуемые keyboard/handler | Перед удалением: callbacks ≤64 байт, forceHub `/menu` |
| **op** | Лишние wizard/SSH/`guest_qr` | Продуктовый UI — Web op |
| **пакеты** | `internal/singboxconfig` | Конфиг VPN в `internal/vpn` |
| **agent / fleet / notify / CLI** | Мёртвые хелперы | Коммиты `5b8f9ee`, `7dbf82a`, `489232c` |
| **build-host** | SSH primary→B | Только agent plane |

**Правило:** символ мёртв только при нуле ссылок (кроме тестов/стабильного API). Лучше явная ошибка-stub (`SetBackupPeer`), чем тихий no-op.

---

## 3. Security / correctness из той же кампании

| Тема | Проблема | Исправление |
|------|----------|-------------|
| **Edge cmds** | `apply_rsc` в allowlist | Убран; явная ошибка на агенте |
| **Secondary cmds** | Enqueue без allowlist | `allowedSecondaryCmd` + тесты |
| **Build-host cmds** | Свободный enqueue | Только `backup_pull`, `recovery_arm` |
| **Session gate :8787** | Token-маршруты agent резались gate | Exempt только token-path; operator — session |
| **Stack apply** | Пустой version → ошибка | Пусто → latest, затем `ValidReleaseTag` |
| **Smoke** | Любой mode | Только `dual\|secondary\|primary\|local` |
| **dnsblock** | Custom URL не включался | id или `https://`; без http/localhost/private/credentials |
| **Recovery client** | Любая схема URL на :8790 | Только **https** |
| **Ship extract** | `cd 'path'` в shell | `cmd.Dir = work` |
| **Formula** | Старый `netductor.rb` | Только `netductor-op.rb` |
| **Op version** | Захардкоженная версия | Pin на линию релиза |

Уже было нормально: mTLS `ClientDir`, recovery constant-time, NVR `PathUnderRoot`, openssl pass через **env**, SR work в skip-proxy.

---

## 4. Коммиты (серия после v0.9.403)

См. `git log v0.9.403..HEAD`. Последний **продуктовый** релиз на GitHub: **v0.9.403**. Tip может быть **впереди** релиза, пока оператор не попросит **0.9.404+**.

---

## 5. Что осталось следующему ИИ

### 5.1 Перед «tip = prod release»

1. Собрать **релиз** со всеми tip-коммитами (ассеты, SHA256SUMS, Formula, VERSION).  
2. Прогнать **REVIEW-SUITE** (включая `go test`, если оператор разрешит).  
3. Обновить стоп-линию **HANDOFF / OPEN_ITEMS** (EN+RU).

### 5.2 Продукт / железо (не закрыто чисткой)

| Пункт | Заметки |
|-------|---------|
| **OpenWrt / Cudy e2e** | План USB в дереве; железо — у оператора |
| **Edge VLESS** | Day-2 после enroll/template |
| **Build-host e2e** | Код есть; без хоста — не слать ложные path-down |
| **Dual-node smoke на проде** | Чеклист оператора |
| **MikroTik / NVR Tapo** | Планы, не фокус этой чистки |

### 5.3 Опциональный бэклог

- Дожать R1 secondary builders.  
- UI для custom DNS lists (бэкенд готов).  
- Pin CA вместо `InsecureSkipVerify` на recovery.  
- IPv6 HA IdP — **не сейчас**.

### 5.4 Как продолжать ревью

1. Этот файл + `git log v0.9.403..HEAD`.  
2. Дальше при необходимости: `notify`, Web static op, `openwrt/`.  
3. Мёртвый код — только с доказательством.  
4. API: `:8787` vs `:8789`, тест exempt.  
5. Документы **EN+RU** с одним смыслом.

---

## 6. Apply у оператора (когда будет тег)

```bash
netductor stack apply v0.9.404
netductor secondary cmd <id> upgrade:v0.9.404
brew upgrade netductor-op
```

---

*Обновляйте таблицу §3 при следующем проходе.*
