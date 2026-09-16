# Передача контекста (0.7.3-dev)

Полный EN: [AGENT_HANDOFF.md](../AGENT_HANDOFF.md)

- primary / secondary, redirect :80, mTLS :8789, netductor.conf
- TG: nav под сообщением, действия в тексте; Access in-place edit
- Корп OpenConnect + наш full-tunnel на одном устройстве часто роняют весь интернет
- Probes: api-health должен ходить на https localhost или tcp :8787
