# Network surface (security)

| Port | Bind | Encryption | Notes |
|------|------|------------|--------|
| **443** TCP | `*` | Reality / TLS (sing-box) | Public VPN |
| **4443** | `*` | HY2/TLS as configured | Public VPN alt |
| **52222** TCP | `*` | SSH | Key-only after harden; fail2ban |
| **8789** TCP | `*` | **mTLS** | Agent plane; no useful access without client cert |
| **8790** TCP | `*` when **armed** | HTTP + Bearer | **Off by default.** `recovery arm` or port-knock (41222→41223→41224). Encrypted `.ndenc` only. |
| **8787** | `127.0.0.1` | plain local | Admin API — not on WAN |
| **9118** | `127.0.0.1` | plain local | Lampac — VPN or SSH tunnel only |
| **5000** | `127.0.0.1` | plain local | OCI registry — local only |
| **1984** | `127.0.0.1` | plain local | go2rtc API (if enabled) |
| **8554** | `127.0.0.1` | RTSP local | go2rtc RTSP restream |
| **8555** | `127.0.0.1` | WebRTC local | go2rtc WebRTC — not public WAN |

Camera native RTSP (e.g. Tapo :554) stays on **LAN only**; netductor does not publish it to the internet.
| **53** | `127.0.0.1` | DNS local | blocky |

Do **not** publish 8787/9118/5000 on `0.0.0.0`.

RU/EN: see also `docs/AGENT_HANDOFF.md` agent plane section.
