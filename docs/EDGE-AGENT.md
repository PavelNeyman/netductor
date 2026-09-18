# Edge agent

Outbound enroll → approve → template apply. No management VPN.

## Provision (from operator machine / VPS)

```bash
# agent binary for router arch must exist locally
netductor edge provision root@192.168.1.1 \
  --id site1 \
  --server https://vps.example:8787 \
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
