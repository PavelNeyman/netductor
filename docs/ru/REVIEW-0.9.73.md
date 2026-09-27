**RU** · [EN](../REVIEW-0.9.73.md)

# Ревью кода и безопасности — v0.9.73

**Вердикт:** готово к dual-node smoke в рамках freeze. Критичных дыр WAN admin нет.

Безопасность: loopback API; mTLS :8789; нет plain agent / permanent public; redirect :8443 only; recovery arm; SSH 52222.

Recover VPN: inbound mux schema + ensure-relay-uplink зафиксированы.

Паритет Mac: TUI и Web → один `internal/deploy`. OPCATALOG day-2 закрыт; deploy не из TG.

Доки: 42×EN/RU. Дальше: dual-node smoke → hardware e2e → фичи.
