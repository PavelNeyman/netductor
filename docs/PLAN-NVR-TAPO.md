# PLAN: Tapo C200 cameras → OpenWrt agent → primary NVR

**Status:** draft / research (2026-09-18)  
**Scope:** 2× TP-Link Tapo C200 (Wi‑Fi, RTSP) behind Cudy OpenWrt; record & manage on netductor **primary**; UI in Admin / TG / TUI; optional encryption at rest.

## Locked decisions (2026-09-18)

1. **Access only via VPN**  
   - No public exposure of RTSP (554), NVR UI, go2rtc, segment download, or Admin camera pages.  
   - Reachability: netductor VPN (primary/secondary profiles) only.  
   - TG delivery of clips is OK as files inside the operator bot (still assumes operator is trusted); do not publish permanent HTTP links without auth + VPN.

2. **Record on primary first**  
   - Default storage: encrypted volume under primary (see encryption).  
   - **Storage backend must be pluggable**: `local` (primary path) now; later `nfs` / `site-agent` / `home-nas` without rewriting camera inventory.  
   - Config knobs: `nvr.storage.backend`, `nvr.storage.path`, mount/endpoint for future home target.  
   - Migration path: stop recorders → remount/retarget path → optional copy/rsync of old segments → start recorders.

3. **Encryption: yes (at rest on primary)**  
   - Default: dedicated directory or block device for NVR data, encrypted (**LUKS** preferred for a partition/volume; **gocryptfs** acceptable for a folder if LUKS is awkward on the VPS).  
   - Unlock: boot/manual passphrase or keyfile on primary (operator-controlled); document unlock in RUNBOOK.  
   - Plaintext exists only while volume is unlocked and processes run — host admin with live access can still see mounts; goal is **disk image / offline host storage theft**, not defeating a malicious live root.  
   - Segment files should land **only** on the encrypted mount (`/var/lib/netductor/nvr` → encrypted backing).  
   - Secrets (RTSP passwords) stay in existing netductor secret store, not in plaintext config in git.

Related: edge agent leases, site/location model, VPN path to primary, storage (local disk / NFS).

---

## 0. Camera facts (Tapo C200)

- **RTSP** supported (local). Create **Camera Account** in Tapo app (Advanced → Camera Account) — **not** TP-Link cloud login.
- URLs (port **554**):
  - Main: `rtsp://USER:PASS@CAM_IP:554/stream1`
  - Sub (lighter): `rtsp://USER:PASS@CAM_IP:554/stream2`
- **ONVIF** often on port **2020** (optional discovery).
- Firmware: keep updated (C200 had serious LAN vulns; patch via Tapo app). Prefer cameras on **isolated IoT VLAN/SSID** if possible.
- Wi‑Fi only: stable DHCP **static leases** on OpenWrt are mandatory.

---

## 1. OpenWrt agent: leases + camera onboarding

### 1.1 List DHCP leases

Agent (or `netductor` remote via SSH/API) should expose:

| Field | Source (typical OpenWrt) |
|-------|---------------------------|
| IP | `/tmp/dhcp.leases` or `ubus call luci-rpc getDHCPLeases` / `ubus call dhcp ipv4leases` |
| MAC | same |
| Hostname | same (may be empty for Tapo) |
| Lease expiry | same |
| Interface / network | `ubus` / `iwinfo` / `ip neigh` |
| Wi‑Fi assoc (optional) | `iwinfo wlanX assoclist` — RSSI, connected |

**API sketch:** `GET /agent/v1/dhcp/leases`, `GET /agent/v1/wifi/clients`.

### 1.2 Identify cameras

Heuristics (combine):

1. MAC OUI (TP-Link ranges — maintain small list; not 100% reliable).
2. Hostname patterns (`Tapo*`, empty + fixed MAC).
3. Operator picks from lease list in UI → “mark as camera”.
4. Optional: probe `554/tcp` or ONVIF from **router LAN** (agent), not from internet.

### 1.3 Static DHCP lease

- UCI: `config host` under `dhcp` — `mac`, `ip`, `name`, optional `dns`.
- Agent applies idempotent change + `/etc/init.d/dnsmasq restart` (or reload).
- Persist in **site/camera inventory** on primary (MAC → name → RTSP user secret ref).

### 1.4 Path: camera → primary (options)

Cameras stay on **LAN behind NAT**. Primary is abroad/RU VPS — do **not** expose RTSP to the world.

