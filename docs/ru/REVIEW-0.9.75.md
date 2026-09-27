**RU** · [EN](../REVIEW-0.9.75.md)

# Ревью кода и безопасности — v0.9.75

**Вердикт:** готово к dual-node smoke в рамках freeze. Footgun-опции **убраны из кода**, не «запрещены при наличии».

Security: loopback API · mTLS :8789 · нет plain :8788 / TRUST_PROXY / VPS admin · redirect :8443 · recovery offline key · ndconfig игнорирует старые ключи.

Recover: two-pass · vpn apply · mux ON · LE re-issue при DOMAIN+LE_EMAIL.

Паритет Mac TUI/Web. Доки 42×EN/RU.

Дальше: dual-node smoke → hardware e2e → фичи.
