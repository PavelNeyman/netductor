# UI parity

## Surfaces

| UI | Deploy VPS | Day-2 ops | Notes |
|----|------------|-----------|--------|
| **WebUI** (`op serve`) | Yes (Installer) | Max operator API + Advanced | Primary day-2 for rare POSTs |
| **TUI** (`op` / `netductor` TUI) | Yes (Wizard) | CLI via local/remote `netductor …` | Same backend as CLI |
| **Telegram** | **No** | Day-2: VPN, nodes, edge, NVR, git, DNS, backup… | Runs on node; admin chat only |

## TUI (0.9.11)

Tools: doctor, status, fleet, nodes, secondary, edge (list/pending/recovery), NVR, git, registry, addons, mTLS, backup, VPN, probes, SSH hosts, audit, sites.

## TG

Tools: guest, DNS, probes, backup, locations, NVR, metrics, updates, mTLS, git, registry, secondary, audit. No fleet deploy from TG.

## Not in any operator UI

Agent-plane only: enroll/heartbeat, plain recovery pull without arm, etc.
