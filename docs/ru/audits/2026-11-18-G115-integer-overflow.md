# Аудит G115 (integer overflow) — ai-tools

| | |
|--|--|
| **Дата** | 2026-11-18 |
| **Отчёт** | `aitools-report-netductor-v1121` (gosec G115) |
| **База** | после N1/N2/N3 (`0ee2870` на старте) |
| **Число** | 43 находки / 16 файлов |

G115 — сужение типа. Не всё баг: часто **ложные** срабатывания на UID Linux или на размеры, уже ограниченные протоколом/константой.

## Классы

| Класс | Смысл |
|--|--|
| **FP-uid** | Сравнение `Uid` (uint32) с `Geteuid()` (int) |
| **safe-bound** | Значение уже ограничено проверкой/константой |
| **real** | Конвертация без верхней границы, влияет peer/данные |
| **protocol** | Формат провода (CCM/KLAP/PKCS7) |

## Не-tapo

| Места | Класс | Решение |
|--|--|--|
| Все `Uid != uint32(os.Geteuid())` (agent, API, firewall, mTLS, secondary, svcpaths) | FP-uid | **Исправлено**: `int64(uid) != int64(os.Geteuid())` |
| `firewall` обход хостов в /16+ | safe-bound | **Исправлено**: явная проверка `> 0xffffffff` |
| `install/backup` длины NDG2-чанков | safe-bound | **Исправлено**: `ndg2PutChunkLen` |
| `nvr` Statfs free/total | real-ish | **Исправлено**: checked mul / float |
| `revbridge/ws` длина кадра из uint64 | **real** | **Исправлено**: лимит 64 MiB до alloc |
| `revbridge/mux` len→uint32 | safe-bound | уже есть `maxFrame` — оставить |
| `ws` write `byte(n>>8)` | protocol | chunk ≤ 32 KiB — оставить |

## Tapo

CCM/KLAP/TPAP/PKCS7 — **protocol**, без правок в этом проходе.

## Проверка

`go test -count=1` по затронутым пакетам — ok.

Дальше: повторный прогон ai-tools; отдельный журнал по U1000 residual.


## Повторный прогон 2026-11-18 (после фикса, tip `9205bbe`)

Инструмент: **ai-tools v1.12.1** · артефакты: `aitools-report-netductor-post-g115.json`

| Метрика | До (v1121) | После |
|--|--|--|
| gosec всего | 1213 | 1186 |
| **G115** | **43** | **20** |
| G123 | 1 | **0** |

### Снято G115: 27 мест

Все FP-uid, firewall, backup старые, nvr, **ws read**, mux/secondary/svcpaths как в EN-журнале.

### Осталось 20

- **tapo** 16 — protocol  
- backup / nvr / mux / ws write — **safe-bound** (см. EN)

Доп. правки по residual G115 не обязательны. deadcode/govulncheck в песочнице убиты по OOM — не доказательство.


## Проход Tapo G115 2026-11-18 (по местам, по запросу)

Не «оставили по договорённости» — разобраны все 16.

| Место | Суть | Исправление |
|--|--|--|
| CCM | L=3, обрезка длины | флаги константы, **отказ** если >24 бит |
| KLAP/TPAP seq | signed↔wire | **uint32** |
| EC parity | byte(Bit) | сравнение bool |
| PKCS7 | pad | проверка + uint8 |

**gosec по `./internal/tapo/...`: G115 = 0.**

