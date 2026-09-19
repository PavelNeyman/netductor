# Передача контекста (0.7.3-dev)

Полный EN: [AGENT_HANDOFF.md](../AGENT_HANDOFF.md)

- primary / secondary, redirect :80, mTLS :8789, netductor.conf
- TG: nav под сообщением, действия в тексте; Access in-place edit
- Корп OpenConnect + наш full-tunnel на одном устройстве часто роняют весь интернет
- Probes: api-health должен ходить на https localhost или tcp :8787

### NVR CLI (0.7.16-dev)

```
netductor nvr config show|set retention_days=7 max_gb=40 min_free_gb=5
netductor nvr cameras list|add name=… site=… mac=… ip=… password=…
netductor nvr leases <device_id>
netductor nvr retention
netductor nvr prepare-storage
```

API: `/api/nvr/config`, `/api/nvr/retention/run`, `/api/nvr/cameras`, …  
Plan: [PLAN-NVR-TAPO.md](PLAN-NVR-TAPO.md). Background retention on `serve`.

### NVR 0.7.17-dev

- TG: Инструменты → NVR; CLI leases ждёт ответ агента; doctor — секция NVR.

### NVR 0.7.18-dev

- TG: **From leases** → edge site → async wait → add camera (password prompt) + optional static DHCP
- `nvr storage` / `GET /api/nvr/storage`

### NVR 0.7.19-dev

- Agent: `rtsp_probe` (TCP + optional ffprobe)
- CLI: `netductor nvr probe <camera_id>`
- Ingest: `POST /api/nvr/ingest` (multipart `file` + `camera_id`) for site→primary segment push

### NVR 0.7.20-dev

- Agent records on LAN (`ffmpeg` segments in `/tmp/netductor-nvr`) and uploads to primary ingest
- CLI: `netductor nvr record start|stop <id>`
- TG: Cameras → 🔍/⏺/⏹ per camera
- Doctor: mountpoint / encryption hint

### Cudy / edge load (0.7.21)

- **Preferred:** record on site with `-c copy` (no re-encode). CPU stays low when stream is healthy.
- **Offline camera:** agent does **TCP probe first**; exponential backoff 5s→5m — no tight ffmpeg restart loop.
- **Limits:** max **2** concurrent cameras per agent; **/tmp** NVR cap **~200MB** (oldest segments dropped).
- **ffprobe:** 8s timeout kill.
- Cudy TR1200-class devices are fine for 1–2 substreams copy; avoid full HD×N + encode on-router.
- Primary `StartRecorder` does **not** auto-restart (prevents CPU spin if URL unreachable from VPS).
### NVR 0.7.22-dev

- Motion schedule (not CV yet): `nvr motion set enabled=true timezone=Europe/Moscow`
- Events JSONL on segment ingest; TG Events/Motion
- API `/api/nvr/motion`, `/api/nvr/events`

### NVR storage (0.7.23)

- **Archive on primary only.** Cudy: `/tmp` buffer **≤64MB**, upload→delete.
- TG alert on segment if motion schedule allows (`AlertOnSegment`).

### NVR + Cudy storage (0.7.24-dev) — handoff (ru)

**Где лежит видео**

1. **Primary** — постоянный архив (`/var/lib/netductor/nvr/…`), retention, encrypt.
2. **OpenWrt agent** — короткий буфер → `POST /api/nvr/ingest` → delete local file.

**Ограничения Cudy**

- Flash **16MB**: never NVR path.
- RAM **128MB**: default buffer **`NVR_MAX_MB=24`** on `/tmp` (tmpfs).
- Optional **USB**: mount + `NVR_DIR=/mnt/…/netductor-nvr`, raise `NVR_MAX_MB` (e.g. 512).
- **LTE modem SD**: only if `ls /dev/sd*` shows it; many modems hide SD from OpenWrt.

**Agent config keys:** `NVR_DIR`, `NVR_MAX_MB` (also env `NETDUCTOR_NVR_DIR`, `NETDUCTOR_NVR_MAX_MB`).

**Safety:** max **2** concurrent cameras; TCP pre-check + backoff; no tight ffmpeg restart.

**CLI (primary):** `nvr cameras|leases|probe|record|motion|events|status|prepare-storage`  
**TG:** Tools → NVR (leases, cameras P/R/S, motion, events)  
**Plan:** [PLAN-NVR-TAPO.md](PLAN-NVR-TAPO.md) · [EDGE-AGENT.md](EDGE-AGENT.md)

### NVR backlog vs done (0.7.25-dev)

**Done:** inventory, leases→camera, agent record→ingest, retention, motion schedule, events, clip one-shot tokens, NVR_DIR/MAX_MB, TG hub.

**Not done (no deploy required to code later):** go2rtc live UI, PTZ/night API, CV zones, auto LUKS unlock, NFS home backend automation, Admin web NVR page, TUI NVR wizard.

**Cudy:** flash 16MB unused for video; RAM tmpfs buffer default 24MB; USB via NVR_DIR.

### Tapo C200
- RTSP Camera Account, stream1/stream2; ONVIF :2020 PTZ best-effort (не HA-плагины).
- Ночь/ИК на камере; go2rtc опционально localhost.
