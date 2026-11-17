# Runbook — Control plane cutover (active → standby)

**Scope:** Phase 3 active/standby. Not multi-master. No automatic promote.

## Preconditions

| Check | Command / action |
|-------|------------------|
| Standby role | `control_role=standby` on target |
| Fresh inbox | `netductor control sync-pull` (or timer) succeeded recently |
| Backup key offline | operator has `backup_key` / `NETDUCTOR_BACKUP_KEY` |
| Agent list | edge/secondary agents have `SERVERS=` including standby URL |
| DNS/docs | know which names point at control API / redirect |

## Cutover steps

1. **Confirm active is dead or fenced** (power off, or network isolate). Do not leave two writers.
2. On **standby**:
   ```bash
   netductor control restore-inbox --key "$BACKUP_KEY" --confirm
   netductor control sot-check
   netductor control promote --confirm
   ```
3. **Fence old active** (if still up):
   ```bash
   netductor control set-role standby --confirm
   # or power off
   ```
4. **DNS / clients**
   - Point control/API hostnames at the new active public IP (or update operator docs).
   - Redirect/`i.*` LE certs: re-issue on new active if names moved (`tlsle` / domain flow).
   - VPN **egress** IPs usually **unchanged** (secondaries); only control management plane moves.
5. **Agents**
   - With `SERVER` + `SERVERS`, agents should stick to the working base after promote accepts writes.
   - If sticky still on dead host, restart agent or wait for next enroll/heartbeat cycle.
6. **Verify**
   ```bash
   netductor doctor | grep -E 'control_role|READY|fail='
   netductor secondary status
   curl -fsS https://NEW/api/control/status
   ```

## Rollback

Only if new active is wrong and old active still has intact disk:

1. New node: `set-role standby --confirm` (or offline).
2. Old node: `promote --confirm` only if SoT there is still authoritative.
3. Revert DNS.

## Non-goals

- Automatic promote on failed health probes.
- Dual live writers.
- Changing user VLESS endpoints solely because control moved (egress peers stay).
