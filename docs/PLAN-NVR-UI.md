**EN** · NVR UI parity (locked 2026-10-08)

# PLAN: NVR UI (table, wizard, live, TG cards)

Agreed with operator. Backend unchanged in spirit: **API-first**, thin UIs.

## Surfaces

| UI | Cameras | Live |
|--|--|--|
| **Web** | Table + add form + wizard (leases → pick → RTSP/cloud → probe → record) | Player via **API stream proxy** (`/api/nvr/stream?id=`) same session/VPN tunnel — not public go2rtc |
| **TG** | List → **per-camera card** (probe/record/stop/PTZ/Live link/clip) | Same live URL text (open in browser on VPN) |
| **TUI** | List / existing wizard | Print live URL |
| **go2rtc** | 127.0.0.1 only | Web reaches it only through node API proxy |

## Live URL shape (all UIs)

- JSON: `GET /api/nvr/live?id=CAM` → `{ web_path, stream_path, rtsp_hint, go2rtc }`
- Stream: `GET /api/nvr/stream?id=CAM` (session) → proxy to local go2rtc
- Never put RTSP password in TG messages

## Out of scope here

- Public WebRTC, embedded TG video, Frigate UI clone

## Done 2026-10-09
- `/api/nvr/site/leases` waits by default (`wait:true`), returns parsed `leases[]` for autofill.
- Web Control + Installer NVR: table, add form, wizard with **Use** on lease rows, live play.
- TG already had wait+parse for DHCP (cards for cameras).