| Option | How | Pros | Cons |
|--------|-----|------|------|
| **A. Site agent pulls? No — primary pulls** | Primary reaches camera via **VPN into site** or reverse tunnel | Central record | Need path home→VPS |
| **B. OpenWrt agent restream** | Agent or small box runs **go2rtc/MediaMTX**; primary records from `rtsp://agent:8554/cam` over VPN | One connection to cam; multi-consumer | CPU on router (Cudy weak) |
| **C. Port forward RTSP to VPN IP only** | DNAT 554 only from VPN/peer IPs | Simple | Still exposes cam stack; hairpin issues |
| **D. WireGuard/OpenWrt site-to-site** | Primary in VPN can use `10.x.cam_ip:554` directly | Cleanest for “primary records” | Requires site VPN (agent already enrolls) |
| **E. Secondary/primary as jump** | SSH/socat tunnel cam:554 → primary localhost | Works without full mesh | Ops heavy |

**Recommended direction for netductor:**

1. Short term: **static lease** + RTSP URL stored on primary; connectivity via **existing edge VPN / agent tunnel** so primary can `ffmpeg` to `rtsp://cam_lan_ip` **only when tunnel up**.
2. If Cudy CPU allows: **go2rtc on LAN** (RPi preferred over Cudy) restream; primary records restream.
3. Avoid public DNAT of 554.

---

## 2. Recording on primary

### Goals

- Continuous or scheduled record; **segmented files** (e.g. 2–10 min), not one endless file.
- Low CPU: **`-c copy`** (no re-encode) when H.264 from Tapo.
- Retention / max disk; optional NFS mount for archive.
- Config from netductor UI: segment length, path, enable/disable per camera, stream1 vs stream2.

### 2.1 Ready-made NVR / record stacks

