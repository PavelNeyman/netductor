# Edge agent

Outbound enroll → approve → template apply. No management VPN between edge and primary for SSH.

## Transport security

- Heartbeat/commands use **HTTP(S) to primary** with device **token** (not mutual SSH).
- **v0.8.23+:** control plane is **mTLS on primary `:8789`** (not VPN-dependent, not plain `:8787`).
- Workstation/deploy sets `SERVER=https://PRIMARY:8789` and installs client certs under `/etc/netductor-agent/mtls/`.
- Admin UI stays on localhost `:8787` (SSH tunnel). Never require `NETDUCTOR_API_PUBLIC=1` for edge enroll.
- OpenWrt recovery UI is **LAN-only** (`:7879`).

After first password bootstrap, provision installs the **operator (Mac) pubkey** and disables dropbear/OpenSSH password auth when possible.


## Provision (from operator machine / VPS)

```bash
# agent binary for router arch must exist locally
netductor edge provision root@192.168.1.1 \
  --id site1 \
  --server https://vps.example:8789 \
  --key ~/.ssh/id_ed25519 \
  --agent ./netductor-agent-linux-arm64
```

Only installs agent + bootstrap token. Config apply is done **by the agent**.

## Enrollment

```bash
netductor edge pending
netductor edge approve site1
netductor edge bind-template site1 default
netductor edge cmd site1 apply_template   # or auto on first run after approve
```

## Templates

Stored on VPS under edge `templates/`. Default created on serve.

```bash
netductor edge templates
# API: GET/POST /api/edge/templates (session)
# Agent: GET /api/edge/template?device_id= (device token)
```

Template ≠ device backup. Overlay on device: `lan_ip`, `ssid`, etc.

## Apply

Idempotent UCI diff (lan IP/mask, wifi ssid/key). VPN client profile — next iteration (`vpn.enabled` in template).

## Offline install from Mac (no WAN on router yet)

1. On Mac: build/download `netductor-agent` for router arch; SSH to router over LAN.
2. Install binary + `/etc/netductor-agent/config` (SERVER, TOKEN, DEVICE_ID).
3. Optional: write `/etc/netductor-agent/local.uci` (SSID, LAN IP, hostname, …).
4. Start agent (procd/init). It applies `local.uci` **immediately**, then **retries enroll** with backoff until the router can reach primary.
5. When WAN appears → enroll → **pending** on primary → approve → server template may refine config.

No internet on the router is required for steps 1–4.

## Agent commands (NVR / LAN discovery)

| Action | Arg | Result |
|--------|-----|--------|
| `dhcp_leases` | — | JSON `{leases:[{expiry,mac,ip,hostname,client_id}]}` from `/tmp/dhcp.leases` |
| `wifi_clients` | — | JSON `{clients:[{iface,mac,raw}]}` via `iwinfo` assoclist |
| `dhcp_static` | `mac=..\|ip=..\|name=..` | UCI `dhcp` host section + dnsmasq reload |

Enqueue: `POST /api/edge/cmd` or NVR helpers `POST /api/nvr/site/leases` etc. Results: `/api/edge/results`.

## Agent roadmap (management beyond NVR)

Worth baking into the agent command set over time (not all implemented yet):

| Area | Commands / data |
|------|-----------------|
| **LAN inventory** | leases, wifi clients, ARP/`ip neigh`, optional port probe (554) from LAN |
| **DHCP policy** | static host add/remove/list |
| **Wi‑Fi** | SSID list, client kick, channel/scan (careful) |
| **Firewall** | show/reload; IoT VLAN status |
| **VPN path** | uplink health, sing-box/client status (already partial) |
| **Storage on site** | future: local record path, disk free (for home NVR backend) |
| **Camera control proxy** | future: PTZ/night via LAN to cam (agent executes, primary never needs L2) |
| **Safe ops** | config_backup, agent_update, sysupgrade (already) |

Prefer **agent-executed** LAN actions (camera control, RTSP probe) so primary only speaks to agent over VPN.

CLI: `netductor nvr leases <device_id>` enqueues `dhcp_leases`.

### NVR 0.7.19-dev

- Agent: `rtsp_probe` (TCP + optional ffprobe)
- CLI: `netductor nvr probe <camera_id>`
- Ingest: `POST /api/nvr/ingest` (multipart `file` + `camera_id`) for site→primary segment push

### NVR 0.7.20-dev

- Agent records on LAN (`ffmpeg` segments in `/tmp/netductor-nvr`) and uploads to primary ingest
- CLI: `netductor nvr record start|stop <id>`
- TG: Cameras → 🔍/⏺/⏹ per camera
- Doctor: mountpoint / encryption hint

## NVR buffer on OpenWrt (Cudy / low flash)

Flash **16MB** is for firmware only — **do not** store video on overlay.

| Storage | Use for NVR buffer? |
|---------|---------------------|
| **/tmp** (tmpfs in **RAM**) | Default. On **128MB RAM** keep **NVR_MAX_MB=24** (or less). |
| **USB stick** | Recommended if available: mount e.g. `/mnt/sda1`, set `NVR_DIR=/mnt/sda1/netductor-nvr`, `NVR_MAX_MB=512`. |
| **LTE modem SD** | Only if the modem exposes a **USB mass-storage / block device** to the host. Many modems keep SD **internal** (modem FS only) — then OpenWrt **will not** see it. Check: `ls /dev/sd*`, `block info`, `dmesg`. |

Agent config (`/etc/netductor-agent/config`):

```
NVR_DIR=/tmp/netductor-nvr
NVR_MAX_MB=24
```

USB example:

```
NVR_DIR=/mnt/sda1/netductor-nvr
NVR_MAX_MB=512
```

Archive always on **primary** (ingest). Buffer is deleted after successful upload.

## PTZ for Tapo C200 — preferred path (HA-compatible)

Working stack in the wild: **[HomeAssistant-Tapo-Control](https://github.com/JurajNyiri/HomeAssistant-Tapo-Control)** → library **[pytapo](https://github.com/JurajNyiri/pytapo)**.

| Method | Reliability on C200 |
|--------|---------------------|
| **pytapo `motorMove`** | Primary — same as HA |
| ONVIF ContinuousMove :2020 | Fallback only; FW-dependent |

**Prerequisites (camera):**
1. Tapo app → **Me → Tapo Lab → Third-Party Compatibility → On**
2. **Camera Account** (Advanced → Camera Account), not TP-Link cloud login

**On site (agent host):**
```
pip3 install pytapo   # or python3-pytapo if packaged
# ship scripts/tapo_control.py to /opt/netductor/scripts/
```

Agent `camera_ptz` tries **pytapo script first**, then ONVIF.

Commands: `move left|right|up|down`, `night:on|off|auto`, `privacy:on|off`

## Recovery (LAN)

1. Primary: TG **Routers → Recovery code** or `netductor edge recovery [--site ID]` or Admin UI.
2. On site Wi-Fi: `http://<router-lan-ip>:7879/netductor-recovery` — enter Primary URL + code.
3. Agent enrolls with `CONTROL_ONLY=1` (no Wi-Fi/UCI template).
4. TG/Admin **Pending → Approve**.

## Register / locations

- `netductor edge register <device_id> [--site ID]`
- `netductor edge set-site <device_id> <site_id>`
- TG: Routers → Register; Admin: recovery card.

## Export

`netductor edge export -o file.json` / Admin Export — include in primary backup drills.
