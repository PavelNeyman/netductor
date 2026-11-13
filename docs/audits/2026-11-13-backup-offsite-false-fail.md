# Audit note: backup offsite false fail (2026-11-13)

**Status:** open — investigation only (no code change in this note).

**Symptom:** `stack apply` console reports `acked=0 pending=1` for offsite `backup_pull` while Telegram shows successful `backup_pull:<cmd_id>` with `key=ok` and secondary disk has the matching `.ndenc` + sha.

**Version:** observed under stack apply to **v0.9.727**. Partial mitigations already in 0.9.727 did not eliminate the console/TG mismatch on production apply.

**Evidence (operator):**

- TG: `✅ Secondary cmd backup_pull:ccebd46b541f1497 … key=ok sha=b0dbaa… cmd_id=ccebd46b541f1497`
- Console: pre-backup and post-apply both printed `🔴 Backup offsite … acked=0 pending=1`
- Secondary path: `/var/lib/netductor/backups/peers/core/netductor-20261113-172934.tar.gz.ndenc`

**Next:** root-cause wait vs heartbeat timing during apply; see OPEN_ITEMS section *Backup offsite false fail during stack apply*.
