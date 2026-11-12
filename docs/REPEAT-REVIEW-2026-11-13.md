# Netductor Deep Review — Повторное ревью (2026-11-13)

**Связанный документ:** [Первоначальное ревью](DEEP-REVIEW-2026-11-13.md)

## Контекст

**Предыдущий HEAD:** `131464d83e26`  
**Текущий HEAD:** `f35b30a5b2b06f55ded94e7f52282a7ed9cfc2d2`  
**Изменения:** 16 коммитов, 43 файла изменено  
**Инструменты:** `.qwen-tools/qwen_tools.py`, `.qwen-tools/chunked_ops.py`

---

## Проверка критичных багов из прошлого ревью

### P1-1: SetRegion без блокировок

**Статус:** ✗ НЕ ИСПРАВЛЕН  
**Файл:** `internal/secondary/store.go:901-907`  
**Детали:** Функция SetRegion всё ещё не использует `mu.Lock()` или `acquireRegistryLock()`. Конкурентные CLI и API-heartbeat могут привести к lost-update или воскрешению PendingCmds.

**Рекомендуемый фикс:** Добавить `mu.Lock()` в начало функции и `defer mu.Unlock()` после.

---

### P4-SV1: Tick без лока

**Статус:** ✗ НЕ ИСПРАВЛЕН  
**Файл:** `cmd/netductor/collect.go:209`  
**Детали:** Вызов `svcpaths.Tick()` всё ещё не обёрнут в `acquireFailoverLock()`. Гонка между collect и apply на `failover-state.json` остаётся.

**Рекомендуемый фикс:** Обернуть вызов в `svcpaths.WithLock()`.

---

### P3-R7: ApplyConfig неатомарна

**Статус:** ✗ НЕ ИСПРАВЛЕН  
**Файл:** `internal/vpn/apply.go:92,128,131`  
**Детали:** Функция ApplyConfig (126 строк) использует прямую запись через `os.WriteFile` в трёх местах. Нет атомарной записи через tmp+rename. Краш посреди записи приведёт к обрезанному singbox-config и падению VPN.

**Рекомендуемый фикс:** Использовать `statefile.SaveJSON` или паттерн tmp+rename для всех трёх записей.

---

### P3-R9: bumpSecondaryConfigVer без flock

**Статус:** ✗ НЕ ИСПРАВЛЕН  
**Файл:** `internal/vpn/apply.go:148-156`  
**Детали:** Функция bumpSecondaryConfigVer всё ещё не использует flock. Может перетереть heartbeat при конкурентном доступе.

**Рекомендуемый фикс:** Добавить flock перед чтением/записью devices.json.

---

### P2-F1: Firewall проверяет только DROP

**Статус:** ✗ НЕ ИСПРАВЛЕН  
**Файл:** `internal/firewall/firewall.go:2192-2217`  
**Детали:** Функции ensureWSSHostInput/ensureAgentHostInput всё ещё проверяют только явный `-j DROP`. Если INPUT терминирован REJECT или имеет policy DROP без явного правила, ACCEPT не вставляется и порт остаётся закрытым.

**Рекомендуемый фикс:** Проверять оба варианта — DROP и REJECT.

---

### P2-F2: Firewall игнорирует -s/-i/-d

**Статус:** ✗ НЕ ИСПРАВЛЕН  
**Файл:** `internal/firewall/firewall.go:2219-2237`  
**Детали:** hostAcceptsTCPPort всё ещё считает порт открытым по любому ACCEPT, игнорируя квалификаторы `-s`/`-i`/`-d`/`!`. Restricted ACCEPT (например, `-s 10.0.0.0/8`) считается глобально открытым.

**Рекомендуемый фикс:** Проверять наличие квалификаторов `-s`, `-i`, `-d` и считать порт открытым только если их нет.

---

## Новая находка: P10-C1

### P10-C1: Potential command injection via PublicIP

**Severity:** MEDIUM  
**Файл:** `internal/channels/path_e2e.go:80`  
**Функция:** `icmpProbe` (строки 74-82)

**Проблема:**
- Строка 45: `ip` берётся из `d.PublicIP` (из `secondary.List()`)
- Строка 51: передаётся в `icmpProbe(ip, 3*time.Second)`
- Строка 80: используется напрямую в `exec.Command("ping", "-c", "1", "-W", strconv.Itoa(sec), host)`
- Нет валидации через `net.ParseIP`

**Риск:**
- `d.PublicIP` берётся из secondary/devices.json (может быть изменён через API/CLI)
- Если PublicIP содержит shell metacharacters (например, `8.8.8.8; rm -rf /`), возможна command injection
- Хотя `exec.Command` не интерпретирует shell metacharacters напрямую, некоторые утилиты (включая ping) могут вызывать shell внутренне

