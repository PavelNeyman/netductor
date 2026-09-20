## 0.8.1 — 2026-09-20

### Security hardening
- Edge recovery: prefer private IP bind, LAN-only clients, optional SERVER_PIN
- Self-update verifies release SHA256SUMS
- TG: edge approve/deny callbacks; mask device_token in chat
- serve: EnsureAll mTLS before agent plane; SNI fallback api.vk.me
- Doctor warns on CLAIM_FIRST

## 0.8.0 — 2026-09-20

### Highlights
- **Secondary-only** RU entry (legacy relay naming removed)
- Edge **LAN recovery** (`:7879/netductor-recovery`), recovery codes, register/set-site
- NVR/Tapo Go port foundations; RU-direct for gosuslugi stack
- Primary self-update from **GitHub Releases** (TG Tools → Updates); agents **manual** `agent_update`
- Docs EN+RU parity pass; review `docs/REVIEW-2026-09-20.md`

### Breaking / cleanup
- CLI `relay` alias removed → `secondary`
- State path only `secondary/`
- API only `/api/secondary/*`

## 0.7.39-dev

- UI: edge recovery/register/set-site/export in TG, Admin, TUI + API

## 0.7.38-dev

- Edge recovery codes + LAN page :7879; register/export/import/set-site; CONTROL_ONLY

## 0.7.37-dev

- Rename remaining Relay* identifiers to Secondary*; docs EDGE-REINSTALL

## 0.7.36-dev

- Remove legacy relay naming: paths, CLI, API, roles — secondary only

## 0.7.35-dev

- SecondaryDir helper; no lab IP in TUI placeholders; client RU tests; release workflow_dispatch; doctor REDIRECT_BASE warn

## 0.7.34-dev (docs)

- REVIEW-2026-09-19, OPEN_ITEMS/handoff snapshot after RU-direct deploy

## 0.7.34-dev

- Client/relay RU-direct lists for Gosuslugi/gov/banks; SR nd-oc + routing.conf; docs

## 0.7.33-dev

- NVR polish: pathsafe clips, zones API, TUI nvr status/go2rtc; code review doc

## 0.7.32-dev

- NVR MVP close: storage backends, motion zones, go2rtc example, TG PTZ extras

## 0.7.31-dev

- tapo: motion/privacy get, alarm, smart_track, hub ChildID; CLI nvr tapo

## 0.7.30-dev

- tapo: KLAP v1/v2 transport; alarm/reboot; login fallback chain

## 0.7.29-dev

- tapo: presets, calibrate, relativeMove, LED, info; expand Control actions

## 0.7.28-dev

- internal/tapo: Go port of pytapo secure control (C200 motor/night/privacy)
- agent PTZ prefers tapo-go over python/ONVIF

## 0.7.27-dev

- Tapo C200 PTZ via pytapo (HA Tapo-Control path); ONVIF fallback
- scripts/tapo_control.py: move/night/privacy; Third-Party Compatibility docs

## 0.7.26-dev

- Tapo C200: ONVIF PTZ (port 2020), model defaults, go2rtc yaml gen
- TG/API/CLI PTZ; docs without HA plugin dependency

## 0.7.25-dev

- one-shot clip tokens `/api/nvr/clip`; CLI token; agent nvr_disk_info
- stream2 when StreamSub; motion windows= JSON; plan/handoff status matrix

## 0.7.24-dev

- agent NVR_DIR / NVR_MAX_MB (default 24MB tmpfs); USB buffer docs; handoff Cudy 16MB/128MB

## 0.7.23-dev

- Edge NVR buffer 64MB, faster upload; docs: archive only on primary
- TG AlertOnce on segment during motion window

## 0.7.22-dev

- NVR motion schedule (windows/tz), events log on ingest
- CLI/API/TG: motion, events, status; prepare-storage storage snapshot

## 0.7.21-dev

- agent NVR supervise: TCP pre-check, backoff, max 2 cams, /tmp 200MB trim
- ffprobe timeout; primary recorder no auto-restart; nvr_record_status

## 0.7.20-dev

- agent nvr_record_start/stop + segment upload to /api/nvr/ingest
- CLI nvr record start|stop; TG Cameras probe/rec/stop
- storage MountPoint + encrypt hint in doctor

## 0.7.19-dev

- agent `rtsp_probe`; CLI `nvr probe <camera_id>`
- `POST /api/nvr/ingest` segment upload (device token or session)
- doctor storage status; IngestSegment tests

## 0.7.18-dev

- TG NVR: From leases → site → wait → pick → RTSP password → camera + dhcp_static
- ParseLeasesResult (agent JSON); storage status API/CLI; GetStorageStatus

## 0.7.17-dev

- TG NVR hub under Tools
- edge.WaitCmdResult; nvr leases/wifi-clients wait for agent
- doctor NVR section

## 0.7.16-dev

- CLI `netductor nvr` (config, cameras, leases, retention, segments, prepare-storage, recorder)
- prepare-storage + gocryptfs/LUKS guidance
- docs/handoff NVR CLI

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