| Project | Role | Pros | Cons | Fit |
|---------|------|------|------|-----|
| **[Frigate](https://github.com/blakeblackshear/frigate)** | Full NVR + optional AI | Mature, go2rtc built-in, recordings UI, HA | Heavier; AI wants Coral/GPU; more RAM | Overkill if only 2 cams continuous copy |
| **[go2rtc](https://github.com/AlexxIT/go2rtc)** | Restream only | Tiny, multi-client RTSP/WebRTC | **Does not record** | Pair with ffmpeg/Frigate |
| **[MediaMTX](https://github.com/bluenviron/mediamtx)** | Media server + **record** | Single Go binary, record segments, live | UI minimal; learn config | Strong lightweight candidate |
| **[MotionEye](https://github.com/motioneye-project/motioneye)** | Simple NVR UI | Easy, light | Older stack; motion false positives | OK for 2 cams |
| **[Shinobi](https://gitlab.com/Shinobi-Systems/Shinobi)** | NVR | Modern UI | Heavier Node; license nuances | Possible |
| **[ZoneMinder](https://zoneminder.com/)** | Classic NVR | Feature-rich | Heavy, dated UI | Poor fit for small VPS |
| **[Scrypted](https://www.scrypted.app/)** | Bridges (HKSV etc.) | Great for Apple | Not primarily “segment archive UI” | Optional later |
| **[Viseron](https://github.com/roflcoopter/viseron)** | NVR + AI | Alternative to Frigate | Still non-trivial | Optional |
| **Agent DVR (iSpy)** | Cross-platform NVR | Easy camera DB | Not Go-native; licensing tiers | Possible but external |
| **Moonfire NVR** | Rust NVR | Efficient design | Smaller community | Research |
| **VibeNVR** (mentioned 2026 press) | Docker NVR | Simple RTSP in | Verify license/activity before commit | Watch list |

### 2.2 Minimal custom (aligned with netductor)

**ffmpeg segment recorder** (per camera systemd or supervised by netductor):

```bash
ffmpeg -rtsp_transport tcp -i 'rtsp://USER:PASS@IP:554/stream1' \
  -c copy -f segment -segment_time 300 -segment_atclocktime 1 \
  -strftime 1 -reset_timestamps 1 \
  /var/lib/netductor/nvr/cam1/%Y-%m-%d_%H-%M-%S.mp4
```

| Pros | Cons |
|------|------|
| Minimal CPU/RAM; full control | Need own index DB, retention, UI |
| Easy NFS path | No built-in motion/AI |
| Fits Go supervisor in netductor | Must handle ffmpeg restart on disconnect |

**Go supervisor (`netductor nvr` or addon):**

- Config YAML/JSON: cameras[], segment_sec, path, transport tcp.
- Spawn/restart ffmpeg; write **index** (sqlite): camera_id, start, end, path, size.
- Retention job: delete oldest until free space / max days.
- Hooks for UI list/download.

Optional: **MediaMTX record** instead of raw ffmpeg — still index with netductor.

### 2.3 Storage

- Default: `/var/lib/netductor/nvr/` on primary.
- Optional **NFS/SMB** mount for bulk; keep index on local disk.
- Estimate: 1080p H.264 ~1–2 Mbit → ~0.5–1 GB/hour/cam → 2 cams 24/7 ≈ **24–50 GB/day** (tune bitrate via stream2).

---

## 3. Viewing / management UI

| Surface | Behavior |
|---------|----------|
| **Admin web** | Camera list; live (if go2rtc/WebRTC or HLS); calendar/list of segments; play in browser; download |
| **Telegram** | List by day/camera → send **file** (size limits!); or link via VPN-only URL |
| **TUI** | List + download to cwd; playback only if external player (`mpv`) available on workstation |
| **External** | Frigate/MotionEye UI behind VPN only |

**Live view:** prefer **go2rtc** restream → browser WebRTC; do not open camera ports publicly.

**TG limits:** large MP4 may fail — prefer short segments (2–5 min) or zip-of-links via temporary auth token over VPN.

---

## 4. Protecting video from host / disk theft

Threat: VPS host operator or stolen disk reads files.

| Approach | Protects against | Notes |
|----------|------------------|-------|
| **VPN-only access to NVR UI** | Remote casual access | Already netductor policy |
| **dm-crypt / LUKS volume** for `/var/lib/netductor/nvr` | Disk theft offline | Key at boot (passphrase or unlock over SSH) — host memory still sees plaintext while mounted |
| **gocryptfs / cryfs** on directory | Same class | User-space; easier on VPS |
| **Per-file encrypt** (age/sodium) after each segment | Host reading files at rest if key not on host | Recorder must encrypt before write; **key only on client** or split — hard for server-side playback |
| **Client-side only archive** | Strong | Primary only buffers; ship encrypted blobs to home NAS — complex |
| **Recording on site (LAN NAS)** not VPS | Host never has video | Best privacy; primary only control plane |

**Pragmatic recommendation:**

1. Prefer **record on site** (RPi/NAS behind Cudy) if paranoia is high; primary = remote view via VPN.
2. If record on primary: **encrypted dataset** (LUKS/gocryptfs) + UI only via VPN + no public ports.
3. Accept: **running** decoder/player on primary means plaintext in RAM — cannot hide from determined host admin.
4. Cameras: firmware update, IoT VLAN, no cloud required for RTSP path.

---

## 5. Suggested architecture (phased)

### Phase A — Discovery & inventory (agent)

1. Agent: leases + wifi clients API.
2. Primary Admin/TG: list leases; bind MAC → camera name; set static lease.
3. Store RTSP secret in primary secrets (not in git).

### Phase B — Path & record MVP

1. Ensure primary can open RTSP (VPN/tunnel).
2. ffmpeg segment recorder supervised by netductor **or** MediaMTX record.
3. sqlite index + retention.
4. Admin: list/download segments; TG: send file.

### Phase C — UX & live

1. go2rtc for live restream.
2. Optional Frigate if AI motion needed later.
3. NFS optional.

### Phase D — Hardening

1. **Encrypt NVR volume (required for primary storage)** — LUKS or gocryptfs; path only on unlocked mount.
2. Enforce VPN-only (firewall + no public listeners for NVR).
3. Pluggable storage backend stub for future **home** target.
4. IoT VLAN docs.
5. Alert on recorder down / tunnel down / unlock missing.

---

## 6. Recommendation for *this* project

| Layer | Choice |
|-------|--------|
| Agent | DHCP leases + static host + camera inventory |
| Transport | Site VPN / agent path — **no public RTSP** |
| Record | **ffmpeg `-c copy` segments** under netductor supervisor **or** MediaMTX |
| Live | **go2rtc** (optional Phase C) |
| Full NVR UI | Only if needed: **Frigate** (heavier) or MediaMTX + our Admin |
| Privacy | **VPN only** + **encrypted volume on primary**; later switch storage backend to home NAS/agent |

**Not recommended as first step:** ZoneMinder, public port forwards, re-encoding on primary.

---

## 7. Open questions (next chat)

1. Is there already a **site VPN** so primary can route to `192.168.x.0/24` behind Cudy?
2. Record **24/7** or motion-only?
3. Prefer **minimal ffmpeg** in-tree vs **MediaMTX** docker addon?
4. Max retention days / NFS yes-no?
5. Cudy CPU headroom for any on-router restream?
6. ~~Encrypt?~~ **Yes** (locked).
7. ~~Public access?~~ **VPN only** (locked).
8. Home storage later: NFS vs site-agent push vs record-on-LAN?

---

## 8. References (research 2026-09)

- Tapo RTSP: TP-Link FAQ; `stream1`/`stream2`; camera account in app.
- Frigate + go2rtc docs.
- MediaMTX (bluenviron) — record + proxy.
- MotionEye, Shinobi, ZoneMinder, Scrypted, Viseron — comparison landscape.
- ffmpeg `-f segment` / `-segment_time` for chunked recording.

---

*Draft for handoff — implement after path-to-LAN decision.*
