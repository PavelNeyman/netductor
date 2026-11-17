# ADR: Роль + region (CF-14)

**Статус:** accepted 2026-11-06; **правка 2026-11-17**  
**План эволюции:** [PLAN-FLEET-MESH-EVOLUTION.md](PLAN-FLEET-MESH-EVOLUTION.md)

## Решение

| Ось | Канон | Legacy |
|-----|--------|--------|
| **Role** | `control`, `egress`, `build`, `edge` | `primary`≡`control`, `secondary`≡`egress` |
| **Region** | ярлык размещения, не роль | опционально |
| **Instance** | peer id + slot | blue-green |

**Роль ≠ страна.** Control — плоскость управления (API, policy, users…), не «зарубежный сервер». Egress — VPN-вход/путь трафика, не «только РФ». Типичный шаблон EU control + RU egress — рекомендация, не определение.

**Region:** предпочтительно `cc-city` (`ru-msk`, `fi-hel`). Hostname: `nd-<role>-<region>`.

Совместимость dual-read `primary`/`secondary` надолго. Multi-master control — не часть этого ADR (фаза 5 плана флота, опционально).
