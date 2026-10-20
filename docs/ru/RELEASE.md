## Чеклист (каждый релиз)

1. Поднять `VERSION` **и** `internal/version.Release` (пин URL деплоя).
2. Сборка: node, tg, agent, op (linux-amd64 + darwin-arm64 по необходимости).
3. Загрузить **все** ассеты в GitHub Release (без `netductor-tg` ломается бот).
4. `SHA256SUMS` в релизе; Formula `netductor-op` с актуальными sha256.
5. Handoff / OPEN_ITEMS — стоп-линия = tip.
6. Primary: `netductor stack apply vX.Y.Z`; secondary — agent upgrade отдельно.
