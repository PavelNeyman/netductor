# Предпроверка VPS (кандидат secondary / primary)

**Статус:** полной Go-реализации пока нет (2026-10-21).

## Смысл

До покупки или смены secondary (или primary) — **read-only** батарея проверок: репутация IP, скорость, CPU/диск, путь до primary.

Старые shell-скрипты — источник; в product-команду **не** перенесены полностью.

## Что есть

| Инструмент | Когда |
|--|--|
| `netductor host-audit` | Day-2 на уже установленном узле: агенты хостера, residual |
| `doctor` / `channel status` | после install / join |
| Harden при deploy | чистка hoster agents |

## Пробел

Нет `netductor-op vps-check` с Mac на «голый» VPS по паролю. Если скрипты сохранились — пришлите, перенесём в `internal/vpscheck` + job + UI.

Подробнее: [HOST-AUDIT.md](../HOST-AUDIT.md).
