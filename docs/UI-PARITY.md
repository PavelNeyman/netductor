# UI parity

## WebUI (`netductor-op serve`) — primary day-2 surface

Covers operator **session** API under `/api/*`, `/vpn/*`, `/health` via `/v1/node` proxy.

- Sections: Overview, VPN, Nodes, Edge, NVR, Git/Registry, Backup, Probes, **Advanced**
- Agent-plane only endpoints (`/api/edge/enroll`, heartbeat, secondary agent mTLS) are **not** operator UI actions
- Use **Advanced** for uncommon bodies

## Telegram / TUI

Core fleet ops + VPN users + nodes status. Rare POSTs → WebUI Advanced or CLI on node.

## Installer

Password allowed for first primary/secondary deploy only.
