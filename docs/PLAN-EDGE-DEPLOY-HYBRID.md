# PLAN: Edge deploy hybrid (op-SSH + presets + facts + modules)

Status: **in progress**  
Goal: factory OpenWrt (Cudy, RPi, …) → one operator deploy → ready device, without assuming Cudy-only topology or device→primary connectivity during install.

## Non-goals

- Long-term MITM of agent traffic through operator.
- RouterOS/MikroTik in the same UCI apply path (separate executor later; shared intent labels only).
- Changing runtime model: after uplink, agent↔primary remains the control plane.

## Architecture (agreed)

### Transport split

| Phase | Who | Channel | Purpose |
|--|--|--|--|
| **Deploy-time** | `netductor-op` | **SSH only** to device | Probe facts, resolve plan, apply all modules (incl. WAN that creates uplink) |
| **Runtime** | agent | HTTPS to primary | Enroll, heartbeat, later policy/remote cmds |

Primary is **source of policy for the operator** (or offline cache on op), not the peer of a factory device with no internet.

### Op modes

1. **Full offline:** presets + agent binary + optional cached template on op; no primary reachability.
2. **Op online:** op fetches token/mTLS/template from primary, then still applies **only via SSH** to device.

### Hybrid content model

- **Preset** — default module set (`travel-router`, `sbc-lab`, later `sbc-dual-nic`).
- **Facts** — probed on device (SSH deploy-time; agent can refresh at runtime).
- **Plan** — `preset ∩ facts` → list of modules with apply|skip + reason.
- **Template/overlay** — intent (lan_ip, ssid, vpn policy…); modules interpret intent when capable.

## Device facts (probe)

Collected without requiring uplink:

```json
{
  "arch": "arm64",
  "board": "raspberrypi,3-model-b",
  "os": "openwrt",
  "ifaces": [
    {"name": "eth0", "type": "ethernet", "up": true},
    {"name": "wlan0", "type": "wireless", "up": false}
  ],
  "radios": [{"name": "radio0", "band": "2g"}],
  "uci": {
    "lan_device": "br-lan",
    "lan_ip": "192.168.1.1",
    "wan_present": false,
    "wan_proto": ""
  },
  "flash_mb": 0
}
```

Rules of thumb:

- `wan_present` = UCI interface `wan` exists **or** a second ethernet suitable for uplink (policy may still require explicit assignment).
- RPi factory: typically one ethernet in `br-lan`, `wan_present=false`.
- Cudy-like: lan + wan, dual radio possible.

Probe implementation:

- Deploy-time: shell one-liner / small script over existing SSH helpers in `internal/deploy` + `internal/edge.Provision`.
- Runtime: agent status/enroll payload may include the same JSON (optional enrichment).

## Presets

| ID | Intended devices | Default modules |
|--|--|--|
| `travel-router` | Cudy TR1200-class | agent, lan, wan, wifi, guest?, vpn?, harden |
| `sbc-lab` | RPi OpenWrt single-NIC | agent, lan, wifi?, guest?, harden; **wan skip** |
| `sbc-dual-nic` | RPi + USB-Ethernet | as travel-router once second eth mapped to wan |

CLI: `--preset travel-router|sbc-lab|…` (default: auto from facts if possible, else `travel-router` with skips).

## Modules (compose)

| Module | Requires | Deploy-time action |
|--|--|--|
| `agent_install` | — | binary + SERVER/TOKEN/DEVICE_ID + optional mTLS (existing Provision) |
| `lan_baseline` | lan | UCI lan_ip/mask/dhcp from template/flags |
| `wan_baseline` | `wan_present` | dhcp/static/pppoe on wan |
| `wifi_ap` | radios ≥ 1 | SSID/key per band |
| `guest` | radios ≥ 1 + intent | existing guest_apply path |
| `vpn_client` | uplink path (wan or default route) + intent | may defer if no route yet |
| `ssh_harden` | — | existing Harden (keys, password off) |

If requires unmet → **skip + reason**, not hard fail of whole deploy (unless module marked critical).

Critical by default: `agent_install`.  
WAN skip on RPi is success with warning, not failure.

## Deploy flow (target)

```text
netductor deploy edge --router 192.168.1.1 --id SITE --preset sbc-lab|travel-router ...
```

1. Resolve agent arch (flag or probe `uname -m`).
2. If op online: pull bootstrap token (+ mTLS issue) from primary; else use offline flags/cache.
3. SSH probe → Facts.
4. Load preset + template/overlay (primary or cache or CLI flags).
5. Build Plan; print plan (optional `--dry-run`).
6. Execute modules in order over SSH (reuse Provision / UCI apply / Harden).
7. Optional: wait for default route; report “runtime enroll expected”.
8. Do **not** require device→primary during steps 1–6.

## Mapping to current code

| Existing | Role after change |
|--|--|
| `internal/deploy.DeployEdge` / `EdgeOpts` | Orchestrator; add Preset, Facts, Plan; keep offline token/mTLS |
| `internal/edge.Provision` / `Harden` | `agent_install` + `ssh_harden` modules |
| CLI `--configure-net`, wan/wifi flags | Feed `lan_baseline` / `wan_baseline` / `wifi_ap` intent |
| `apply_template` / guest on agent | Runtime + optional post-uplink; deploy-time prefer SSH UCI |
| `board` / `arch` on enroll | Keep; extend with facts blob when agent online |
| Templates on primary | Intent for op fetch; not device pull at factory |

## Implementation phases

### P0 — foundation (this iteration)

- [x] `docs/PLAN-EDGE-DEPLOY-HYBRID.md` (this file)
- [x] `internal/edge/facts.go` — Facts types, JSON marshal
- [x] `internal/edge/preset.go` — preset IDs, default module lists
- [x] `internal/edge/plan.go` — `BuildPlan(preset, facts, intent) []Step`
- [x] Unit tests for RPi-like vs Cudy-like facts → expected skips
- [x] OPEN_ITEMS pointer

### P1 — probe + dry-run

- [x] SSH/local probe implementing Facts (`internal/deploy/facts_probe.go`) (uci + ip link + board.json / os-release)
- [x] `netductor deploy edge --dry-run` prints plan
- [ ] Agent optional `facts` in status payload (non-blocking)

### P2 — wire modules into DeployEdge

- [x] Gate wan proto when plan skips wan (P1 partial)
- [ ] `--preset` flag on CLI + TUI label
- [ ] Explicit skip logs in deploy output

### P3 — polish

- [ ] Auto-preset from facts (single eth → sbc-lab hint)
- [ ] Offline template cache next to agents
- [ ] Docs: OPENWRT-LAB + operator deploy contract
- [ ] MikroTik called out as separate track

## Acceptance

1. RPi factory facts → plan skips `wan_baseline`; agent_install still runs.
2. Cudy-like facts → wan + wifi modules present when intent set.
3. Deploy with no device internet completes agent install via SSH.
4. Op offline path still works with bootstrap token (existing).
5. No regression: existing DeployEdge flags still function (preset defaults to current behavior as much as possible).

## AI implementation notes

- Prefer small PRs: types/tests first, then probe, then DeployEdge gates.
- Do not rewrite Provision in P0; only add plan layer beside it.
- Do not break mipsle agent releases; facts are additive JSON.
- Russian/English user strings: follow existing op CLI style.
