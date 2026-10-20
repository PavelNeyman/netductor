# Disaster recovery (primary down)

1. Reach **secondary** (SSH port 52222, key-only after harden).
2. Restore primary data from peer copy:
   ```bash
   netductor recover --from-secondary https://SECONDARY:8790 \
     --recovery-token "$TOKEN" --key "$BACKUP_KEY"
   ```
3. Run `netductor doctor`, re-issue LE if needed, `vpn` apply, check SP/PS.
4. Confirm secondary agent heartbeat / uplink.
5. `netductor update apply` if versions drifted.

Encrypted backups land on secondary via agent `backup_pull`. Key is offline (`--key` / `NETDUCTOR_BACKUP_KEY`).

TG: **Menu → 📡 Fleet → DR** checklist.

## Key custody (accepted)

Offsite copies under `peers/core/` on secondary may include `BACKUP_KEY.txt` next to the `.ndenc` archives (mode 0600). At-rest encryption on that host is void: anyone who can read the archive can read the key. This is intentional so DR does not depend on the operator having the key offline. Do not treat secondary disk as a second factor.

