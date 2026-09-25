# NVR / Tapo — code review (0.7.33-dev)

Date: 2026-09-19. Scope: `internal/nvr`, `internal/tapo`, agent NVR cmds, TG/API/CLI/TUI hooks.

## Verdict

**MVP is complete in code** for inventory → record path → retention → control (C200) → UI hooks.  
Remaining risk is **integration on real Cudy + C200 firmware**, not missing packages.

## Architecture

| Layer | Role | Notes |
|-------|------|--------|
| Edge agent | leases, static DHCP, RTSP probe, segment buffer, `camera_ptz` | Runs on LAN; must reach cameras |
| Primary | camera registry, secrets, ingest, retention, motion schedule, clip tokens | VPN-only admin/API |
| `internal/tapo` | pytapo-compatible control (secure + KLAP) | Prefer over ONVIF for C200 |
| go2rtc | optional live | YAML only; binary external |

## Strengths

1. Clear split edge vs primary; agent does not need full nvr state.
2. Retention multi-threshold (days / max GB / min free).
3. Clip tokens are short-lived; path check uses `PathUnderRoot` (not naive prefix).
4. Tapo stack: secure login + KLAP v1/v2 + motor/presets/motion/alarm/ChildID.
5. TG flow async with `WaitCmdResult`; cameras cache in chat state.
6. Storage backends declared: local / local_encrypted / nfs.

## Issues / risks (severity)

### High (ops, not bugs)

- **C200 requires** Tapo Lab → Third-Party Compatibility + Camera Account. Without this, tapo-go and HA fail alike.
- **Cudy 16MB flash**: recording buffer must be USB/tmpfs (`NVR_DIR`); flash is not viable.
- **go2rtc** not installed by netductor; only config generation.

### Medium

- **KLAP vs classic**: auto-fallback exists; wrong port/http vs https may need tuning per FW.
- **Secrets** in state files (0600): OK for single-tenant VPS; no HSM.
- **Primary ffmpeg recorder** optional vs edge-push: two paths — document which is default in production (edge-push preferred for NAT).
- **Motion zones** stored only; no CV pipeline (by design).

### Low

- TG camera UI is dense (many buttons per cam); fine for ≤10 cameras.
- API PTZ returns `cmd_id` only; client must poll edge result if needed.
- `InsecureSkipVerify` on camera TLS: expected for local cams.

## Test coverage

- Unit: retention, tokens, leases parse, pathsafe, tapo crypto helpers.
- No integration tests against real camera (expected).
- `go test` packages: nvr, tapo, netductor, netductor-tg, agent — green at review time.

## Security checklist

- [x] Clip path confinement (`PathUnderRoot`)
- [x] Session required on mutating `/api/nvr/*` (clip GET is token-gated)
- [x] go2rtc listen 127.0.0.1 in generated YAML
- [x] VPN-only product assumption for admin/live
- [ ] Encrypt-at-rest automation still operator-driven (gocryptfs/LUKS prepare helpers exist)

## Follow-ups (non-blocking)

1. Live drill: two C200 + Cudy enroll → static DHCP → record → TG PTZ.
2. Optional: poll edge cmd result on API PTZ.
3. Optional: Frigate as addon if AI motion wanted.

