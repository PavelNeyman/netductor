# Recover drill (primary from secondary)

**Goal:** prove backup → wipe primary → recover without permanent data loss.

## Preconditions
- Secondary online; agent heartbeat OK
- Offsite/secondary holds latest encrypted backup + recovery material
- Operator Mac has SSH key for primary (`~/.ssh/netductor_primary`)
- Note `NETDUCTOR_OPERATOR_PUBKEY` if using recover env inject

## Steps (operator)

1. **Snapshot / backup now** on primary  
   `netductor backup now` (or TG Backup → run)  
   Confirm peer/agent pull on secondary if configured.

2. **Record versions**  
   `netductor version` · secondary agent version · domain/SNI.

3. **Provider wipe** primary VPS (reinstall OS). Fresh root password.

4. **Bootstrap binary** (A6 path)  
   ```bash
   curl -fsSL -o /usr/local/bin/netductor \
     https://github.com/PavelNeyman/netductor/releases/download/vVERSION/netductor-linux-amd64
   chmod 755 /usr/local/bin/netductor
   ```

5. **Recover** (from Mac or console)  
   Prefer documented `netductor recover` / `restore` with key + archive from secondary.  
   Inject operator pubkey **before** harden if required by current recover path.

6. **Verify**  
   `netductor doctor` · TG bot · secondary heartbeat · VPN client · domain :8443 `/healthz`.

7. **Mark drill** in OPEN_ITEMS / handoff with date.

## Failure notes
- If recover locks SSH: provider console + pubkey append (see handoff recover notes).
- Binaries always from Release by component list — not from backup archives.
