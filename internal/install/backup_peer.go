package install

import "fmt"

// SetBackupPeer is removed — offsite is agent backup_pull only (no primary→secondary SCP).
func SetBackupPeer(scpTarget string, scpOpts string) error {
	return fmt.Errorf("backup peer-set removed: offsite uses secondary agent backup_pull (HTTPS mTLS), not SCP")
}

// BackupPeerStatus reports that SCP peer is disabled.
func BackupPeerStatus() string {
	return "scp peer disabled; use agent backup_pull → /var/lib/netductor/backups/peers/core/ on secondary"
}
