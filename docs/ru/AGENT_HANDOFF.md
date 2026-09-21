# Передача контекста

**Версия:** 0.8.27  

Полный текст (EN): [../AGENT_HANDOFF.md](../AGENT_HANDOFF.md)  
Ревью: [../REVIEW-2026-09-21-FULL.md](../REVIEW-2026-09-21-FULL.md)  
Риски: [../RESIDUAL_RISKS.md](../RESIDUAL_RISKS.md)  
Паритет UI: [../UI-PARITY.md](../UI-PARITY.md)

## Архитектура (зафиксировано)

- **Primary** (зарубежный VPS) — control plane, API localhost, TG, blocky, опционально NVR  
- **Secondary** (RU) — вход VLESS + agent, не полное зеркало сервисов  
- **Edge OpenWrt** — agent → mTLS `:8789`, enroll/approve, recovery только LAN  

Деплой с **Mac TUI**. Upgrade primary/secondary — pure Go, версия `deploy.Release`.

## Следующий шаг оператора

Hardware e2e (Cudy / Tapo / MikroTik), при наличии домена — HTTPS redirect.
