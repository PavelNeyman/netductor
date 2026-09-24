# Operator credentials file

After **primary** or **secondary** deploy (or manual collect), netductor writes:

`~/.netductor/credentials/<role>-<host>-<timestamp>.txt` (mode `0600`)

Symlink: `~/.netductor/credentials/latest-<role>.txt`

## Collect again

```bash
export NETDUCTOR_SSH_PORT=52222
netductor credentials collect --host 2.27.118.70 --role primary --key ~/.ssh/netductor_primary
netductor credentials collect --host 92.255.77.253 --role secondary --key ~/.ssh/netductor_primary
```

No file is written if SSH fails or required secrets are empty.

## Contents

| Field | Purpose |
|-------|---------|
| SSH command + key path | Access after harden |
| `BACKUP_KEY` | Decrypt backups / recover |
| `RECOVERY_TOKEN` | Download from secondary after `recovery arm` |

## Warnings

- Do **not** put `~/.netductor` in iCloud/Dropbox/Google Drive.
- Prefer a password manager; delete local copies if desired.
- Never commit these files.

## Recovery

```bash
ssh -p 52222 root@SECONDARY
netductor recovery arm --ttl 30m
# new primary:
netductor recover --from-secondary http://SECONDARY:8790 \
  --recovery-token "$RECOVERY_TOKEN" --key "$BACKUP_KEY"
netductor recovery disarm
```
