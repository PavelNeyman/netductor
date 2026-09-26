# Node naming & registry / Именование нод и реестр

## Identity

- **`id`**: stable UUID in `/etc/netductor/node_uuid` (never changes on rename)
- **`hostname`**: `nd-<role>-<marker>` in `/etc/netductor/node_id` + OS hostname

## Name format / Формат имени

```
nd-<role>-<marker>
```

| Part | Meaning | Examples |
|------|---------|----------|
| `nd` | project prefix | always |
| `role` | fleet role | **`primary`**, **`secondary`**, `edge`, `lab` |
| `marker` | which one | `nl01`, `ru01`, `11870` (from IP), `home` |

Legacy aliases (normalized in code): `core` → `primary`, `relay` → `secondary`.

**Examples**

| Name | Meaning |
|------|---------|
| `nd-primary-nl01` | main control plane |
| `nd-primary-11870` | auto from public IP `x.x.118.70` |
| `nd-secondary-ru01` | RU ingress / secondary VPS |
| `nd-edge-home` | OpenWrt / site edge |
| `nd-lab-01` | scratch / test |

**Rules**

- lowercase letters, digits, hyphen only (`a-z`, `0-9`, `-`)
- max ~32 characters
- unique in the fleet registry

**How to set**

1. Env: `NETDUCTOR_HOSTNAME=nd-primary-nl01 netductor install`
2. TUI / Web Control → hostname
3. Existing `/etc/netductor/node_id`
4. Auto: `nd-primary-<suffix from public IP>` (secondary: `nd-secondary-<suffix>`)

## Bidirectional registry

Operator sets `desired_hostname` → device applies on next sync → heartbeat confirms.

Storage: `/var/lib/netductor/nodes/registry.json`

## API & CLI

- `GET /api/nodes` — list
- `POST /api/nodes/self` — register this VPS
- `POST /api/nodes/hostname` — `{"id":"…","hostname":"nd-primary-nl01"}`
- `netductor nodes list`
- `netductor nodes rename <id> <hostname>`
