# Архитектурный freeze

**RU** · [EN](../ARCHITECTURE-FREEZE.md)

Зафиксированный канон продукта (не ломать без явного решения):

- **Роли:** primary (control plane) + secondary (RU VPN entry)
- **Пользователи VPN:** VLESS (+ Reality); сервисная плоскость SP/PS
- **Agent plane:** mTLS **:8789** (временный public arm только через TG)
- **SSH:** порт **52222**, только ключ после bootstrap; один Mac-ключ на primary и secondary
- **FHS:** без `/opt` как корня продукта
- **Оператор:** Mac (`netductor-op`), не панель на VPS
- **Redirect:** только HTTPS **:8443**, без публичного :80
- **Убрано навсегда:** `API_ALLOW_PUBLIC` / `PLAIN_AGENT` / `API_PUBLIC` / `TRUST_PROXY` как постоянные режимы

Детали планов close-out: [ARCHITECTURE-PLAN](../ARCHITECTURE-PLAN.md).
