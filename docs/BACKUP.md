**Baseline: v0.8.49**

# Backups (RU + EN)

## Concept

- Archive holds **configs + data**, not application binaries/images.
- `components.json` / `COMPONENTS.txt` lists what to reinstall on recover.
- `SyncComponentsFromDisk()` runs before each backup: lampac/git/registry appear/disappear with install/uninstall.

## What is packed

`/etc/netductor`, `/etc/blocky`, `/etc/sing-box`, `/var/lib/netductor` (incl. git + registry data), `/opt/netductor/lampac`, profiles.

## Offsite on secondary (RU)

After each successful backup on primary, online secondary agents get `backup_pull`:

1. Agent `GET https://primary:8789/api/secondary/agent/backup/latest` (mTLS + agent token)
2. Stores under `/var/lib/netductor/backups/peers/core/`
3. Also pulls `BACKUP_KEY.txt` + COMPONENTS header

No primary→secondary SSH.

## Disaster recovery (new primary, primary dead)

Secondary agent listens **`:8790` recovery API** (Bearer `recovery_token` from `/etc/netductor/secrets/recovery_token` on secondary).

```bash
# on clean Debian VPS (after placing netductor binary)
netductor recover --from-secondary http://SECONDARY_IP:8790 \
  --recovery-token "$(cat recovery_token)" 
# optional --key if key not served
```

Flow: download .ndenc → install components from manifest → extract data → restart.

Save `recovery_token` offline when secondary is first provisioned (printed in provision log).

## Manual

```bash
netductor backup
netductor recover --key KEY file.ndenc
```

## RU

Бэкап = данные/конфиги. Список компонентов синхронизируется с диском при каждом backup.  
Копия на RU — agent pull по расписанию backup.  
Восстановление primary: `recover --from-secondary` с recovery API secondary `:8790`.


## Recovery server env (secondary)

| Env | Default | Meaning |
|-----|---------|---------|
| `NETDUCTOR_RECOVERY_BIND` | `0.0.0.0` | Listen address |
| `NETDUCTOR_RECOVERY_UFW` | off | If `1`, `ufw allow 8790/tcp` |
| `NETDUCTOR_RECOVERY_ALLOW_CIDR` | empty | Comma-separated IPs allowed (else any + token) |


## SCP peer
**Removed** in 0.8.51. Do not use `backup peer-set` / `backup.offsite`. Offsite is agent `backup_pull` only.


## Hostname on backup/recover (0.8.52+)

- **Backup** writes current hostname to `/etc/netductor/hostname.backup` and aligns `node_id`.
- **Recover** does **not** call auto `nd-core-<ip>`; after data extract applies hostname from backup.
- Fresh install (not recover) still uses `applyHostname` / `NETDUCTOR_HOSTNAME`.

## Operator SSH keys (public only)

- **Never** store private keys in backup.
- **Backup** copies lines from `/root/.ssh/authorized_keys` → `/etc/netductor/operator_authorized_keys` (in tar).
- **Recover** merges into `/root/.ssh/authorized_keys` from:
  1. `operator_authorized_keys` (from backup)
  2. `NETDUCTOR_OPERATOR_PUBKEY` (one line env)
  3. `NETDUCTOR_OPERATOR_PUBKEY_FILE` (path to pubkey file)
- Phone/other device recover without local key still works if pubkey was in backup at last backup time.


## Recover and SSH keys

Backups include **public** operator keys (`/etc/netductor/operator_authorized_keys`).
On `recover`, those keys are **read from the archive before harden**, so password auth can be disabled without lockout.
Optional: `NETDUCTOR_OPERATOR_PUBKEY` / `_FILE` merge additional pubs.
