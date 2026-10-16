# PLAN: Edge USB storage (extroot + internal fallback + backup/restore)

Status: **implemented in tree** (lab e2e pending; release deferred) (agreed product decisions below; not implemented)  
Related: `PLAN-EDGE-DEPLOY-HYBRID.md`, modules `overlay_ext` / `vpn_client`, TR1200-class (16 MiB flash, USB 2.0).  
Context: post v0.9.402 (vpn_client free-space check + optional data-mount for sing-box).


## Implementation progress

| Phase | Status |
|-------|--------|
| A Detect + card + vpn gate | **done** (facts USB_DISK/BLOCK, Card, plan skip without USB, Web card/modules) |
| B Mount existing ext4 | **done** (data mount path in applyEdgeStorage) |
| C Format + extroot | **done** (opt-in `usb_format` + overlay_ext → fstab UUID, reboot) |
| D vpn after large overlay | **done** (skip vpn same pass when extroot needs reboot) |
| E Pin sync | **done** (nd-pin-sync + cron 12h) |
| F Alerts + mute | **done** (recovery ✅) |
| G Backup A/B + remote restore | **done** (A existing + B push API) |
| H Offline restore | **done** (storage_restore local + pin merge) |


## Goal

1. Detect USB and show it on the device card (size, FS, used/free, mount, UUID).
2. With **explicit operator consent**, format (optional) and use USB as **full extroot**.
3. Keep **internal flash** as warm fallback: agent, network, Wi‑Fi, firewall — shop runs **without VPN** if USB is missing/dead.
4. **Periodic sync** (whitelist) USB/active overlay → internal pin so fallback stays current.
5. **Alerts** for missing/unhealthy USB; per-device ability to **disable** USB expectation alerts.
6. **Backups on primary** + remote (and offline) restore onto a **new** USB stick.

Non-goal: require USB for basic LAN/AP/agent. Modem/4G on same USB port is **out of scope for now**.

## Agreed decisions (product)

| # | Decision |
|---|----------|
| D1 | Target mode with USB + consent = **full extroot**, not sing-box-only data mount as end state. Data-mount remains emergency path only. |
| D2 | Deploy order: **internal thin stack first** → USB prepare/extroot/reboot → **sing-box on large overlay** → pin sync. |
| D3 | **Modules step:** if probe found **no USB**, disable / grey out **vpn_client** (and extroot/format controls). No silent hard-fail after operator enabled VPN with nowhere to put it. |
| D4 | Agent → primary **direct** (`SERVER=https://primary:8789` mTLS). SoftFallback + PrimaryHost direct so VPN loss ≠ blackhole; agent must not depend on sing-box. |
| D5 | USB loss → boot internal pin → AP + agent + ISP path; only extra features (VPN binary) gone. |
| D6 | Destructive USB ops only with UI checkboxes + warning in the same session. |
| D7 | fstab by **UUID**, never rely on `/dev/sda1` alone. |
| D8 | Pin sync: whitelist only; never sing-box on internal; safe replace (checksum / stage+rename); do not overwrite working agent with a failed copy. |
| D9 | LuCI ASU off applied on **internal before** extroot **and** included in pin whitelist so fallback has no upgrade nag. |
| D10 | Single source of truth while USB healthy = **USB overlay**. Sync direction normal ops: **active → internal pin** only (not internal overwriting USB on every boot). |
| D11 | USB alerts: missing / extroot failed / unhealthy + recovery OK. UI: per-device **expect USB / mute storage alerts**. |
| D12 | Guest Wi‑Fi / nft / essential firewall snippets in pin whitelist. |
| D13 | Backups on primary: type **A** sysupgrade-style + type **B** overlay **file** archive (not disk image). |
| D14 | Remote restore new USB: agent format → extroot → fetch B → extract → **merge internal pin on top** → alert ready → **manual reboot** from UI. |
| D15 | Offline restore path when no path to primary (see below). |

## Principles

| Principle | Detail |
|-----------|--------|
| Agent direct | Control plane must survive VPN and USB loss whenever path to primary exists. |
| VPN loss ≠ shop blackout | LAN egress via ISP when tunnel down. |
| USB loss ≠ brick | Internal pin sufficient for AP + DHCP + agent. |
| No silent destroy | Format/extroot only opt-in. |

### What fits on internal without sing-box

**Yes:** agent binary, mTLS, UCI (network/wireless/dhcp/firewall/system), guest rules, small state, ASU-off.  
**No:** full sing-box (and other large packages).

## Architecture

```text
  USB ext4 (UUID)  --extroot-->  /overlay   (sing-box, packages)
         |
         |  sync whitelist
         v
  Internal overlay pin  --boot if USB fail-->  AP + agent + ISP
         |
         |  backup A + B
         v
  Primary  /var/lib/netductor/edge-backups/<device_id>/
```

## Deploy sequence (agreed)

```text
1) Probe (incl. USB facts) -> card
2) Module selection:
   - no USB -> vpn_client OFF (disabled); format/extroot OFF
   - USB present -> operator may enable Use USB / Format / Extroot / Pin
3) Apply thin stack on **internal** (agent, UCI, guest, ASU off) — never sing-box yet
4) If format/extroot:
   - mkfs if consented -> copy overlay -> fstab UUID -> reboot
5) After extroot up -> vpn_client / sing-box on large overlay
6) Enable pin sync timer; take backup A (+ B if extroot active)
```

