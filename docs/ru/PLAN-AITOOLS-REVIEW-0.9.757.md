# План: находки ai-tools (netductor 0.9.757)

**Статус:** открыт · **релиз не режем**, пока не закроем или явно не отложим  
**Источник:** `aitools v1.12.0`, `main` @ `37f6dac` / VERSION **0.9.757**  
**Артефакты:** sandbox `artifacts/aitools-report-netductor-0.9.757.{json,txt}`  
**Правило:** сначала код; релиз только по согласованию. Миграции/floors — CF-15 / `AGENTS.md`.

[EN](../PLAN-AITOOLS-REVIEW-0.9.757.md) · RU

---

## Объём прогона

| Инструмент | Результат |
|------------|-----------|
| gosec | Отработал; **1213** (много шума G204/G304) |
| staticcheck | Полезный сигнал (SA4023, U1000, стиль) |
| deadcode / errcheck / ineffassign / govulncheck | В aitools **≤1.12.0** ломалось (`.` вместо `./...`) — исправлено в **ai-tools v1.12.1**; перепрогон после релиза |
| trivy / shellcheck / golangci-lint | В том прогоне не ставились |

---

## Очередь

### P1

| ID | Суть | Где | Действие |
|----|------|-----|----------|
| **N1** | **SA4023** — сравнение всегда true; `sessionOnce` не отдаёт nil iface | `internal/revbridge/client.go` | Нормальный error-path + тест |
| **N2** | **G123** mTLS / session resumption | `internal/mtls/mtls.go` | VerifyConnection или отключение tickets; граница доверия в доке |

### P2

| ID | Суть | Действие |
|----|------|----------|
| **N3** | U1000 мёртвый код | Удалить или подключить |
| **N4** | Стиль staticcheck | Косметика без смены поведения |
| **N5** | G115 overflow | Безопасные приведения |

### P3 (не «чинить вслепую»)

G204/G304 на deploy SSH · G402 на probe/vendor · SHA1/MD5 Tapo/ONVIF · G101 false positive на пути секретов.

---

## Процесс

1. **N1 → N2 → N3** (N4/N5 по возможности).  
2. Повторный review **ai-tools ≥ 1.12.1**.  
3. Релиз netductor — только после `[x]` с SHA/тестами или явного defer в этом файле.  
4. Ломающий on-disk / apply / peer → `DeclareBinaryFloor` в том же коммите.

## История

| Дата | Событие |
|------|---------|
| 2026-11-18 | Первичный план по прогону 0.9.757 |
