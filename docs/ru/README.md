| [REVIEW-SUITE.md](REVIEW-SUITE.md) | [EN](../REVIEW-SUITE.md) | Полный комплекс ревью (тесты, UI, i18n, доки) |
| [REVIEW-PLAN-2026-10-19.md](REVIEW-PLAN-2026-10-19.md) | [EN](../REVIEW-PLAN-2026-10-19.md) | План прогона ревью с чекбоксами и доработками F1–F10 |
| [REVIEW-REPORT-2026-10-19.md](REVIEW-REPORT-2026-10-19.md) | [EN](../REVIEW-REPORT-2026-10-19.md) | Находки ревью: безопасность, качество, мёртвый код, рефакторинг |
| [UPDATE-SYSTEM-IMPLEMENTATION-PLAN.md](UPDATE-SYSTEM-IMPLEMENTATION-PLAN.md) | [EN](../UPDATE-SYSTEM-IMPLEMENTATION-PLAN.md) | План единой системы обновления компонентов с чек-листом |
| [AUDIT-ARTIFACTS.md](AUDIT-ARTIFACTS.md) | [EN](../AUDIT-ARTIFACTS.md) | Правила хранения доказательств, защиты секретов и статусов аудита |
**RU** · [EN](../README.md)

# Документация netductor

Сначала: [ARCHITECTURE-FREEZE](ARCHITECTURE-FREEZE.md) · [AGENT_HANDOFF](AGENT_HANDOFF.md) · [RECOVER-DRILL](RECOVER-DRILL.md)

Индекс продукта, планов и ops — те же имена файлов, что в EN `docs/`.

## Безопасность и планы

- [HOST-AUDIT.md](HOST-AUDIT.md) — аудит хостера
- [PLAN-SERVICE-ACCESS-POLICY.md](PLAN-SERVICE-ACCESS-POLICY.md) — политики сервисов

## Архив аудитов

- [Правила хранения артефактов аудита](AUDIT-ARTIFACTS.md)
- [Шаблон отчёта аудита](audits/TEMPLATE.md)
- Каждый существенный аудит сохраняется в виде датированного файла в `docs/audits/` и эквивалентного отчёта в `docs/ru/audits/`; ссылки добавляются в соответствующий план и issue/PR.
