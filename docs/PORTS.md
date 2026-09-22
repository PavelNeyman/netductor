# Network surface (security)

| Port | Bind | Encryption | Notes |
|------|------|------------|--------|
| **443** TCP | `*` | Reality / TLS (sing-box) | Public VPN |
| **4443** | `*` | HY2/TLS as configured | Public VPN alt |
| **52222** TCP | `*` | SSH | Key-only after harden; fail2ban |
| **8789** TCP | `*` | **mTLS** | Agent plane; no useful access without client cert |
| **8787** | `127.0.0.1` | plain local | Admin API — not on WAN |
| **9118** | `127.0.0.1` | plain local | Lampac — VPN or SSH tunnel only |
| **5000** | `127.0.0.1` | plain local | OCI registry — local only |
| **53** | `127.0.0.1` | DNS local | blocky |

Do **not** publish 8787/9118/5000 on `0.0.0.0`.

RU/EN: see also `docs/AGENT_HANDOFF.md` agent plane section.
