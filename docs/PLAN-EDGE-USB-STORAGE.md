# PLAN: Edge USB storage (extroot + internal fallback)

Status: **planned** (not implemented)  
Related: `PLAN-EDGE-DEPLOY-HYBRID.md`, edge modules `overlay_ext` / `vpn_client`, Cudy TR1200 class (16 MiB flash, USB 2.0).  
Stop-line context: after v0.9.402 (vpn_client free-space check + optional data-mount path for sing-box).

## Goal

On travel/site OpenWrt with tiny internal flash:

1. Detect USB storage and show it on the device card.
2. With **explicit operator consent**, prepare USB (format optional) and use it as **full extroot** (`/overlay` on USB).
3. Keep **internal flash** as a **warm fallback**: agent, network, Wi‑Fi, firewall — enough to run the shop without VPN if USB is missing or dead.
4. **Periodic sync** of critical files from USB overlay → internal overlay so fallback stays reasonably current.
5. **Alerts** when USB is missing/unhealthy; health signals for storage.

Non-goal: require USB for basic LAN/AP/agent operation.

## Product principles (must not regress)

| Principle | Detail |
|-----------|--------|
| **Agent → primary is direct** | Agent uses `SERVER=https://<primary>:8789` (mTLS). Must **not** depend on LAN VLESS/sing-box. `SoftFallback` + `PrimaryHost` direct rules in client JSON exist so control plane survives VPN loss. |
| **VPN loss ≠ shop blackout** | When sing-box/VLESS is down, **LAN egress uses ISP WAN** (soft-fallback / direct). Staff keep internet; only “via netductor path” features drop. |
| **USB loss ≠ brick** | Boot without USB → internal overlay; AP + DHCP + agent still work. Lost: large packages on extroot (notably **sing-box**). |
| **Destructive USB ops are opt-in** | Format / partition / extroot pivot only after UI checkboxes + clear warning. |

### Agent path today (verify in code)

- Provision writes `SERVER=https://primary:8789` (or host:8789), not via user VLESS.
- Edge VLESS client generator supports `SoftFallback` and **direct** to `PrimaryHost` so agent/enroll traffic is not hairpinned into the tunnel.
- Implementers must **regression-test**: with VPN down or USB unplugged, agent heartbeats still reach primary when WAN works.

## Hardware reality (TR1200-class)

| Item | Typical |
|------|---------|
| Internal flash | 16 MiB SPI NOR; usable overlay often ~8–12 MiB free |
| RAM | ~128 MiB |
| USB | 1× USB 2.0 (power budget ~2.5 W — modems may need powered hub) |
| Internal microSD | **No** on TR1200 |
| What fits **internal without sing-box** | Agent binary, mTLS material, UCI (network/wireless/dhcp/firewall), guest rules, small state — **yes, by design**. **sing-box full binary does not** (tens of MiB). |

## Architecture

```text
                    ┌──────────────────────────────┐
                    │ USB (ext4, UUID in fstab)     │
                    │  = /overlay (extroot)         │
                    │  sing-box, optional packages  │
                    └──────────────▲─────────────────┘
                                   │ pivot at boot
┌──────────────────────────────────┴──────────────────┐
│ Boot: ROM + try extroot                             │
│ If USB OK  → run with large overlay (VPN possible)  │
│ If USB fail → internal overlay (AP + agent + ISP)   │
└─────────────────────────────────────────────────────┘
         periodic sync (whitelist) ──► internal pin
```

### Two storage roles

| Role | Location | Contents |
|------|----------|----------|
| **Primary runtime overlay** | USB extroot | Full writable root after pivot: packages, sing-box, logs policy as configured |
| **Fallback pin** | Internal overlay | Whitelist only: agent, agent.env, device_id, mTLS, essential UCI snapshots, **not** sing-box |

### Why full extroot (decision)

Operator choice for this plan: **full overlay on USB**, not “sing-box-only data mount” as the end state.

- Data-mount-only remains a **fallback install path** if operator declines format/extroot but has a pre-mounted volume (already partially supported for vpn_client).
- Default recommended path when USB present + consent: **extroot**.

## UI / operator flow

### Device card (all UIs: Web, op TUI/CLI; not TG deploy)

Show storage block:

- Internal: overlay total/free (existing facts).
- USB: present yes/no; device node; size; partitions; FS type; mounted path; free/used; UUID if any.
- Health: last seen, mount ok, optional SMART-ish / I/O error count if available.

### Explicit controls (checkboxes)

