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