First boot after format must not leave the shop without AP: thin stack on internal **before** pivot.

## UI

### Card

Internal overlay free/total; USB present, size, partitions, FS, mount, free/used, UUID; health summary.

### Checkboxes

- Use USB for storage
- Format USB (destroy data) — confirm
- Extroot (move /overlay to USB)
- Pin fallback to internal (default on if extroot)
- Expect USB / storage alerts (default on if extroot ever configured)

### Restore actions (day-2)

- Backup now (A and/or B)
- Init new USB + restore from primary
- Init new USB + restore from **offline pack** (local)
- Mute/unmute storage alerts
- Manual reboot after restore ready

## Backups on primary

| Type | Content | Use |
|------|---------|-----|
| **A** | OpenWrt-style config backup (sysupgrade -b or equivalent) | Fast config rollback; works **without** USB |
| **B** | tar.gz of **files** from USB extroot (not dd image) | Rebuild new stick |

Store under e.g. `/var/lib/netductor/edge-backups/<device_id>/` with manifest: `device_id`, time, OpenWrt version, path list, sha256. Keep **last + previous** slots. Schedule + after successful deploy. Agent uploads over mTLS; files mode `0600`.

### Remote restore (new stick, path to primary OK)

1. Operator: UI -> Init new USB + restore (confirm format).
2. Agent detects disk -> format -> extroot structure -> download **B** -> extract.
3. **Merge internal pin on top** (newer agent.env, mTLS, SERVER, critical UCI) so a weeks-old B cannot roll back connectivity to primary.
4. Alert: restore ready — reboot when convenient.
5. **Manual** reboot from UI.
6. Post-boot: extroot + heartbeat -> recovery alert OK.

Idempotent: second click does not re-format without confirm. Primary must have free space for archives.

### Offline restore (no path to primary)

When the shop has no WAN/path to primary (or primary down), remote fetch of B is impossible.

**Agreed approach:**

1. **Prepare offline pack** on a machine that can reach primary (operator laptop or another site), same family as edge offline-prep:
   - device_id-scoped archive **B** (and optional **A**)
   - manifest + checksums
   - optional last pin snapshot if stored on primary
2. Deliver pack: second USB, `nd-restore/` on the replacement stick, or copy from laptop on local LAN to the router when only LAN works.
3. On device: local action "Restore from local pack" (path or auto-detect `/mnt/.../nd-restore` / labeled volume).
4. Same pipeline as remote: format target if needed -> extract B -> **merge local internal pin on top** -> alert -> manual reboot.
5. When primary is reachable again: agent uploads fresh A/B and resumes normal sync.

**Constraints:**

- Offline pack is **secrets-bearing** — treat like backup material.
- Pack must be **device_id-bound** (refuse restore on mismatch).
- Prefer generating pack while device still healthy; emergency pack from primary last B only.

## Pin whitelist (indicative)

- netductor-agent binary (if size allows)
- `/etc/netductor-agent/**` (config, mtls; no huge caches)
- `/etc/config/network`, `wireless`, `dhcp`, `firewall`, `system` + guest/netductor snippets
- ASU-off UCI

**Never:** sing-box binary, opkg caches, large logs

Sync: every ~12h + after successful deploy; skip if internal free below threshold; stage + verify then replace.

## Alerts

| Event | |
|-------|--|
| Expect USB but missing | warning |
| Extroot configured, not mounted | critical |
| RO / repeated I/O errors | warning |
| Recovered | recovery OK |
| Restore ready / restore failed | info / critical |

Mute when operator clears "expect USB".

## Implementation phases

| Phase | |
|-------|--|
| **A** | USB detect + card fields + RU/EN; module gate: no USB -> vpn_client disabled |
| **B** | Mount existing ext4 by UUID (no mkfs) |
| **C** | Opt-in format + extroot + reboot |
| **D** | vpn_client only after large overlay; SoftFallback + PrimaryHost |
| **E** | Internal pin + sync |
| **F** | Storage alerts + mute |
| **G** | Backup A/B to primary; remote restore + merge pin |
| **H** | Offline restore pack + local restore path |

## Testing

1. No USB -> vpn_client disabled in UI; thin deploy OK.
2. USB + extroot -> sing-box OK; soft-fallback OK.
3. Pull USB -> AP + agent; alert; no VPN.
4. New stick remote restore + merge -> manual reboot -> same function.
5. Offline pack restore without primary -> merge pin -> reboot -> later re-register backups.
6. VPN down, USB present -> ISP LAN + agent heartbeat.

## Out of scope (now)

- 4G modem sharing the USB port
- Disk images / RAID
- Auto-reboot after restore
- Guaranteeing sing-box on internal flash

## Open (minor)

- Exact free-space threshold on internal for pin
- Whole-disk vs single-partition format default (prefer single partition; whole-disk only if disk empty)


## Residual (not code)

- Hardware lab e2e on Cudy + USB
- Release cut when operator requests
