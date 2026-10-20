# Security-ревью и план исправлений — tip v0.9.444 → v0.9.445

**Дата:** 2026-10-20 · **Tip:** `1b1946b` / v0.9.444

Чеклист закрытия. Пункт закрыт только при критериях приёмки (негативные тесты + вызывающие пути).

## Находки

| ID | Серьёзность | В 0.9.445 |
|--|--|--|
| **H-1** Bootstrap → существующий `device_token` | Критично | **Закрыто** |
| **H-3** `IsRevoked` fail-open | Критично | **Закрыто** |
| **H-2** Ошибки firewall / слабый Collect | Высокий | **Закрыто** |
| **N-10** USB-пакеты OpenWrt (glob / имена) | Высокий | **Закрыто** |
| **M-5 / M-7 / N-8 / M-3 / L-3** | Высокий / local | **Закрыто** |
| **M-1 / M-4** | Высокий | **Открыто (P1)** |
| **N-9 / M-2 / M-6** | Medium / process | **Открыто (P2)** |
| **L-1** `:80` ACME | Архитектура | **Решение продукта** |

## By design

- iptables fallback без default DROP (anti-lockout)
- Disable VPN отзывает sub-URL
- Bootstrap для first enroll остаётся; убрана эскалация до personal token

## Осталось

M-1 NVR path, M-4 local SUMS, N-9 timeouts, M-2 Host/Origin, M-6 ship tag, L-1 ACME.

Полный текст критериев — EN: [SECURITY-REVIEW-2026-10-20.md](../SECURITY-REVIEW-2026-10-20.md).
