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

## Перепрогон 2026-11-18 (ai-tools 1.12.1-dev)

- Сообщение `no Go files` **исчезло**.
- **errcheck**: ~330 hits (defer Close/Remove — triage позже).
- **ineffassign**: ~37 (TG keyboards, mtls rollover, vpn apply…).
- **deadcode/govulncheck**: `signal: killed` (OOM на полном дереве в sandbox) — повторить на машине с запасом RAM или с `-test=false`/узким scope.
- **SA4023** подтверждён staticcheck.

Артефакт: `artifacts/aitools-report-netductor-0.9.757-r2.*`


## Progress 2026-11-18

| ID | Status | Evidence |
|----|--------|----------|
| **N1** | **done** | `sessionOnce` returns `nil` on normal end, `errClientStopped` on Stop; loopKind distinguishes. Test `TestSessionOnceReturnsNilInterfacePossible`. staticcheck SA4023 cleared on package. |
| **N2** | **done** | `ServerTLSConfig`: `VerifyConnection` + `SessionTicketsDisabled` (revoke on every conn). |
| **N3** | **partial** | Removed unused `reFrom`, `dirSizeBytes`, `lastFlush`, `isChecksumish`, `holdHostLock`, `runCmd`. op TUI U1000 still open. |

## Progress 2026-11-18 (pass 2)

| ID | Status |
|----|--------|
| **N1** | done (prior) |
| **N2** | done (prior) |
| **N3 UI/cmd** | **done** — op TUI dead styles/menuItem/list/capture; cli_flags helpers; TG widget dead helpers; formatClusterHTML; lastAccessMsg; thin wrappers applyUFW/installHealthTimer/recoverPrevSnapshotSwap |
| **N4** | partial — mux S1000 fixed |
| **N3 internal residual** | **deferred** — ~20 U1000 in deploy/firewall/install/svcpaths/secondary (need wire-or-delete audit, not blind delete) |
| **N5 G115** | deferred — needs per-site bounds review |

`staticcheck -checks U1000 ./cmd/...` = 0. Full `./...` residual listed under deferred.

## G115 pass 2026-11-18

Journal: [audits/2026-11-18-G115-integer-overflow.md](audits/2026-11-18-G115-integer-overflow.md) (RU: [ru/audits/…](ru/audits/2026-11-18-G115-integer-overflow.md)).

- [x] Site-by-site classification (43 findings)
- [x] Real: ws frame DoS bound; nvr Statfs math; euid compare without uint32 narrow
- [x] safe-bound: firewall hostU32, backup chunk len helper
- [ ] Re-run ai-tools/gosec for delta count
- [ ] U1000 residual internal (separate)