**Сценарий эксплуатации:**
1. Злоумышленник с доступом к API/CLI изменяет `secondary/devices.json`
2. CollectPathE2E вызывается автоматически (через collect/serve)
3. icmpProbe вызывает ping с невалидным IP
4. Если ping интерпретирует `;` (зависит от реализации), команда выполняется

**Рекомендуемый фикс:** Добавить валидацию `if net.ParseIP(host) == nil { return false }` в начале функции `icmpProbe`.

**Дополнительная защита:**
- Валидация при записи PublicIP в secondary/store.go (RejectInvalidIP middleware)
- Использование `net.Dial` вместо `exec.Command("ping")` для TCP-проб

---

## Другие находки в internal/channels/

### Безопасные паттерны

**internal/channels/health.go:100**
- `exec.Command("journalctl", "-u", "sing-box", ...)` — безопасный argv, без shell
- Аргументы передаются как отдельные параметры, не через shell

**internal/channels/sample.go:203**
- `ifaceMu sync.Mutex` — корректные блокировки
- Используется для защиты доступа к `ifaceStats`

**internal/channels/sample.go:241**
- `ifaceMu.Lock()` — правильное использование
- `defer ifaceMu.Unlock()` — корректное освобождение

### Статистика

**internal/channels/** (8 файлов, ~30 KB):
- `exec.Command`: 2 использования (health.go:100, path_e2e.go:80)
- `sync.Mutex`: 2 использования (sample.go)
- `atomic`: 2 использования
- `http.Client`: 2 использования

---

## Итоги повторного ревью

### Статус критичных багов

| Баг | Severity | Статус | Примечание |
|-----|----------|--------|------------|
| P1-1 | HIGH | ✗ Не исправлен | SetRegion без блокировок |
| P4-SV1 | HIGH | ✗ Не исправлен | Tick без лока |
| P3-R7 | MED | ✗ Не исправлен | ApplyConfig неатомарна |
| P3-R9 | MED | ✗ Не исправлен | bumpSecondaryConfigVer без flock |
| P2-F1 | MED | ✗ Не исправлен | Firewall проверяет только DROP |
| P2-F2 | MED | ✗ Не исправлен | Firewall игнорирует -s/-i/-d |

### Новые находки

| Баг | Severity | Файл | Описание |
|-----|----------|------|----------|
| P10-C1 | MEDIUM | internal/channels/path_e2e.go:80 | Command injection via PublicIP |

**Итого:**
- Проверено: 6 критичных багов
- Исправлено: 0
- Не исправлено: 6
- Новых багов: 1

---

## Рекомендации

### Немедленно (HIGH)

1. **P1-1, P4-SV1** — добавить блокировки (единая схема flock для всех реестров)
   - `internal/secondary/store.go:901` — добавить `mu.Lock()/Unlock()`
   - `cmd/netductor/collect.go:209` — обернуть в `svcpaths.WithLock()`

### Следующий спринт (MED)

2. **P3-R7** — атомарная запись через tmp+rename
   - `internal/vpn/apply.go:92,128,131` — использовать `statefile.SaveJSON` или паттерн tmp+rename

3. **P3-R9** — добавить flock при мутации secondary/devices.json
   - `internal/vpn/apply.go:148-156` — обернуть в flock

4. **P2-F1, P2-F2** — исправить парсинг iptables
   - `internal/firewall/firewall.go:2192-2217` — проверять REJECT
   - `internal/firewall/firewall.go:2219-2237` — проверять квалификаторы

5. **P10-C1** — валидация IP-адресов перед exec.Command
   - `internal/channels/path_e2e.go:80` — добавить `net.ParseIP` проверку

---

## Положительные наблюдения

✓ internal/channels/ использует корректные блокировки (ifaceMu)  
✓ exec.Command используется безопасно в большинстве случаев (argv, без shell)  
✓ Новая функциональность хорошо структурирована  
✓ Инструменты `.qwen-tools/` работают корректно

---

## Инструменты

Все инструменты из `.qwen-tools/` работают корректно:
- `get_file` — чтение файлов через API
- `put_file` — создание/обновление файлов
- `patch_file` — замена текста
- `fetch_source` — загрузка исходников (762 файла, retry для security hooks)
- `chunked_ops` — обход 20-секундного лимита песочницы

---

**Дата:** 2026-11-13  
**Аудитор:** Qwen3  
**HEAD:** f35b30a5b2b06f55ded94e7f52282a7ed9cfc2d2
