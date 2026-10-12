# ПЛАН: единый project ship (CI → пауза → publish)

**Статус:** дизайн согласуется · **не реализовано** (на 0.9.354)  
Полный текст и фазы: [../PLAN-PROJECT-SHIP.md](../PLAN-PROJECT-SHIP.md).

## Суть

- Одна команда `project ship <name>` для любого проекта.
- Не больше **одной** running-сборки на VPS; паузы и ожидание Mac — **без** слота, можно много.
- Пауза только **после** завершения текущего step; контейнеры выгружаются; на диске — снимок job.
- Mac (и любой external) — `waiting_external`; resume/complete по `job_id`.
- Авто tag patch+1, ручной `--tag`.
- Publish: `releases` (netductor) / `artifacts` / `registry`. Тесты fail → publish нет.
