# Глубокое ревью кода — 2026-10-02

Полные таблицы и ID находок: [../DEEP-CODE-REVIEW-2026-10-02.md](../DEEP-CODE-REVIEW-2026-10-02.md) (смысловой паритет с EN).

## Краткий итог

- `go test ./...` — **все пакеты с тестами зелёные**.
- Ядро безопасности (loopback API, mTLS :8789, hashed sessions, CT token compare, guest nft, stack без авто-даунгрейда) — **в целом сильное**.
- Замечания: legacy session-файлы по plaintext token (**S1**), `git show` rev без allowlist (**S2**), guest только radio0 (**S7**), пакет `svcpaths` без тестов, `InsecureSkipVerify` на recovery/LAN (принятый риск).
- Не заменяет live Phase C–G; сбор состояния: `scripts/collect-vps-state.sh`.

## Приоритет правок

1. Убрать legacy sessions (S1)  
2. Санитизация git rev (S2)  
3. Multi-radio guest (S7)  
4. Тесты svcpaths  
5. Ужесточение PIN/harden/ash -n тесты  

