# Recover drill (primary from secondary)

**EN** · [RU](ru/RECOVER-DRILL.md)

**Status:** live unattended drill **passed** 2026-09-27 on **v0.9.71**.

**Goal:** backup → wipe primary → recover without permanent data loss and **without manual post-fix**.

## Preconditions
- Secondary online; latest `.ndenc` under `peers/core/`
- `recovery_token` on secondary; operator has `backup_key`
- Mac key `~/.ssh/netductor_primary` (pubkey injected early on recover)
- Node binary from GitHub Release (**not** from backup)

## Operator steps

1. **Backup** on primary: `netductor backup now` — check `COMPONENTS.txt` includes baseline (dirs…telegram…backup) + optionals.
2. Confirm secondary has the new archive (agent pull or copy).
3. **Wipe** primary OS; note root password.
4. On fresh primary:
   ```bash
   curl -fsSL -o /usr/local/bin/netductor \
     https://github.com/PavelNeyman/netductor/releases/download/vVERSION/netductor-linux-amd64
   chmod 755 /usr/local/bin/netductor
   ```
5. On secondary: `netductor recovery arm --ttl 2h` (background/`nohup` if needed).
6. Recover:
   ```bash
   export NETDUCTOR_OPERATOR_PUBKEY='ssh-ed25519 AAAA… netductor-primary'
   netductor recover --from-secondary https://SECONDARY:8790 \
     --recovery-token TOKEN --key BACKUP_KEY
   ```
7. **Verify (no manual install):**
   - `netductor doctor` → fail=0
   - `systemctl is-active netductor-api sing-box blocky netductor-telegram-bot`
   - real binary `/usr/local/bin/netductor-tg` (not self-symlink)
   - `ufw status` allows secondary IP on **:8789**
   - `netductor secondary status` → online
   - SSH day-2: `-p 52222` + Mac key

8. Secondary: disarm recovery (`pkill` arm / wait TTL); `:8790` closed.

## How recover works (0.9.71+)

| Phase | What |
|-------|------|
| Pre-restore install | Full component list; `vpn-users` may soft-fail (no secrets yet) → **continuing** |
| Tar restore | Secrets, users, state |
| Post-restore pass | api / telegram / backup / vpn-users again with secrets |
| UFW | Secondary public IPs → `api-allow.cidr` + ApplyAgentFirewall |

Expected log shape:
```
component vpn-users: … (continuing)
==> api / telegram / backup
recover: post-restore component pass
recover: done
```

## Failure notes
- SSH lockout: provider console + pubkey (should be rare if early inject works).
- Binaries always from Release by component — not from `.ndenc`.


## Post-recover Reality / uplink checklist (required)

After wipe+recover, **do not assume VPN works** until:

1. `netductor vpn apply` succeeds (primary sing-box conf **from secrets** + registry users including **relay-uplink**).
2. Primary inbound Reality: `private_key` / `short_id` / `server_name` match `/etc/netductor/secrets/singbox_*`.
3. Secondary uplink: `public_key`+`short_id` = primary **public** secrets; SNI same as primary handshake dest.
4. **Multiplex stays ON** (canon for disconnect mitigation):
   - **Primary inbound** `vless-reality.multiplex.enabled` (fields: `enabled`, `padding` only — no `max_connections` on inbound; sing-box 1.14 rejects them).
   - **Secondary uplink outbound** multiplex via `uplinkMultiplexObject()` (`max_connections` OK on outbound).
5. Client links: secondary `pbk`/`sid` from agent heartbeat (`devices.json`), not a stale pre-recover QR.

**Incident 2026-09-27:** recover left empty inbound users + conf keys ≠ secrets + apply failed on invalid inbound mux fields → uplink `unknown UUID` / x509. Fixed by apply schema + ensure-relay-uplink in post-restore.



## LE after recover (0.9.74+)

Certs are **not** in `.ndenc`. If conf has `DOMAIN` + `LE_EMAIL`, recover re-runs LE + redirect unit. Otherwise set once: `netductor domain set --base … --le --email …`.

Post-restore **sanitize conf** strips footgun keys (PLAIN_AGENT, ALLOW_PUBLIC, LEGACY_ADMIN, …).
