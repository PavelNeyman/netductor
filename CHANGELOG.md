## 0.7.15-dev

- NVR: configurable retention (days / max GB / min free GB), background rotate, segment list API, ffmpeg recorder skeleton

## 0.7.14-dev

- NVR Phase A: agent dhcp_leases/wifi_clients/dhcp_static; /api/nvr/cameras inventory
- docs: EDGE-AGENT agent management roadmap; PLAN-NVR progress

# Changelog

## 0.7.3-dev — 2026-09-16

- Doctor: netductor-redirect + :80 /healthz
- redirect-serve: optional HTTPS (-tls-cert/-tls-key)
- Secondary provision: per-node mTLS client material
- Alerts: secondary offline; watch netductor-redirect
- netductor backup verify|list; docs VPS-REINSTALL.md
- CI: redirect allowlist/encoding tests


- **TG Access (Bot API 10.x rich):** QR via `sendRichMessage` + `tg://photo?id=`; URI in `<pre><code>`; mode switch VLESS/Core/HY2 in body; nav under message ([docs/TG-UI.md](docs/TG-UI.md)).
- **Import deep-links:** Telegram rejects custom schemes in url-buttons. Solution: `netductor redirect-serve` on **:80** (`GET /r?u=<base64url(deep-link)>` → 302). Allowlist: shadowrocket/happ/incy/vless/hy2/ss/trojan.
- Unit: `netductor-redirect.service`; bot env `NETDUCTOR_REDIRECT_BASE` (default `http://<primary-ip>`).
- Access buttons Shadowrocket / Happ / INCY = `type=url` → redirect base (one-tap open client on phone).
- Link generation: `coreAdvertiseHost` vs `vpnAdvertiseHost` (domain-aware); bot `shareURIFrom` + stdout-only `runVPN`.

## 0.7.2-dev

## 0.7.2-dev

- TG: split format.go → format_status/vpn/nodes/sites
- mTLS: `mtls issue-client <id>` per-node client certs under `secrets/mtls/clients/`
- TLS: `netductor tls self-signed` for optional HTTPS admin lab certs

## 0.7.2-dev — 2026-09-15

- Security/refactor follow-up: single devices path, secondary-* ids, mTLS DNS SAN rotate
- Secrets `secondary_*` (+ relay_* fallback); VPN links via `NETDUCTOR_VPN_HOST`
- Split `serve.go` → `api_edge` / `api_session` / `api_vpn_http` / `api_secondary`
- sing-box config write always mode 600

## 0.7.1 —
 2026-09-15

- Release: agent-plane **mTLS :8789**, secondary naming, security doctor
- Package rename: `internal/relay` → `internal/secondary` (state path migrates `relay/` → `secondary/`)
- `internal/singboxconfig` schema marker + helpers
- CI: golangci-lint + govulncheck + gosec
- G7 paths: bins `/usr/local/bin`; `/opt/netductor` data-only
- UFW: agent ports limited to secondary IP

## 0.7.1-dev — 2026-09-15

- **Naming**: operator-facing `relay` → `secondary` (CLI, role, hostname, docs)
- CLI: `netductor secondary` (alias `relay`)
- Agent API: `/api/secondary/agent/*` (+ legacy `/api/relay/agent/*`)
- Unit: `netductor-secondary-agent.service` (+ legacy)
- Doctor: role-aware; security checks (SSH, X11, zabbix, blocky bind, API bind, config perms); subscription check removed
- Agent: `restart:` allowlist (sing-box, netductor-*, blocky only)
- SSH harden: X11Forwarding no
- Blocky default: listen 127.0.0.1:53 / 127.0.0.1:4000
- CI: go test/build + govulncheck + gosec

## 0.7.0-dev — 2026-09-09 (Netductor)

- **Rename** product/repo: Netductor → **Netductor** (`PavelNeyman/netductor`)
- **Go orchestrator** `netductor`: version, doctor, vpn, edge, serve, install, probe
- **Go API**: edge, session, metrics, live probes, VPN users, admin SPA; proxy to Python
- **Units**: `netductor-api` (parallel :8790 / cutover :8787)
- **Agent**: `netductor-agent` (OpenWrt outbound)
- **Telegram**: `netductor-tg` + unit `netductor-telegram-bot`
- **G7 paths**: dual layout `/opt|/etc|/var/lib/netductor` → legacy symlinks
- Cleanup: prefer Go bots over `cmd/netductor-tg` and bash `bot.sh`

## 0.5.4 — 2026-09-08

- Hardening: blacklist virtio_gpu (headless KVM soft lockup mitigation)
- Hardening: 2G /swapfile by default (NETDUCTOR_SWAP_MB, NETDUCTOR_SKIP_SWAP=1)
- Kuma/Beszel optional off by default


## 0.5.2 — 2026-09-08

- Admin UI: auto-refresh, probe strip, session expiry, toasts, note edit, search, copy VLESS/HY2
- Metrics: net counters, probe 24h uptime %, Settings for alerts/probes JSON
- Alerts: recovery messages; UDP hy2; flock + cooldown before notify
- API: `/api/session`, `/vpn/users/:name/note`, `/api/probes/uptime`, POST probes config
- CLI: `netductor vpn note`
- Doctor: admin UI + metrics timer checks

## 0.5.1 — probes + charts + UDP hy2 fix
## 0.5.0 — Admin UI + self metrics (no Prometheus); Kuma/Beszel off by default
## 0.4.x — install/prepare, telegram bot, uninstall tiers
