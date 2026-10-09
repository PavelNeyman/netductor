**EN** · [RU](ru/PLAN-NVR-IMPLEMENTATION.md)

# PLAN: NVR implementation (next steps)

**Status:** active — Phase 1 started in code (2026-10-08)
**Upstream primary:** [pytapo](https://github.com/JurajNyiri/pytapo) + [freeKC/tapo-v4-protocol](https://github.com/freeKC/tapo-v4-protocol) (MIT, PROTOCOL.md)  
**Baseline:** MVP marked done ~0.7.32–0.7.33 in [PLAN-NVR-TAPO.md](PLAN-NVR-TAPO.md); product code still on primary (`internal/nvr`, `internal/tapo`, TG/Web/API).  
**Upstream watch:** [pytapo](https://github.com/JurajNyiri/pytapo) **3.4.26** (2026-09-28), [HomeAssistant-Tapo-Control](https://github.com/JurajNyiri/HomeAssistant-Tapo-Control) **7.2.x** (pytapo pin 3.4.26).

Locked product rules unchanged: **VPN-only** access; archive on **primary**; edge = ephemeral buffer; no public RTSP/go2rtc.

---

## 1. Where we are (netductor)

| Area | State |
|------|--------|
| Camera inventory, leases, DHCP static | Done |
| Agent record (`-c copy`) + upload + primary ingest | Done |
| Retention / storage backends `local` \| `local_encrypted` \| `nfs` | Done (operator mounts LUKS/NFS) |
| go2rtc YAML gen (127.0.0.1) | Done — **binary install operator** |
| Clip tokens / TG segments/events | Done |
| Motion schedule + zones schema | Done (CV optional later) |
| PTZ / night / privacy via `internal/tapo` (secure + KLAP v1/v2) | Done |
| Agent order: tapo-go → python pytapo → ONVIF | Done |
| **Two-way audio** (`tapo://` in go2rtc) | **Planned only** |
| **Hardware e2e** (C200 + Cudy + primary) | **OPEN** |
| **TPAP / SPAKE2+** transport | **Missing** (critical for new FW) |

---

## 2. Upstream delta (pytapo / HA Tapo-Control) — what matters for us

### Must track (control path / login)

| Upstream | Why |
|----------|-----|
| **TPAP transport + SPAKE2+ login** (pytapo **3.4.20+**) | New camera login path; without it newer FW may reject classic/KLAP-only clients |
| Safer **direct stream** handling + snapshot index fix (**3.4.22**) | If we ever touch direct/media path or python fallback scripts |
| **TPAP SHA256** encryption method for recordings (**3.4.25**) | Only if we download SD-card recordings via cloud API |

### Nice to have (not core NVR)

| Upstream | Netductor relevance |
|----------|---------------------|
| Faster SD recording download (camera download request, 8–10×) | Optional “pull SD clips to primary” feature — **not** continuous RTSP archive |
| Camera event **thumbnails** without full download (HA 7.2) | Nice for TG event cards; needs control API + storage |
| Partial **setRecordPlan** day updates (**3.4.23**) | Only if we manage on-camera record schedule (today we record on primary) |
| Multi-lens / dual-cam entities (HA 7.1+) | Later if multi-lens Tapo models |
| Media Sync recovery, map entity, battery poll | HA-specific — **skip** |

### Explicitly out of scope (still)

- Becoming a full HA/Frigate replacement  
- Public WebRTC / public go2rtc  
- AI detect (Frigate/Coral) unless separate decision  
- Relying on HA as runtime dependency  

### go2rtc two-way audio (unchanged industry path)

```yaml
streams:
  cam1:
    - rtsp://USER:PASS@IP:554/stream1
    - ffmpeg:cam1#audio=aac
    - ffmpeg:cam1#audio=opus
    - tapo://CLOUD_PASSWORD@IP   # talk-back; cloud pass ≠ Camera Account
```

Needs: go2rtc with WebRTC, HTTPS for browser mic, Third-Party Compatibility, firmware luck. Netductor only generates config + secrets; **no** public bind.

---

## 3. Implementation phases

### Phase 0 — Operator readiness (no big code)

1. Confirm C200: Third-Party Compatibility **On**, Camera Account, RTSP `stream1`/`stream2`.  
2. Install **go2rtc** binary on primary (or LAN host); enable unit from `deploy/go2rtc.service.example`.  
3. Unlock NVR volume if `local_encrypted`; set `nvr.storage.*`.  
4. One camera end-to-end: lease → add cam → record → segment list → clip in TG.  
5. Document failures in `docs/OPENWRT-LAB.md` / NVR runbook.

**Exit:** one camera continuous record visible in TG + disk under retention.

### Phase 1 — TPAP / login resilience (code, high priority)

Goal: control plane works on **new Tapo FW** without forcing python-only.

| Task | Detail |
|------|--------|
| Research | Port minimal TPAP/SPAKE2+ from pytapo 3.4.20+ (or call python-kasa) into `internal/tapo` |
| Login order | **TPAP → KLAP v2/v1 → secure classic → legacy** |
| Fallback | Keep agent `tapo-go → pytapo script → ONVIF` |
| Tests | Unit tests with recorded handshakes if possible; live C200 matrix |
| Docs | PLAN-NVR-TAPO “transport” section + FW notes |

**Exit:** `netductor nvr tapo <host> … getBasicInfo` succeeds on FW that requires TPAP.

### Phase 2 — Live + talk (config, medium)

| Task | Detail |
|------|--------|
| go2rtc yaml | Optional `tapo://` source per camera; cloud password in secrets (`nvr.tapo_cloud_password` / per-cam) |
| Bind | 127.0.0.1 + VPN path only; document WebRTC ICE via VPN IP |
| UI | TG/Web: “Live” link (VPN), “Talk” opens go2rtc UI or notes HTTPS requirement |
| Audio one-way | Ensure RTSP path includes audio when cam supports it (`-c copy`) |

**Exit:** operator can view live over VPN; talk documented and optional.

### Phase 3 — Hardening & product polish

| Task | Detail |
|------|--------|
| Retention alerts | TG when disk > threshold / retention purge summary |
| Doctor | `nvr` checks: path writable, go2rtc active, last segment age, cam RTSP probe |
| Ingest reliability | Retry/backoff already partial — review after OOM-class incidents |
| Multi-cam | 2× C200 defaults (substream on Wi‑Fi) |
| Web thin UI | Camera card parity with TG (already partially in Control) |

### Phase 4 — Optional SD-card pull (upstream-inspired)

Only if primary continuous RTSP is not enough (gaps, offline cam):

- List recordings / thumbnails via control API (TPAP-capable)  
- Fast download path (pytapo download request) **or** python helper  
- Import into same segment store + retention  

**Defer** until Phase 0–1 proven on hardware.

### Phase 5 — Non-goals until requested

- Frigate AI  
- Site-rooms integration of cameras  
- Public sharing links  
- Battery/solar Tapo direct-stream primary path  

---

## 4. Suggested order of work (sprints)

```text
Sprint A (ops):     Phase 0 hardware smoke on one C200
Sprint B (code):    Phase 1 TPAP in internal/tapo + tests
Sprint C (config):  Phase 2 go2rtc tapo:// + secrets + UI hints
Sprint D (polish):  Phase 3 doctor + retention alerts
Later:              Phase 4 SD pull if needed
```

---

## 5. Risks

| Risk | Mitigation |
|------|------------|
| New FW breaks KLAP-only | Phase 1 TPAP; python pytapo as safety net |
| Cudy RAM /tmp overflow | USB NVR_DIR; substream; 24MB tmpfs cap |
| Cloud password for talk | secrets 0600; never in git; document ≠ Camera Account |
| Primary disk | retention days/GB + encrypted volume |
| OOM (history: Lampac) | NVR ffmpeg/go2rtc MemoryMax; no Chrome on primary |

---

## 6. Acceptance (product)

- [ ] Two C200 continuous record to primary over VPN path  
- [ ] PTZ/night/privacy via netductor (Go or fallback) after any FW that needs TPAP  
- [ ] Live view over VPN only  
- [ ] Clip to TG works  
- [x] Doctor surfaces NVR path / stale segments / go2rtc listen (0.9.306)  
- [x] Optional talk documented; no public ports (go2rtc tapo:// + PLAN)  

---

## 7. References

- [PLAN-NVR-TAPO.md](PLAN-NVR-TAPO.md) — original design lock  
- pytapo releases: https://github.com/JurajNyiri/pytapo/releases (3.4.20 TPAP; 3.4.25–26 fixes)  
- HA Tapo-Control 7.2.x: recording previews, media sync, pytapo 3.4.22+  
- go2rtc Tapo talk: community `tapo://` source (AlexxIT)  


## Implementation log

### 2026-10-08
- Vendored **freeKC** `tapo_v4` under `scripts/tapo_v4/` (MIT) + `cli.py` helper.
- `internal/tapo`: `CloudPassword`, Login order **KLAP → TPAP (if cloud) → classic**, TPAP fallback on confirm/stok fail.
- TPAP **pure Go** SPAKE2+ + AES-CCM (no Python runtime).
- Extended actions: person detect, smart track, OSD, flip, alarm test, record plan, daynight get.
- CLI: `TAPO_CLOUD_PASSWORD=… netductor nvr tapo …`; action `tpap_ping`.
- go2rtc: ffmpeg aac/opus + optional `tapo://admin:SHA256UP@ip` when secret `<ref>_cloud` or `tapo_cloud` set.
- Still need: media 8800 SD pull, hardware e2e, optional native media path.
- Removed Python runtime dependency for TPAP (scripts/tapo_v4 = reference only).

### 2026-10-08 (later)
- Replaced Python TPAP helper with pure Go in `internal/tapo/tpap.go` + `ccm.go`.
- `scripts/tapo_v4` kept as protocol reference only.

### 2026-10-08 UI surface (no new product invent)
- opcatalog/Web: cameras add/delete, record start/stop, PTZ, clip token, motion (existing API only).
- Camera POST accepts `cloud_password` → secret `<id>_cloud` for TPAP/talk.
- TG: Live/Talk help (VPN-only go2rtc), existing cam probe/record/PTZ kept.
- TUI: cameras list + storage shortcuts.
- **Not in repo as detailed UX:** full camera card SPA, embedded WebRTC player, multi-step add wizard beyond DHCP lease → RTSP password. Discuss if needed.
