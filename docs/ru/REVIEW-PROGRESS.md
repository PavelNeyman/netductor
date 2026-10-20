# Журнал security/code review (прогресс)

Полный построчный EN-журнал остаётся каноническим маркером проходов в `docs/REVIEW-PROGRESS.md` (EN). Ниже — **тот же смысл** для операторов: что закрыто, куда смотреть дальше.

## Правила прохода (как в EN)

- Пакет не «прочитан», пока не открыт каждый файл.
- Находки дописываются; маркер сдвигается только после чтения.
- Рефакторинг копится отдельно от security findings.

## Закрытые волны (сводка)

| Волна | Суть |
|--|--|
| Pass 1 API / session / mTLS / policy / update / install / edge / NVR | F-пункты downloads SHA, session hash, allowlists |
| Channel / face / uplink monitoring | алерты + recovery |
| Build-host / jobs thin UI | agent plane :8789 auth |
| Host baseline / integrity / SSH allowlist / DoT | doctor WARN |
| TG menu / alerts channel | topics → channel |
| Recent 0.9.439+ | recovery HTTP client PEM fail-closed; incident CLI path; NVR live deadline; subtoken rename/revoke; firewall truncate; secreq.Eq; dead code purge |

## Открытое / ops

- D-1: RU-пары (этот файл — часть закрытия)
- `vpn sub-prune` + doctor WARN orphan sub tokens (0.9.440)
- Железо e2e: OpenWrt / NVR / MikroTik

## Стоп-линия

Сверяйте `VERSION` / `docs/AGENT_HANDOFF.md` / GitHub Release. Не считайте этот журнал заменой `OPEN_ITEMS`.
