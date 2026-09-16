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
