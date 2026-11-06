# Бэклог правок кода после деплоя Secondary B (2026-11)

**Полная версия (EN):** [../PLAN-CODE-FIXES-AFTER-B.md](../PLAN-CODE-FIXES-AFTER-B.md)

Кратко — только **код**, не lab/e2e и не «просто операторские шаги».

| P | ID | Суть |
|---|-----|------|
| P0 | CF-1 | Remote TCP probe не должен становиться ok из‑за локального `ss` на том же порту |
| P0 | CF-2 | Watchdog не рестартит sing-box только из‑за SP/PS fail streak |
| P1 | CF-3 | Slot-aware agent/heartbeat/uplink VIP (slot1 ≠ 10.87.10/11) |
| P1 | CF-4 | Согласовать prepare и host-baseline UFW по `:22` |
| P1 | CF-5 | Опциональная зачистка hoster ALLOW (80/panel) с подтверждением |
| P2 | CF-6 | Остатки harden на re-deploy уже key-only хоста |
| P2 | CF-7 | Шум integrity:drift после ручной подмены бинаря |
| P2 | CF-8 | Протухшие «focused tests not run» в safety-плане → тесты или вычистить |

Сделано в ходе деплоя B (не открывать заново): 0.9.664–683 (ошибки SSH, mTLS, join, UFW routed, harden idempotency, prepared `/sub` IP).

Диагностика B: [../scripts/diag-secondary-b.sh](../scripts/diag-secondary-b.sh) (см. EN § scripts).
