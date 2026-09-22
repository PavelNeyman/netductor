# Backup peer (legacy SSH)

**Deprecated.** Primary does **not** SCP backups to secondary.

Current model: [BACKUP.md](BACKUP.md)

- After `netductor backup`, online secondaries run agent cmd **`backup_pull`** (HTTPS mTLS to primary `:8789`).
- Files land in `/var/lib/netductor/backups/peers/core/` on secondary.
- Disaster recovery: secondary **:8790** recovery API + `netductor recover --from-secondary`.

`netductor backup peer-set` (SCP) remains in code for rare manual use but is **not** configured by deploy.

RU: [ru/BACKUP-PEER.md](ru/BACKUP-PEER.md)
