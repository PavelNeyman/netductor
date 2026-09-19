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
