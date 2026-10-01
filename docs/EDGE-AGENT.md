
## Route: private IP → always `direct`

On the edge sing-box client, one of the first route rules is:

`ip_is_private: true` → outbound **`direct`**

**What counts as private (typical):**
- RFC1918: `10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`
- Link-local, CGNAT blocks if classified private by sing-box
- Traffic to hosts on the **router’s own LAN** (phones, PCs, Tapo cameras, printers, LuCI at `192.168.x.1`)

**Why:**
1. **LAN must work without VPN.** Cameras RTSP, SMB, SSH to the router, LuCI, guest isolation targets, DHCP — none of this should be stuffed into VLESS.
2. **No hairpin through secondary.** Sending `192.168.50.20` to the RU VPS and back is useless and breaks local services.
3. **Soft fallback safety.** Even when the default path is VLESS, local traffic never depends on secondary being up.

**What still goes via VLESS (when proxy is healthy):**
- Public Internet destinations from **private Wi‑Fi / LAN clients** whose default gateway is the router (TUN `auto_route` on the router).

**Guest Wi‑Fi:** separate firewall zone; policy is **ISP only**, not “force everything into TUN”. Private-IP direct still applies on the router’s own stack.

**Primary / secondary addresses:** also forced **direct** so agent mTLS to primary and the VLESS dial to secondary do not recurse into the tunnel.

---

## Template `dns: vpn` (default)

Template field (default on new templates):

```json
"vpn": { "enabled": true, "mode": "tun", "fallback": "wan", "dns": "vpn" }
```

**Meaning of `dns: vpn`:**
1. Route rule **`protocol: dns` → `hijack-dns`**: DNS packets seen by sing-box are handled by its DNS module (not leaked “raw” to a random upstream on WAN alone).
2. DNS servers in client config:
   - **`ya`**: `77.88.8.8` — used for suffixes `.ru` / `.р` / `.su` (no detour through VLESS).
   - **`remote` / `remote2`**: `9.9.9.9` / `1.1.1.1` with **`detour: auto|proxy`** — resolution for the rest goes **through the VPN path** (secondary). That is the “LAN uses DNS over the same path as web traffic” behaviour; blocky on primary still sits on the control plane — edge does not talk to `127.0.0.1:53` on primary.
3. **`dns: wan`**: do not rely on hijack + remote detour (ISP/resolver on WAN).
4. **`dns: off`**: no dedicated DNS block in client JSON (OpenWrt dnsmasq defaults).

**LAN clients:** still use the **router** as DNS (dnsmasq). Upstream of dnsmasq effectively follows router routing: with TUN + hijack, queries are pulled into sing-box policy above.

**Not the same as “every answer comes from blocky on primary”.** Edge has no direct localhost path to primary’s blocky; filtering on primary applies to traffic that **exits via** secondary→primary uplink when that path is used. Edge-side lists can be added later if needed.




- Template `vpn.enabled` (default **true**): agent installs sing-box client, TUN `nd-tun`
- Traffic: private LAN → **VLESS to secondary**; **urltest** falls back to **ISP WAN** if proxy down
- Guest Wi‑Fi: separate zone, **ISP only** (not forced into TUN isolation beyond firewall)
- DNS mode `vpn`: DNS hijack + queries via proxy (foreign); `.ru` → Yandex DNS
- Agent mTLS to primary: **direct** (not through user VLESS)
- VPN account: `edge-<device_id>` — not shown in human Users list

## LuCI (temporary admin UI)

- Enable/disable **uhttpd** only (package stays).
- Default TTL **1h**; extend 4/24/72h.
- **SSH from Mac** (same LAN, no router internet) or **agent queue** on primary.
- Auto-disable when TTL expires (agent loop).
- TG: Routers → LuCI; Web/TUI/CLI `edge-luci`.

**EN** · [RU](ru/EDGE-AGENT.md)

# Edge agent

Outbound enroll → approve → template apply. No management VPN between edge and primary for SSH.

## Transport security

- Heartbeat/commands use **HTTP(S) to primary** with device **token** (not mutual SSH).
- **v0.8.30+:** control plane is **mTLS on primary `:8789`** (not VPN-dependent, not plain `:8787`).
- Workstation/deploy sets `SERVER=https://PRIMARY:8789` and installs client certs under `/etc/netductor-agent/mtls/`.
- Admin UI stays on localhost `:8787` (SSH tunnel). Node API stays loopback; enroll uses mTLS :8789 (no public API flag).
- OpenWrt recovery UI is **LAN-only** (`:7879`).

After first password bootstrap, provision installs the **operator (Mac) pubkey** and disables dropbear/OpenSSH password auth when possible.



## Root password (first-boot)

