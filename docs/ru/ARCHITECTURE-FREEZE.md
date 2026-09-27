**RU** · [EN](../ARCHITECTURE-FREEZE.md)

# Заморозка архитектуры (канон)

**Статус:** зафиксировано. Можно наращивать **функции**; топологию/роли/пути ниже **не менять** без явного решения владельца.

## Канон

| Правило | Деталь |
|---------|--------|
| Роли | **primary** (зарубежный control) + **secondary** (вход VPN в РФ). Не полное зеркало primary на RU |
| Пользователи | VLESS Reality; клиенты → secondary → uplink primary |
| Service plane | Dual **SP/PS** WG-over-WSS (`10.87.10.0/30`, `10.87.11.0/30`); agent S→P на SP; recovery P→S на PS |
| API :8789 | только mTLS; WAN deny кроме service CIDR + `api-allow.cidr`; временный open **только** `api-public arm` (UI/CLI), **без** permanent env |
| Operator API | loopback на ноде; **нет** публичной admin UI на VPS |
| Оператор | **Mac `netductor-op`** (TUI + local Web) через tunnel/VPN |
| SSH | **52222**, key-only после bootstrap (**обе** VPS; один Mac-ключ) |
| Пути | FHS: `/usr/local/bin`, `/usr/local/share/netductor`, `/etc/netductor`, `/var/lib/netductor`. **Без `/opt/netductor`** |
| Failover policy | primary — источник правды; secondary — через agent heartbeat `failover_policy` |
| Backup | конфиг + данные; бинарники с Release по списку компонентов |
| Redirect | **только HTTPS :8443** (после LE). **Без public :80**. Пути: `/r`, `/profiles/*`, `/healthz` |

## Убрано из продукта (не возвращать)

- permanent `NETDUCTOR_API_ALLOW_PUBLIC`
- `NETDUCTOR_PLAIN_AGENT` / plain :8788
- `NETDUCTOR_API_PUBLIC` (non-local serve)
- `NETDUCTOR_TRUST_PROXY`
- Legacy admin UI на VPS
- префикс `/opt/netductor`

## До «только фичи»

См. [ARCHITECTURE-PLAN.md](ARCHITECTURE-PLAN.md).

**Код (0.9.75):** снятые knobs не загружаются из conf. Не возвращать пути включения.
