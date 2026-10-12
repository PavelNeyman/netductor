# ПЛАН: единый project ship

**Статус:** решения зафиксированы 2026-10-12 · реализация по фазам S0→S6  
Полный текст: [../PLAN-PROJECT-SHIP.md](../PLAN-PROJECT-SHIP.md)

## Решения

- Без timeout; удаление job явно (`project job rm`).
- Resume **не** повторяет ok-шаги; Mac — complete с артефактами.
- Одна команда цикла: **`project ship`** (тест → сборка → версия → publish).
- Без алиасов старых `release ship` / «build без publish».
- VPS: max 1 running; много paused/waiting.