- **Current** password: optional — leave empty for factory OpenWrt (no password).
- **New** password: optional — set once during provision, then password SSH is disabled (operator key only).

## Agent architecture (first-boot)

**No OpenWrt package feed.** First-boot always places a pure-Go binary via SCP from the operator machine.

1. SSH to the router (password or key).
2. Probe: `uname -m`, `/etc/openwrt_release`, `opkg`/`apk` presence.
3. Map to asset:

| `uname -m` / OpenWrt | Asset |
|----------------------|-------|
| `aarch64`, `arm64` | `netductor-agent-linux-arm64` |
| `armv7*`, `arm` | `netductor-agent-linux-arm` (GOARM=7) |
| `x86_64` | `netductor-agent-linux-amd64` |
| `mips` / `mipsel` (e.g. **Cudy TR1200** MT7628) | `netductor-agent-linux-mipsle` (GOMIPS=softfloat) |
| `riscv64` | `netductor-agent-linux-riscv64` |

4. Download that asset for `deploy.Release`, then `edge.Provision`.
5. UI/CLI field **arch** defaults to **`auto`**; set explicitly only to override a bad probe.

Day-2 agent upgrades stay on netductor stack / `agent_update` (manual confirm), not `opkg`.

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


## DeployEdge dry-run checklist (0.9.155)

Ordered steps inside `deploy.DeployEdge` / `edge.Provision`:

| # | Step | Failure mode |
|---|------|----------------|
| 1 | Validate router host, device id, version; arch empty/auto or known GOARCH | Invalid id/arch aborts |
| 2 | Resolve `ServerURL` → must be `https://…:8789` (never plain :8787) | Wrong URL → TLS fail later |
| 3 | SSH **primary**: read `edge_bootstrap_token` | No key/token → abort |
| 4 | SSH primary: `mtls ensure` + `mtls issue-client <device_id>` | Warn if certs missing; agent may fail handshake |
| 4b | **SSH probe** `uname -m` (+ opkg/apk, DISTRIB_ARCH) → map to GOARCH | Unsupported CPU; override with `--arch` |
| 5 | Download `netductor-agent-linux-<arch>` (`amd64`/`arm64`/`arm`/`mipsle`/`riscv64`) | Missing GitHub asset / wrong arch |
| 6 | `edge.Provision`: SCP agent + write config + mTLS material + enable service | SSH to router (password or key) |
| 7 | Optional `NetConfigure`: UCI LAN/DHCP/Wi‑Fi 2.4+5 / WAN dhcp\|static\|pppoe | Network apply is **warn-only** if fails |
| 8 | Optional guest Wi‑Fi | Warn-only |
| 9 | Optional **Reboot** (`reboot` / `--reboot` / TUI yes) | Agent enrolls after boot |

### Operator flow (factory OpenWrt)

1. Reset router → join default Wi‑Fi/LAN → SSH `root@192.168.1.1`.
2. Mac: Settings → Remote = primary host + `~/.ssh/netductor_primary`.
3. **TUI** OpenWrt wizard **or** Web Installer → OpenWrt **or** CLI `netductor-op deploy edge …`.
4. Fields: device id, arch, server `https://PRIMARY:8789`, network (2.4/5, WAN), **reboot=yes**.
5. After reboot: TG/Web **Edge pending** → **Approve**.
6. Device appears in edge list; heartbeat + cmds.

### Parity Web / TUI / CLI (0.9.154)

- Web form: 2.4 + 5 SSID/key, LAN/DHCP, WAN static/pppoe, guest, reboot checkbox.
- TUI wizard: same field set.
- CLI: full flags + `--reboot`.


## Where `vpn.dns` / soft fallback are set

| Layer | Location |
|-------|----------|
| **Default seed** | Code `edge.EnsureDefaultTemplate()` → file **`/var/lib/netductor/edge/templates/default.json`** (on primary) |
| **Per-device** | Device `template_id` + optional overlay in devices store; served by `GET /api/edge/template?device_id=` via `TemplateWithVPN` |
| **Runtime on router** | Agent writes `/etc/netductor-agent/sing-box-client.json` on `apply_template` |
| **CLI** | `netductor edge template-get [id]` · `netductor edge template-set-vpn [id] dns=vpn mode=tun fallback=wan` |
| **API** | `POST /api/edge/templates` with full JSON body (session); bind via bind-template |
| **Web Day-2** | Templates get/bind actions (no dedicated form fields yet — use CLI/API or edit JSON) |
| **TG** | Routers → Templates / Bind / Apply (apply enqueues agent); no field editor for `dns` yet |

If template has no `vpn.dns`, `TemplateWithVPN` still injects **`dns=vpn`** when merging links.

