# VPS reinstall checklist (primary / secondary)

Use after a clean OS image. Password/SSH key are temporary until harden finishes.

## Primary (abroad)

1. SSH key-only; disable password auth (`netductor install` / ssh harden).
2. Install netductor release binary → `/usr/local/bin` + `/opt/netductor/bin`.
3. `netductor mtls ensure` (agent plane CA/server/client).
4. Restore secrets/users from `.ndenc` if recovering: `netductor restore <file>` (see BACKUP.md).
5. Bring up: sing-box, blocky (localhost), `netductor-api`, `netductor-telegram-bot`.
6. **Import redirect:** enable `netductor-redirect.service` (`redirect-serve -listen :80`).
7. Bot drop-in: `NETDUCTOR_REDIRECT_BASE=http://<primary-ip-or-domain>`.
8. UFW/nft: 22, 443, 80 (ACME+redirect), agent **8789 only from secondary IP**, UDP/TCP VPN ports as needed. No public Blocky/API.
9. Domains: apex → primary; `vpn.` → secondary (if used).
10. `netductor doctor` — 0 FAIL; redirect `/healthz` OK.
11. TG Access: QR + SR/Happ/INCY url buttons open clients.

## Secondary (RU)

1. Prefer `netductor fleet provision-secondary` / `secondary provision` from primary (bundle + agent token + **per-node mTLS client cert**).
2. Hostname `nd-secondary`; role secondary; TG bot **standby**.
3. Heartbeat to primary `:8789` (mTLS) or legacy `:8788` only if needed.
4. Remove hoster zabbix agent if present.
5. `netductor doctor` role=secondary.

## After both up

- Refresh client links; confirm core vs secondary pbk/sid not mixed.
- `netductor backup now` + `netductor backup verify`.
- Optional: set `NETDUCTOR_REDIRECT_BASE` to domain when DNS is stable.
