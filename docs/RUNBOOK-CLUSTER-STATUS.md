# Runbook: cluster status & egress thin API

Phase 4 read-only views. **No mutations** on these paths.

## Control (active or standby)

```bash
netductor node-status
netductor cluster status
# session API (on control):
# GET /api/node/status
# GET /api/node/leader
# GET /api/cluster/status
```

Cluster aggregates:
- local control role / write_allowed / leader
- secondary registry heartbeats (`singbox_ok`, `uplink_ok`, version)
- optional `nodes` registry rows (edge/build)

TG: **Status → Cluster**. Web Control → **Nodes → Cluster status / Node status**.

Standby: SoT writes return **503**; promote is manual (`control promote`) — see [RUNBOOK-CONTROL-CUTOVER.md](RUNBOOK-CONTROL-CUTOVER.md).

## Egress (secondary agent)

While `netductor-secondary-agent` runs:

```bash
# on the secondary host (SSH)
curl -sS http://127.0.0.1:8791/status
curl -sS http://127.0.0.1:8791/healthz
netductor secondary local-status --json
```

Listen address is **loopback only** (`127.0.0.1:8791`). Not a second control plane.

## Upgrade note

Shipped on `main` after Phase 3–4 work. Production still needs a **batch release** then:

```bash
# on primary (after release tag exists)
netductor stack apply vX.Y.Z
netductor secondary cmd <id> upgrade:vX.Y.Z
```

Until then, thin status and cluster UI are code-only. See [UPGRADE-SAFETY-PHASE3-4.md](UPGRADE-SAFETY-PHASE3-4.md).