| Control | Meaning |
|---------|---------|
| **Use USB for storage** | Master enable |
| **Format USB (destroy data)** | mkfs.ext4 on chosen partition/device — **required confirmation** |
| **Extroot (move /overlay to USB)** | Pivot after copy; forces reboot |
| **Pin fallback to internal** | Enable sync whitelist internal←USB (default **on** when extroot on) |

Hints: CIDR map / site LAN as elsewhere; warn that format is irreversible; warn USB power for hubs/modems.

### Deploy order (modules)

1. Probe facts (incl. USB).
2. If format/extroot selected → prepare USB (see phases) **before** or **with** network stage as documented in implementation notes.
3. Agent + UCI on device (works on internal).
4. Extroot pivot + reboot when selected.
5. After USB overlay up → install **sing-box** / vpn_client onto large overlay.
6. Start fallback sync timer.

If extroot selected but USB missing → **hard fail** before destructive network claims success (same class as vpn space check).

## Implementation phases

### Phase A — Detect + card (no destructive ops)

- Extend edge facts probe: `block info`, USB size, partitions, mounts, free.
- API/card fields + Web/op rendering (RU/EN).
- Tests: golden facts with/without USB.

### Phase B — Safe mount of existing ext4

- If partition already ext4 + operator enables “use USB”: fstab by **UUID**, mount `/mnt/netductor` or prepare for extroot path.
- No mkfs without checkbox.

### Phase C — Opt-in format + extroot

- Format only if checkbox set; target selection rules (prefer single empty disk; refuse mounted system disk).
- Copy internal `/overlay` → USB; fstab `target=/overlay` UUID; `block-mount` packages as needed.
- Reboot; verify extroot active (`mount | grep overlay` / `df`).
- Document recovery: pull USB → boot internal.

### Phase D — vpn_client on large overlay

- After extroot, install sing-box to `/usr/sbin` on USB overlay (space pre-check still required).
- SoftFallback + PrimaryHost direct remain mandatory in generated client config.

### Phase E — Internal pin + sync

**Whitelist (indicative):**

- `/usr/sbin/netductor-agent` (or compressed copy if size requires policy)
- `/etc/netductor-agent/**` (config, mtls, **not** huge caches)
- `/etc/config/network`, `wireless`, `dhcp`, `firewall`, `system` (and netductor guest snippets)
- device_id / bootstrap tokens as applicable

**Not on pin:** sing-box binary, large lists, package caches.

**Sync:** procd timer or agent cmd (e.g. every 6–24h + after successful deploy). Copy USB→internal only when both healthy; atomic replace where possible.

**Boot without USB:** internal pin must be sufficient for AP + agent + ISP path.

### Phase F — Alerts + health

| Event | Alert |
|-------|--------|
| USB expected but missing after boot | ⚠️ edge storage missing |
| Extroot configured but not mounted | 🔴 edge extroot failed |
| Mount read-only / I/O errors | ⚠️ edge storage unhealthy |
| Recovered | ✅ recovery alert (same pattern as channel/path) |

Health: periodic `df`, mount check, optional `iostat`/kernel USB disconnect count; expose in agent heartbeat facts → primary metrics/TG.

## Security / safety

- Never format without explicit UI consent in the same session.
- Prefer UUID over `/dev/sda1` in fstab.
- Do not store operator private keys on edge.
- Sync must not push secrets to world-readable paths.
- TG: alerts only; no edge deploy from channel.

## Testing (lab)

1. No USB → deploy without extroot; agent+LAN OK; vpn_client fails or skips with clear reason if no space.
2. USB + format + extroot → after reboot overlay on USB; sing-box installs; VPN soft-fallback works.
3. Pull USB → reboot → AP + agent up; alert fired; no sing-box.
4. Reinsert USB → extroot returns; optional clear recovery alert.
5. Kill VLESS/secondary → LAN still has ISP; agent still heartbeats primary.

## Documentation

- EN: this file.
- RU: `docs/ru/PLAN-EDGE-USB-STORAGE.md` (same meaning).
- Link from `OPEN_ITEMS` / handoff when work starts.
- Operator hint in Web edge wizard: short “USB / extroot” note.

## Out of scope (unless added later)

- RAID or multi-disk.
- Automatic modem+storage combo on one stick as primary design.
- Guaranteeing sing-box on internal flash.
- Silent background format.

## Open questions (product)

1. Default sync interval (proposal: 12h + on deploy success).
2. Whether “format” allows whole-disk wipe vs single partition only (proposal: single partition, whole-disk only if one partition table empty).
3. Minimum free internal after pin (reject sync if internal free &lt; threshold).
'''
