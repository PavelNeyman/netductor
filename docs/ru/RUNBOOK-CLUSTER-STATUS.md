# Runbook: статус кластера и thin API на egress

Только чтение (Phase 4). **Мутаций** на этих путях нет.

## Control (active или standby)

```bash
netductor node-status
netductor cluster status
# session API на control:
# GET /api/node/status
# GET /api/node/leader
# GET /api/cluster/status
```

Кластер собирает:
- локальную роль control / write_allowed / leader
- heartbeat secondary (`singbox_ok`, `uplink_ok`, version)
- при наличии — строки `nodes` registry (edge/build)

TG: **Status → Cluster**. Web Control → **Nodes → Cluster status / Node status**.

Standby: запись SoT → **503**; promote вручную (`control promote`) — [RUNBOOK-CONTROL-CUTOVER.md](RUNBOOK-CONTROL-CUTOVER.md).

## Egress (secondary agent)

Пока работает `netductor-secondary-agent`:

```bash
# на secondary (SSH)
curl -sS http://127.0.0.1:8791/status
curl -sS http://127.0.0.1:8791/healthz
netductor secondary local-status --json
```

Слушает **только loopback** (`127.0.0.1:8791`). Это не второй control plane.

## Обновление

Код на `main` после Phase 3–4. На прод — **batch-релиз**, затем:

```bash
# на primary (когда тег релиза есть)
netductor stack apply vX.Y.Z
netductor secondary cmd <id> upgrade:vX.Y.Z
```

До релиза thin status и cluster UI только в коде. См. [UPGRADE-SAFETY-PHASE3-4.md](UPGRADE-SAFETY-PHASE3-4.md).
